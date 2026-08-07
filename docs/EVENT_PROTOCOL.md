# Event protocol

Version 1.0 is defined by `protocol/schema/v1/event.schema.json`. Adapters submit events as UTF-8 JSON. Unknown fields are accepted so compatible v1 readers survive additive changes; incompatible semantics require a new schema version.

Only the universal evidence envelope is required: schema version, event ID, event type, source type, source confidence, and RFC 3339 timestamp. Session, project, path, operation, duration, and delta fields are present when meaningful rather than filled with invented values.

Confidence is semantic: `exact` is explicitly named by a native integration; `correlated` joins matching pre/post evidence; `observed` is a direct filesystem/Git fact; `inferred` is a heuristic. Consumers must not promote confidence.

Raw commands are discarded before persistence; command and tool labels are single-line, bounded, and redact common credential forms. Paths, including `secondary_paths`, are normalized to the selected project root and rejected if an existing symbolic link escapes it. Numeric durations, deltas, byte sizes, process IDs, and monotonic timestamps are non-negative. Payload, string, metadata-size, metadata-depth, and metadata-entry limits are enforced at ingress in addition to schema validation.

Normalized metadata is intentionally narrow. Consumers retain only deterministic reducer keys such as `access_sequence`, `secondary_paths`, hook/evidence provenance, work source/confidence, bounded drop/coalescing counts, and wrapper exit/signal state. Unknown protocol fields remain wire-compatible, but unknown metadata is not retained by the application.
