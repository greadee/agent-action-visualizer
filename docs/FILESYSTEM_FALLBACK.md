# Filesystem fallback

P7-S3 adds a deterministic, local fallback for generic commands that do not
provide native lifecycle events or a supported structured stream. The observer
is metadata-only and runs outside the wrapped process's I/O and exit path.

## Observation flow

1. Resolve the project root and load the scanner's default exclusions,
   `.gitignore`, and `.aavignore`.
2. Walk non-ignored directories once, recording only path, kind, size, and
   modification time, then install an OS filesystem watch on each directory.
3. Drain OS notifications into a 1,024-item non-blocking queue.
4. Debounce and coalesce changes per path, with at most 512 pending paths and
   128 paths per batch.
5. Run one bounded `git status` and one bounded `git diff --numstat` inspection
   for the batch.
6. Emit normalized protocol events through the wrapper evidence gate.

New directories are added recursively. Symlinks are not followed. Queue or
pending-map saturation drops visualization evidence only and emits a bounded
`event_dropped_or_coalesced` record when capacity becomes available.

## Evidence semantics

Direct create, write, and delete notifications are `filesystem/observed`.
Repeated notifications for the same path become one event with a
`coalesced_events` count.

Rename and move events require either Git rename evidence or one unique
old/new metadata match. A same-directory file change becomes `file_renamed`; a
cross-directory change becomes `file_moved`. Ambiguous candidates are never
guessed and remain inspectable delete/create events. Directory renames require
Git evidence.

Git line counts are `git_numstat/correlated` metadata attached only when Git
returns a supported tracked-file delta. Binary results set `is_binary`; absent
or failed Git evidence leaves line values unknown.

The wrapper delays fallback evidence for 100 milliseconds. Matching native or
structured evidence wins and suppresses the duplicate. The gate and recent-key
set are bounded; unmatched fallback evidence flushes deterministically by key.

## Failure-open boundary

Watcher startup failure, inaccessible directories, Git timeout/failure,
malformed Git output, queue overload, collector disconnect, and observer panic
cannot change command arguments, environment, stdin, stdout, stderr, working
files, signals, or exit status. Observer startup, delivery, Git inspection, and
shutdown all remain asynchronous or deadline-bounded.

The fallback attributes changes to the wrapper session window, not to a
specific tool invocation. OS notifications may be lost during extreme kernel
overflow or on unsupported/network filesystems. The current shared ignore
matcher intentionally supports the scanner's established subset and does not
implement negation rules.
