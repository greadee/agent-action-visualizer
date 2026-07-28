package ipc

import (
	"context"
	"encoding/binary"
	"io"
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
