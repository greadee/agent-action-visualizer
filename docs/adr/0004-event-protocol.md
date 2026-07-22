# ADR 0004: Agent-independent event protocol

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Agent integrations expose different lifecycle and tool payloads while the session and renderer need stable semantics.

## Decision
Normalize all inputs into a versioned JSON Schema with matching Go and TypeScript types. Preserve source, confidence, and unknown metadata.

## Alternatives
Renderer-specific adapter payloads tightly couple integrations; raw terminal parsing cannot support trustworthy focus.

## Consequences
Adapters evolve independently and compatibility is contract-tested, with explicit schema migration cost.

## Evidence
The required adapters share session, tool, file, command, and access-interval concepts.
