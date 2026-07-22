# ADR 0002: Wails desktop shell

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
The product needs a native desktop lifecycle with a modern web renderer but should avoid bundling a full browser runtime.

## Decision
Use stable Wails v2 as the desktop shell and Go/TypeScript bridge.

## Alternatives
Electron has a mature ecosystem but a larger runtime. Tauri is capable but would move the core host to Rust.

## Consequences
Wails keeps the Go core direct and packages against platform webviews; platform SDK prerequisites remain.

## Evidence
The Wails project identifies v2 as stable and v3 as alpha; version pinning occurs in P0-S3.
