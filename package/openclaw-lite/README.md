# xiaohongshuritter OpenClaw Lite Package

This package is a lightweight deployment wrapper for standard Docker environments.

## What it does

- Builds from the current repository source
- Uses the top-level `Dockerfile`
- Suitable for Linux amd64 and general server deployment
- Provides a minimal OpenClaw-ready MCP endpoint

## Included

- `docker-compose.yml`
- `.env.example`
- `config/accounts.json.example`
- `scripts/*.sh`
- `openclaw-mcp-example.json`

## Install

```bash
cd package/openclaw-lite
chmod +x scripts/*.sh
./scripts/install.sh
```

## Update

Normal update:

```bash
./scripts/update.sh
```

Force a full rebuild:

```bash
./scripts/update.sh --full
```

## MCP endpoint

- `http://<SERVER_IP>:18060/mcp`

## Useful commands

```bash
./scripts/start.sh
./scripts/stop.sh
./scripts/logs.sh
./scripts/health.sh
```
