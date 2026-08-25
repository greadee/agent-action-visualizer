# ADR 0005: Local IPC

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Hook submission must be local, bounded, authenticated where applicable, and fast across platforms.

## Decision
Abstract transport behind a local client/server interface: owner-only Unix sockets on Unix, current-user/SYSTEM named pipes on Windows, and token-protected loopback HTTP/WebSocket only for development and bridge use. Native adapter IPC relies on operating-system account access control rather than an application token, so other processes running as the same user remain in the trust boundary.

## Alternatives
Public TCP increases attack surface; direct database writes couple adapters to storage and block agent operations.

## Consequences
Platform implementations differ, but adapters share one wire format and deadline behavior.

## Evidence
Unix socket mode tests, Windows DACL construction, bounded framing tests, and the P9-S3 threat review document the OS-scoped boundary and its same-user residual risk.
