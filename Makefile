GO ?= go
NPM ?= npm
WAILS ?= wails

.PHONY: test lint typecheck build desktop package-windows

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

package-windows:
	powershell -ExecutionPolicy Bypass -File scripts/package.ps1 -Wails $(WAILS)
