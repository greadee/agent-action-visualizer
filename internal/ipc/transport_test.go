package ipc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"sync/atomic"
	"testing"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestLocalRoundTripAndDisconnect(t *testing.T) {
	endpoint := testEndpoint(t.Name())
	received := make(chan protocol.Event, 1)
	server := NewServer(endpoint, func(event protocol.Event) bool {
		received <- event
		return true
	})
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	event := validEvent()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := NewClient(endpoint).Send(ctx, []protocol.Event{event}); err != nil {
		t.Fatal(err)
	}
	select {
	case actual := <-received:
		if actual.EventID != event.EventID {
			t.Fatalf("event = %#v", actual)
		}
	case <-time.After(time.Second):
		t.Fatal("collector did not receive event")
	}

	missingCtx, missingCancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer missingCancel()
	if err := NewClient(testEndpoint(t.Name()+"-missing")).Send(missingCtx, []protocol.Event{event}); err == nil {
		t.Fatal("expected disconnected collector error")
	}
}

func TestClientReconnectsAfterCollectorRestart(t *testing.T) {
	endpoint := testEndpoint(t.Name())
	client := NewClient(endpoint)
	event := validEvent()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	if err := client.Send(ctx, []protocol.Event{event}); err == nil {
		cancel()
		t.Fatal("disconnected send unexpectedly succeeded")
	}
	cancel()
	received := make(chan string, 1)
	server := NewServer(endpoint, func(event protocol.Event) bool {
		received <- event.EventID
		return true
	})
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Send(ctx, []protocol.Event{event}); err != nil {
		t.Fatalf("send after restart failed: %v", err)
	}
	select {
	case id := <-received:
		if id != event.EventID {
			t.Fatalf("event id = %q", id)
		}
	case <-time.After(time.Second):
		t.Fatal("reconnected event was not received")
	}
}

func TestServerRejectsOversizedFrame(t *testing.T) {
	endpoint := testEndpoint(t.Name())
	server := NewServer(endpoint, func(protocol.Event) bool { t.Fatal("oversized frame was submitted"); return false })
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	connection, err := dialLocal(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, MaxPayloadBytes+1)
	if _, err := connection.Write(header); err != nil {
		t.Fatal(err)
	}
	_ = connection.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buffer := make([]byte, 1)
	if _, err := connection.Read(buffer); err == nil {
		t.Fatal("oversized frame unexpectedly acknowledged")
	}
}

func TestServerRejectsMalformedFrame(t *testing.T) {
	endpoint := testEndpoint(t.Name())
	called := make(chan struct{}, 1)
	server := NewServer(endpoint, func(protocol.Event) bool { called <- struct{}{}; return false })
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	connection, err := dialLocal(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	payload := []byte(`{"version":"1","events":[`)
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))
	if err := writeAll(connection, header); err != nil {
		t.Fatal(err)
	}
	if err := writeAll(connection, payload); err != nil {
		t.Fatal(err)
	}
	_ = connection.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buffer := make([]byte, 1)
	if _, err := io.ReadFull(connection, buffer); err == nil {
		t.Fatal("malformed frame unexpectedly acknowledged")
	}
	select {
	case <-called:
		t.Fatal("malformed frame reached submitter")
	default:
	}
}

func TestClientRejectsInvalidEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := NewClient(testEndpoint(t.Name())).Send(ctx, []protocol.Event{{}}); err == nil {
		t.Fatal("expected invalid event error")
	}
}

func TestServerRejectsInvalidBatchAtomically(t *testing.T) {
	endpoint := testEndpoint(t.Name())
	var submitted atomic.Int32
	server := NewServer(endpoint, func(protocol.Event) bool { submitted.Add(1); return true })
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	valid := validEvent()
	invalid := validEvent()
	invalid.EventID = ""
	payload, err := json.Marshal(batch{Version: "1", Events: []protocol.Event{valid, invalid}})
	if err != nil {
		t.Fatal(err)
	}
	if acknowledged := sendRawFrame(t, endpoint, payload, uint32(len(payload))); acknowledged {
		t.Fatal("invalid batch was acknowledged")
	}
	if submitted.Load() != 0 {
		t.Fatalf("invalid batch was partially submitted: %d", submitted.Load())
	}
}

func TestServerRejectsZeroTruncatedAndWrongVersionFrames(t *testing.T) {
	for name, test := range map[string]struct {
		payload  []byte
		declared uint32
	}{
		"zero":          {declared: 0},
		"truncated":     {payload: []byte(`{"version":"1"}`), declared: 100},
		"wrong version": {payload: []byte(`{"version":"2","events":[]}`), declared: uint32(len(`{"version":"2","events":[]}`))},
	} {
		t.Run(name, func(t *testing.T) {
			endpoint := testEndpoint(t.Name())
			server := NewServer(endpoint, func(protocol.Event) bool { t.Fatal("invalid frame was submitted"); return false })
			if err := server.Start(); err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			if acknowledged := sendRawFrame(t, endpoint, test.payload, test.declared); acknowledged {
				t.Fatal("invalid frame was acknowledged")
			}
		})
	}
}

func sendRawFrame(t *testing.T, endpoint string, payload []byte, declared uint32) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	connection, err := dialLocal(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, declared)
	if err := writeAll(connection, header); err != nil {
		t.Fatal(err)
	}
	if len(payload) > 0 {
		if err := writeAll(connection, payload); err != nil {
			t.Fatal(err)
		}
	}
	_ = connection.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
	response := make([]byte, 1)
	_, err = io.ReadFull(connection, response)
	return err == nil && response[0] == ack[0]
}

func BenchmarkLocalRoundTrip(b *testing.B) {
	endpoint := testEndpoint(b.Name())
	server := NewServer(endpoint, func(protocol.Event) bool { return true })
	if err := server.Start(); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(server.Close)
	client := NewClient(endpoint)
	events := []protocol.Event{validEvent()}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := client.Send(ctx, events)
		cancel()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func validEvent() protocol.Event {
	return protocol.Event{
		SchemaVersion: protocol.SchemaVersion, EventID: "event", EventType: protocol.EventSessionStarted,
		SourceType: protocol.SourceNativeHook, SourceConfidence: protocol.ConfidenceExact, Timestamp: time.Unix(1, 0).UTC(),
	}
}
