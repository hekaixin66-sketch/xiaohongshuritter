# Docker Deployment

This deployment uses the current repository source to build a local image for `xiaohongshuritter`.

## Quick Start

1. Prepare account config:

```bash
mkdir -p config
cp ./config/accounts.example.json ./config/accounts.json
```

2. Start service:

```bash
docker compose up -d
```

3. Check logs:

```bash
docker compose logs -f
```

## Mounted Paths

- `./data` -> `/app/data` (cookies and runtime data)
- `./images` -> `/app/images` (local images/videos for publishing)
- `./config/accounts.json` -> `/app/config/accounts.json` (multi-tenant config)

## Environment Variables

- `XHS_ACCOUNT_CONFIG=/app/config/accounts.json`
- `XHS_MAX_CONCURRENCY=12`
- `XHS_ACCOUNT_MAX_CONCURRENCY=3`
- `XHS_ACQUIRE_TIMEOUT=180s`
- `ROD_BROWSER_BIN=/usr/bin/google-chrome`

## Image and Container

- Image: `xiaohongshuritter:local`
- Container: `xiaohongshuritter`

## Verify

- HTTP Health: `GET http://127.0.0.1:18060/health`
- Account List: `GET http://127.0.0.1:18060/api/v1/accounts`
- MCP Endpoint: `http://127.0.0.1:18060/mcp`
