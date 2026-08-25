# ADR 0003: React, TypeScript, and Three.js

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
The UI combines dense control surfaces, replay state, and a GPU-backed interactive 3D scene.

## Decision
Use React 19.2.8, TypeScript 6.0.3, Three.js 0.185.1, and React Three Fiber 9.6.1 with a small reducer/store boundary. Exact versions and lockfiles keep the verified peer set reproducible.

## Alternatives
Imperative Three.js alone complicates lifecycle composition; a large application state framework is unnecessary.

## Consequences
Scene logic remains declarative while hot geometry updates use direct buffers and refs.

## Evidence
React Three Fiber 9 supports React 19 and exposes current Three.js objects without waiting for wrapper releases.
