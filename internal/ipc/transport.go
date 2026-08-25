// Package ipc implements the bounded local adapter-to-collector transport.
package ipc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

const (
	MaxPayloadBytes = 256 << 10
	MaxBatchEvents  = 128
	maxConnections  = 32
)

var ack = []byte{0x06}

type batch struct {
	Version string           `json:"version"`
	Events  []protocol.Event `json:"events"`
}

type Client struct{ Endpoint string }

func NewClient(endpoint string) *Client { return &Client{Endpoint: endpoint} }

func DefaultEndpoint() string {
	if value := strings.TrimSpace(os.Getenv("AAV_COLLECTOR_ENDPOINT")); value != "" {
		return value
	}
	return platformDefaultEndpoint()
}

func (c *Client) Send(ctx context.Context, events []protocol.Event) error {
	if len(events) == 0 || len(events) > MaxBatchEvents {
		return errors.New("invalid event batch size")
	}
	for _, event := range events {
		if err := event.Validate(); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(batch{Version: "1", Events: events})
	if err != nil {
		return err
	}
	if len(payload) > MaxPayloadBytes {
		return errors.New("event batch exceeds payload limit")
	}
	connection, err := dialLocal(ctx, c.Endpoint)
	if err != nil {
		return err
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))
	if err := writeAll(connection, header); err != nil {
		return err
	}
	if err := writeAll(connection, payload); err != nil {
		return err
	}
	response := make([]byte, 1)
	if _, err := io.ReadFull(connection, response); err != nil {
		return err
	}
	if response[0] != ack[0] {
		return errors.New("collector rejected event batch")
	}
	return nil
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}

type Server struct {
	endpoint string
	submit   func(protocol.Event) bool
	listener net.Listener
	stop     chan struct{}
	done     chan struct{}
	workers  sync.WaitGroup
	once     sync.Once
	slots    chan struct{}
}

func NewServer(endpoint string, submit func(protocol.Event) bool) *Server {
	return &Server{endpoint: endpoint, submit: submit, stop: make(chan struct{}), done: make(chan struct{}), slots: make(chan struct{}, maxConnections)}
}

func (s *Server) Start() error {
	listener, err := listenLocal(s.endpoint)
	if err != nil {
		return err
	}
	s.listener = listener
	go s.serve()
	return nil
}

func (s *Server) Close() {
	s.once.Do(func() {
		close(s.stop)
		if s.listener != nil {
			_ = s.listener.Close()
		}
		<-s.done
	})
}

func (s *Server) serve() {
	defer close(s.done)
	defer cleanupLocal(s.endpoint)
	for {
		connection, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stop:
				s.workers.Wait()
				return
			default:
				continue
			}
		}
		select {
		case s.slots <- struct{}{}:
			s.workers.Add(1)
			go s.handle(connection)
		default:
			_ = connection.Close()
		}
	}
}

func (s *Server) handle(connection net.Conn) {
	defer s.workers.Done()
	defer func() { <-s.slots }()
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(250 * time.Millisecond))
	header := make([]byte, 4)
	if _, err := io.ReadFull(connection, header); err != nil {
		return
	}
	size := binary.BigEndian.Uint32(header)
	if size == 0 || size > MaxPayloadBytes {
		return
	}
	payload := make([]byte, int(size))
	if _, err := io.ReadFull(connection, payload); err != nil {
		return
	}
	var incoming batch
	if json.Unmarshal(payload, &incoming) != nil || incoming.Version != "1" || len(incoming.Events) == 0 || len(incoming.Events) > MaxBatchEvents {
		return
	}
	for _, event := range incoming.Events {
		if event.Validate() != nil {
			return
		}
	}
	for _, event := range incoming.Events {
		if s.submit != nil {
			_ = s.submit(event)
		}
	}
	_, _ = connection.Write(ack)
}
