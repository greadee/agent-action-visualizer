# Shared UI integration

This application consumes version `0.1.0` of the canonical `@prool-ui/*` packages. Until a remote package registry exists, npm `file:` dependencies point at the sibling `shared-ui` development repository. The shared repository must be built before this frontend.

## Ownership

The shared repository owns tokens, themes, reusable React rendering, accessibility contracts, tests, and documentation. This application owns Wails integration, graph/session models, Three.js rendering, activity semantics, and app-specific composition.

`GraphLegend` consumes the shared `VisuallyHidden` accessibility primitive while keeping graph kinds, marker colors, and activity language local. The activity-mode adapter supplies application-owned `ActivityMode` options and state to the canonical shared `SegmentedControl`.

## Upgrade

1. Build and validate the target shared UI version.
2. Update the three exact dependency versions or temporary `file:` paths together.
3. Run `npm test`, `npm run typecheck`, `npm run build`, and the app's live browser check.
4. Review the dark-theme visual diff.
5. Commit the dependency update separately.

## Rollback

Restore the previous validated package versions and lockfile, rerun the same checks, and revert the isolated migration commit if necessary. Never edit installed package files.

## Promotion

Generalizable changes begin as an application adapter or proposal. Submit provenance, API impact, token impact, consumer impact, tests, and release type to the shared repository. Remove any temporary local override only after the upstream release is consumed.
