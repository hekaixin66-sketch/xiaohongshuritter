# xiaohongshuritter v1.0.1 Update Notes

Release date: `2026-04-04`

## Summary

`v1.0.1` upgrades the MCP from a "submit and hope" publisher to a loop-friendly publishing pipeline designed for OpenClaw and agent-driven automation.

## Highlights

- Async publish jobs
  - Supports non-blocking image/video publish submission
  - Returns `job_id` immediately
  - Supports polling through publish job status APIs and MCP tools

- Publish entity backfill
  - After publish success, the system attempts to resolve the final note entity
  - Backfills `note_id`, `note_url`, `feed_id`, and `xsec_token`
  - Uses current-account recent-note snapshots as a fallback strategy

- Publish verification
  - Supports verification by `job_id`, `note_id`, or `feed_id/xsec_token`
  - Returns structured verification fields such as:
    - `publish_visible`
    - `cover_visible`
    - `product_visible`
    - `verify_status`
    - `verify_reason`

- Structured product bind reporting
  - Publish results now include:
    - `product_bind_status`
    - `product_bind_count`
    - `products_requested`
    - `products_resolved`
    - `products_missing`

- OpenClaw operations improvement
  - Adds one-command update scripts for package deployment
  - Improves login QR reuse during polling
  - Uses bounded shutdown for async jobs so container updates do not hang indefinitely

## Recommended OpenClaw Flow

1. `check_login_status`
2. `submit_publish_content_async` or `submit_publish_video_async`
3. `get_publish_job_status`
4. `verify_published_note`
5. If needed, `list_recent_published_notes`

## Update Commands

If the deployment directory is `xiaohongshuritter`:

```bash
cd xiaohongshuritter && git fetch https://github.com/hekaixin66-sketch/xiaohongshuritter.git codex/publish-backfill-verify && (git switch codex/publish-backfill-verify || git switch -c codex/publish-backfill-verify FETCH_HEAD) && cd package/openclaw-source-lite && ./scripts/update.sh --no-pull
```

## Notes

- Browser integration tests are disabled by default and only run when explicitly enabled.
- Scheduled publishes do not immediately backfill a visible note entity, so they may return `backfill_status=skipped` until the actual publish time arrives.
