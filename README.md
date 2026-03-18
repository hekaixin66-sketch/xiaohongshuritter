# xiaohongshuritter

`xiaohongshuritter` 是一个面向企业场景的小红书 MCP 系统，支持多租户、多账号、并发控制，并提供 `Docker 部署`、`源码部署`、`OpenClaw 部署` 三种交付方式。

项目适合以下场景：

- 企业内多个品牌、多个业务线共用一套小红书自动化能力
- 同一服务同时托管多个账号，并为每个账号单独隔离 Cookie、代理和浏览器参数
- 需要通过 MCP 被 OpenClaw、Claude、Cherry Studio、AnythingLLM 等客户端调用
- 需要在 `Windows`、`macOS`、`Linux`、`OpenClaw` 等环境中稳定部署

## 项目定位

与“单账号、单进程、手工操作型工具”不同，`xiaohongshuritter` 更强调企业级运行能力：

- 多租户：通过 `tenant_id` 隔离企业或业务线
- 多账号：通过 `account_id` 管理同租户下的多个账号
- 多并发：提供全局并发和单账号并发控制，避免单账号被高并发打爆
- 多平台：支持 Docker、Windows 源码部署、macOS M4、OpenClaw
- 多入口：同时提供 MCP 与 HTTP API

## 核心能力

- 登录状态检测与二维码登录
- 图文笔记发布
- 视频笔记发布
- 首页推荐流获取
- 关键词搜索
- 笔记详情获取
- 评论发布与回复
- 账号列表与运行时并发状态查看
- 企业级账号路由：`tenant_id` / `account_id`

## 架构特点

- 每个账号拥有独立 `cookie_path`
- 每个账号可独立配置 `max_concurrency`
- 支持默认租户和默认账号回退
- HTTP 与 MCP 共用统一账号解析逻辑
- 支持通过 Header、Body、Query 三种方式路由账号
- ARM64 Docker 镜像可内置 Chromium，适合 OpenClaw 场景

## 部署方式

建议按你的使用方式选择：

| 场景 | 推荐方式 | 文档 |
| --- | --- | --- |
| 本地或服务器快速上线 | Docker 部署 | [docs/docker_deployment.md](./docs/docker_deployment.md) |
| 需要直接调试源码、二次开发 | 源码部署 | [docs/source_deployment.md](./docs/source_deployment.md) |
| 给 OpenClaw 稳定调用 | OpenClaw 部署 | [docs/openclaw_deployment.md](./docs/openclaw_deployment.md) |
| Apple Silicon / M4 | macOS M4 + OpenClaw | [docs/macos_m4_openclaw.md](./docs/macos_m4_openclaw.md) |
| Windows 本地部署 | Windows 企业版部署 | [docs/windows_enterprise.md](./docs/windows_enterprise.md) |

## 快速开始

### 1. 准备账号配置

企业版账号配置示例：

```bash
cp configs/accounts.enterprise.example.json configs/accounts.json
```

最小配置示例：

```json
{
  "default_tenant": "default",
  "default_account": "default",
  "global_max_concurrency": 12,
  "tenants": [
    {
      "id": "default",
      "name": "Default Tenant",
      "default_account": "main",
      "accounts": [
        {
          "id": "main",
          "name": "Main Account",
          "cookie_path": "./data/default/main/cookies.json",
          "max_concurrency": 3
        }
      ]
    }
  ]
}
```

### 2. 选择部署方式

Docker：

```bash
docker compose -f docker/docker-compose.yml up -d
```

源码：

```bash
go run .
```

### 3. 检查服务

```bash
curl http://127.0.0.1:18060/health
curl http://127.0.0.1:18060/api/v1/accounts
```

### 4. MCP 地址

默认 MCP 地址：

```text
http://127.0.0.1:18060/mcp
```

## OpenClaw 调用方式

在 OpenClaw 中新增 MCP Server：

- Transport: `streamable_http`
- URL: `http://<服务器IP>:18060/mcp`

企业级调用时，强烈建议每次都显式传入：

- `tenant_id`
- `account_id`

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "title": "企业内容发布示例",
  "content": "支持多租户、多账号与并发发布",
  "images": ["/app/images/demo.jpg"],
  "tags": ["内容运营", "企业发布"],
  "visibility": "公开可见"
}
```

更多 OpenClaw 说明见 [docs/openclaw_deployment.md](./docs/openclaw_deployment.md)。

## HTTP API 账号路由

账号路由优先级如下：

1. Body 中的 `tenant_id`、`account_id`
2. Header 中的 `X-XHS-Tenant`、`X-XHS-Account`
3. Query 中的 `tenant_id`、`account_id`

如果都不传，则回退到 `configs/accounts.json` 中定义的默认账号。

## 主要文档

- 企业级能力说明：[docs/enterprise_deployment.md](./docs/enterprise_deployment.md)
- Docker 部署：[docs/docker_deployment.md](./docs/docker_deployment.md)
- 源码部署：[docs/source_deployment.md](./docs/source_deployment.md)
- OpenClaw 部署：[docs/openclaw_deployment.md](./docs/openclaw_deployment.md)
- macOS M4 部署：[docs/macos_m4_openclaw.md](./docs/macos_m4_openclaw.md)
- Windows 部署：[docs/windows_enterprise.md](./docs/windows_enterprise.md)
- 企业 API 扩展：[docs/API_ENTERPRISE.md](./docs/API_ENTERPRISE.md)
- 原始 API 文档：[docs/API.md](./docs/API.md)

## 平台支持

- `Windows`
- `macOS Intel / Apple Silicon`
- `Linux`
- `Docker / Docker Compose`
- `OpenClaw`

## 生产使用建议

- 单账号不要同时在多个网页端登录，否则 Cookie 很容易失效
- 初始并发建议从 `global_max_concurrency=8~12`、`account max_concurrency=2~3` 开始压测
- 企业内不同账号必须使用独立 Cookie 文件
- OpenClaw 侧不要依赖“默认账号”，必须显式传账号参数
- 若容器环境无法下载浏览器，优先使用内置 Chromium 的 ARM64 Dockerfile

## 合规与风险提示

本项目用于自动化接入和企业内部效率提升，请在遵守平台规则、账号授权和当地法律法规的前提下使用。涉及发布、评论、点赞等写操作时，应做好限流、审核与审计。
