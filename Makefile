GO ?= go
NPM ?= npm
WAILS ?= wails

.PHONY: test lint typecheck build desktop

test:
	$(GO) test ./...
	cd apps/desktop && $(GO) test .
	cd apps/desktop/frontend && $(NPM) test

lint:
	$(GO) vet ./...
	cd apps/desktop && $(GO) vet .
	cd apps/desktop/frontend && $(NPM) run lint

typecheck:
	cd apps/desktop/frontend && $(NPM) run typecheck

build:
	cd apps/desktop/frontend && $(NPM) run build

desktop:
	cd apps/desktop && $(WAILS) build
