# macOS M4 + OpenClaw 部署指南

本文档适用于在 `Apple Silicon` 设备上部署 `xiaohongshuritter`，并提供给 OpenClaw 调用。

## 推荐方式

对于 M1、M2、M3、M4 设备，优先推荐：

- Docker 部署
- ARM64 镜像
- OpenClaw 通过 `streamable_http` 访问 `/mcp`

这样可以减少本机浏览器路径差异、依赖缺失和运行时漂移。

## 1. 准备目录

```bash
cd /path/to/xiaohongshuritter
mkdir -p docker/config docker/data docker/images
cp configs/accounts.enterprise.example.json docker/config/accounts.json
```

编辑 `docker/config/accounts.json`，为每个账号配置独立 `cookie_path`。

## 2. 启动服务

```bash
docker compose -f docker/docker-compose.yml up -d
docker compose -f docker/docker-compose.yml logs -f
```

如果你使用的是 OpenClaw 专用轻量包，则在包目录内执行它自带的脚本。

## 3. 验证服务

```bash
curl http://127.0.0.1:18060/health
curl http://127.0.0.1:18060/api/v1/accounts
```

## 4. 在 OpenClaw 中配置 MCP

- Name: `xiaohongshuritter`
- Transport: `streamable_http`
- URL: `http://<你的Mac局域网IP>:18060/mcp`

如果 OpenClaw 与服务在同一台机器：

```text
http://127.0.0.1:18060/mcp
```

## 5. 多账号调用方式

OpenClaw 调用时显式传入：

- `tenant_id`
- `account_id`

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "keyword": "品牌运营"
}
```

## 6. 浏览器说明

Apple Silicon 环境常见问题是容器内浏览器依赖不一致。建议：

- 优先使用已验证的 ARM64 Dockerfile
- 尽量使用内置 Chromium 方案
- 不要依赖运行时再下载浏览器

验证浏览器：

```bash
docker exec -it <container_name> /usr/bin/chromium --version
```

## 7. 故障处理

### 无法生成二维码

- 检查容器内浏览器是否存在
- 检查日志是否打印浏览器路径
- 重建镜像并清理旧缓存

### 能访问健康检查但 OpenClaw 连不上

- 检查 Mac 的局域网 IP 是否变化
- 检查端口 `18060` 是否被防火墙拦截
- 检查 OpenClaw 配置的是 `/mcp` 而不是根路径

### 发布时串账号

- 确认每次调用都传了 `tenant_id` / `account_id`
- 确认不同账号的 Cookie 文件没有复用

## 8. 建议

- M4 场景优先交付 Docker 版本，不建议手工拼依赖
- OpenClaw 侧使用固定版本部署包
- 升级时同时更新版本号、运行手册和 SHA256
