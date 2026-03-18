# xiaohongshuritter OpenClaw Source Lite Package

This package is the recommended OpenClaw delivery wrapper for ARM64 and browser-bundled deployment.

Release version: `v1.0.0`

## Why use this package

- Builds from the current repository source
- Uses `Dockerfile.arm64`
- Chromium is installed during image build
- No runtime browser CDN download is required
- Better suited for Apple Silicon, ARM64 hosts, and OpenClaw handoff

## Included

- `docker-compose.yml`
- `.env.example`
- `config/accounts.json.example`
- `scripts/*.sh`
- `openclaw-mcp-example.json`
- `OPENCLAW_RUNBOOK.md`
- `VERSION`

## Install

```bash
cd package/openclaw-source-lite
chmod +x scripts/*.sh
./scripts/install.sh
```

## MCP endpoint

- `http://<SERVER_IP>:18060/mcp`

## Verify

```bash
./scripts/health.sh
./scripts/smoke.sh
curl http://127.0.0.1:18060/api/v1/accounts
```

If browser initialization fails, run:

```bash
./scripts/rebuild.sh
```
