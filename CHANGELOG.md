# CHANGELOG

All notable changes to this project will be documented in this file.

## v1.0.1 - 2026-04-04

This release focuses on making OpenClaw-driven publishing form a stable, observable loop from submission to verification.

### Added

- async publish job flow for image and video publishing with `job_id` polling
- publish backfill that resolves `note_id`, `note_url`, `feed_id`, and `xsec_token` after publish success
- publish verification flow for checking note visibility, cover visibility, and product bind outcome
- recent published notes query for account-scoped backfill and verification fallback
- one-command OpenClaw update scripts for `openclaw-lite` and `openclaw-source-lite`

### Changed

- publish responses are now structured and include backfill, cleanup, and product binding fields
- product binding returns structured requested/resolved/missing information instead of relying only on logs
- HTTP and MCP publish APIs support async mode and job status querying
- login QR code flow now reuses pending watchers to reduce duplicate browser sessions during polling
- job manager shutdown now uses bounded wait semantics so container restarts are not blocked by long-running jobs
- cookie persistence now writes through a temp file before replacing the destination file

### Fixed

- browser integration tests are skipped by default unless explicitly enabled
- OpenClaw publish workflows can now verify the final note instead of stopping at "submitted successfully"

## v1.0.0 - 2026-03-18

Initial open-source release of `xiaohongshuritter`.

### Added

- enterprise multi-tenant and multi-account MCP routing
- concurrency controls for shared account usage
- Docker, source, and OpenClaw deployment documentation
- ARM64 deployment workflow with bundled Chromium strategy
- OpenClaw delivery templates under `package/`

### Changed

- project branding renamed to `xiaohongshuritter`
- publishing visibility now normalizes common aliases to Xiaohongshu-supported Chinese values
- repository documentation reorganized for public GitHub release

### Removed

- legacy donation records and unrelated historical publishing examples
- inherited repository metadata not aligned with this public release
