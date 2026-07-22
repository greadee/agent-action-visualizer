package main

import (
	"context"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Health reports the embedded collector shell state without network access.
func (a *App) Health() map[string]string {
	return map[string]string{
		"service": "agent-action-visualizer",
		"status":  "ready",
	}
}
