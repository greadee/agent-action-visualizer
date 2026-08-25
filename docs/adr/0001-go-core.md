# ADR 0001: Go core

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
The collector needs cross-platform concurrency, filesystem/process integration, small native binaries, and predictable latency.

## Decision
Implement protocol handling, ingestion, sessions, scanning, graph state, adapters, IPC, and persistence in Go 1.26.

## Alternatives
Rust improves memory control but raises contribution and Wails integration cost. Node.js shares frontend tooling but increases runtime and packaging surface.

## Consequences
The application gains simple concurrency and deployment at the cost of a second language boundary.

## Evidence
Go ships first-class cross-platform tooling and Wails v2 uses Go as its native host language.
