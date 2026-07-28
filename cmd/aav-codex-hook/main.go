package main

import (
	"context"
	"os"
	"time"

	codexadapter "github.com/greadee/agent-action-visualizer/internal/adapters/codex"
	"github.com/greadee/agent-action-visualizer/internal/ipc"
)

func main() {
	defer func() { _ = recover() }()
	codexadapter.Run(
		context.Background(),
		os.Stdin,
		ipc.NewClient(ipc.DefaultEndpoint()),
		time.Now,
	)
}
