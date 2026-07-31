# Adapter SDK

The public Go SDK is `github.com/greadee/agent-action-visualizer/adapter/go`.
It is a local-only contract for translating supported lifecycle evidence into
the versioned event protocol. It never calls a model, changes prompts, or sends
events outside the local collector selected by the host application.

## Contract

An adapter implements `adapter.Adapter` by returning a stable `Descriptor` and
by implementing `Observe(context.Context, adapter.Emitter)`. Its descriptor
must declare the protocol version and only the evidence capabilities it can
actually produce. `adapter.ValidateAdapter` is the static contract check used
by integration tests.

`Observe` is silent and failure-open. It must not write to an observed agent's
stdout or stderr, change its files, change its exit status, or wait on
rendering, persistence, Git, network, or collector availability. Use
`adapter.NewBoundedEmitter` at the process boundary. It validates events,
limits each batch and in-flight collector calls, applies a short deadline, and
drops data when the collector is unavailable or saturated.

```go
type Adapter struct{ events []protocol.Event }

func (Adapter) Descriptor() adapter.Descriptor { /* stable metadata */ }

func (a Adapter) Observe(ctx context.Context, out adapter.Emitter) {
	// Build metadata-only, validated protocol events from supported local input.
	out.Emit(ctx, a.events)
}
```

The runnable shape is demonstrated by `adapter/go/example.Static` and tested
with `adapter.MockCollector`.

## Event hygiene

Emit only metadata required by `protocol.Event`. Do not submit source content,
prompts, model responses, transcript contents, environment variables,
credentials, or raw tool output. Paths must be relative to the selected project
root; use `adapter.NormalizeProjectPath` and drop values that escape the root.
Never create a replacement path or infer a missing value.

Set `AdapterID` and `AdapterVersion` on every emitted event. Use a stable,
adapter-owned event ID and an RFC 3339 timestamp. Validate the event before
handing it to the emitter; the bounded emitter drops invalid envelopes as a
second safeguard.

## Evidence mapping

Choose source and confidence from evidence, not convenience:

| Evidence | Source type | Confidence |
| --- | --- | --- |
| Explicit local lifecycle hook | `native_hook` | `exact` |
| Explicit supported structured agent record | `structured_stream` | `exact` |
| Direct editor observation | `editor` | `observed` |
| Filesystem watcher event | `filesystem` | `observed` |
| Git-derived fact | `git` | `observed` |
| Matching pre/post records | matching source | `correlated` |
| Heuristic fallback | applicable source | `inferred` |

Do not promote confidence downstream. If a path, duration, line delta, rename,
or encoding state is unknown, omit that field and emit only the supported fact.
Binary and unsupported encodings remain explicit unknown work states rather
than zero-valued deltas.

## Author checklist

- Declare only supported capabilities and validate the descriptor in tests.
- Bound input bytes, queue size, concurrency, and local collector deadlines.
- Preserve observed process arguments, environment, stdout, stderr, files, and exit behavior.
- Keep ordinary failures silent; collect diagnostics only through safe local channels.
- Test malformed input, unavailable collectors, timeout, saturation, and path escape rejection.
- Add sanitized fixtures that contain no source content or secrets.
