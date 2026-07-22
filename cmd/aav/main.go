package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "mock-event" {
		fail("usage: aav mock-event [event.json|-]")
	}
	reader := io.Reader(os.Stdin)
	if len(os.Args) > 2 && os.Args[2] != "-" {
		file, err := os.Open(os.Args[2])
		if err != nil {
			fail(err.Error())
		}
		defer file.Close()
		reader = file
	}
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	var event protocol.Event
	if err := decoder.Decode(&event); err != nil {
		fail("decode event: " + err.Error())
	}
	if err := event.Validate(); err != nil {
		fail("validate event: " + err.Error())
	}
	payload, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	fmt.Println(string(payload))
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(2) }
