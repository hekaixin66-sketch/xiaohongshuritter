# xiaohongshuritter

`xiaohongshuritter` is an enterprise-grade Xiaohongshu MCP system with multi-tenant, multi-account, and concurrency control support. It provides `Docker deployment`, `source deployment`, and `OpenClaw deployment` delivery modes.

This project is intended for:
- enterprises managing multiple brands or business units with one MCP service
- teams operating multiple Xiaohongshu accounts with isolated cookies and runtime settings
- OpenClaw or MCP-based clients that need stable production deployment
- Windows, macOS, Linux, Docker, and Apple Silicon environments

## Highlights

- Multi-tenant routing via `tenant_id`
- Multi-account routing via `account_id`
- Global and per-account concurrency control
- MCP and HTTP API entrypoints
- Docker and source deployment options
- OpenClaw-ready delivery workflow
- ARM64 container support with bundled Chromium

## Main Features

- Login status check
- Login QR code generation
- Image post publishing
- Video post publishing
- Search feeds
- List recommended feeds
- Get feed detail
- Post comments and replies
- List configured accounts and current in-flight usage

## Documentation

- Enterprise overview: [docs/enterprise_deployment.md](./docs/enterprise_deployment.md)
- Docker deployment: [docs/docker_deployment.md](./docs/docker_deployment.md)
- Source deployment: [docs/source_deployment.md](./docs/source_deployment.md)
- OpenClaw deployment: [docs/openclaw_deployment.md](./docs/openclaw_deployment.md)
- macOS M4 guide: [docs/macos_m4_openclaw.md](./docs/macos_m4_openclaw.md)
- Windows enterprise guide: [docs/windows_enterprise.md](./docs/windows_enterprise.md)
- Enterprise API additions: [docs/API_ENTERPRISE.md](./docs/API_ENTERPRISE.md)
- Base API reference: [docs/API.md](./docs/API.md)

## Quick Start

### 1. Prepare account config

```bash
cp configs/accounts.enterprise.example.json configs/accounts.json
```

### 2. Start the service

Docker:

```bash
docker compose -f docker/docker-compose.yml up -d
```

Source:

```bash
go run .
```

### 3. Verify service

```bash
curl http://127.0.0.1:18060/health
curl http://127.0.0.1:18060/api/v1/accounts
```

### 4. MCP endpoint

```text
http://127.0.0.1:18060/mcp
```

## OpenClaw Integration

In OpenClaw, add a new MCP server with:

- Transport: `streamable_http`
- URL: `http://<server-ip>:18060/mcp`

For enterprise usage, always pass:

- `tenant_id`
- `account_id`

Example:

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "keyword": "brand content"
}
```

## Package Templates

This repository includes ready-to-adapt delivery templates:

- [package/README.md](./package/README.md)
- [package/openclaw-lite/README.md](./package/openclaw-lite/README.md)
- [package/openclaw-source-lite/README.md](./package/openclaw-source-lite/README.md)

## Upstream Reference

This repository references the open-source project [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp) for design and implementation ideas, and extends it with enterprise multi-account routing, concurrency controls, OpenClaw delivery, Docker workflows, and multi-platform deployment support.

See [NOTICE.md](./NOTICE.md) for attribution details.

## Open Source Collaboration

- Change log: [CHANGELOG.md](./CHANGELOG.md)
- Security policy: [SECURITY.md](./SECURITY.md)
- Code of conduct: [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)
- Contributing guide: [CONTRIBUTING.md](./CONTRIBUTING.md)

## Compliance

Use this project only with proper account authorization and in compliance with platform rules, local laws, and your internal review processes.
