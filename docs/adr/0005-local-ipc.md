# ADR 0005: Local IPC

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Hook submission must be local, bounded, authenticated where applicable, and fast across platforms.

## Decision
Abstract transport behind a local client/server interface: Unix sockets on Unix, named pipes on Windows, and token-protected loopback HTTP/WebSocket only for development and bridge use.

## Alternatives
Public TCP increases attack surface; direct database writes couple adapters to storage and block agent operations.

## Consequences
Platform implementations differ, but adapters share one wire format and deadline behavior.

## Evidence
Local sockets/pipes provide OS-scoped access and avoid network dependencies.
