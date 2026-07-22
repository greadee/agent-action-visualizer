# Event protocol

Version 1.0 is defined by `protocol/schema/v1/event.schema.json`. Adapters submit events as UTF-8 JSON. Unknown fields are accepted so compatible v1 readers survive additive changes; incompatible semantics require a new schema version.

Only the universal evidence envelope is required: schema version, event ID, event type, source type, source confidence, and RFC 3339 timestamp. Session, project, path, operation, duration, and delta fields are present when meaningful rather than filled with invented values.

Confidence is semantic: `exact` is explicitly named by a native integration; `correlated` joins matching pre/post evidence; `observed` is a direct filesystem/Git fact; `inferred` is a heuristic. Consumers must not promote confidence.

Commands are redacted before persistence. Paths are normalized to the selected project root. Numeric durations, deltas, and byte sizes are non-negative. Payload and string limits are enforced at ingress in addition to schema validation.
