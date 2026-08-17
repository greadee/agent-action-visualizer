# Toolchain

Pinned on 2026-07-21 after checking official release and package metadata.

| Component | Version | Compatibility evidence |
| --- | --- | --- |
| Go | 1.26.6 | Current Go release; Windows archive checksum verified before local use |
| Wails | 2.12.0 | Current stable v2; v3 remains alpha |
| Node.js | 22.22.3 | Satisfies Vite 8's `>=22.12` engine range |
| React / React DOM | 19.2.8 | Satisfies React Three Fiber's `>=19 <19.3` peer range |
| Three.js | 0.185.1 | Satisfies React Three Fiber's `>=0.156` peer range |
| React Three Fiber | 9.6.1 | Current stable React 19 renderer |
| Drei | 10.7.7 | Requires React 19 and React Three Fiber 9 |
| Vite / React plugin | 8.1.5 / 6.0.3 | Plugin requires Vite 8 |
| TypeScript | 6.0.3 | Newest stable line within typescript-eslint's `<6.1` peer range |
| Vitest | 4.1.10 | Supports Node 22 |

Dependencies use exact versions and lockfiles. Wails remains pinned through the desktop `go.mod`; CI installs the declared Go and Node versions.
