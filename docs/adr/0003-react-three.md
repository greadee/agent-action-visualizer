# ADR 0003: React, TypeScript, and Three.js

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
The UI combines dense control surfaces, replay state, and a GPU-backed interactive 3D scene.

## Decision
Use React 19, TypeScript, Three.js, and React Three Fiber 9 with a small reducer/store boundary.

## Alternatives
Imperative Three.js alone complicates lifecycle composition; a large application state framework is unnecessary.

## Consequences
Scene logic remains declarative while hot geometry updates use direct buffers and refs.

## Evidence
React Three Fiber 9 supports React 19 and exposes current Three.js objects without waiting for wrapper releases.
