# Docker 部署指南

本文档说明如何使用 Docker 部署 `xiaohongshuritter`。

## 适用场景

- 希望快速启动服务
- 希望统一部署到 Linux 服务器或 NAS
- 希望给 OpenClaw 提供稳定的远程 MCP 地址
- 希望隔离浏览器依赖与本地环境差异

## 前置要求

- 已安装 Docker
- 已安装 Docker Compose
- 主机可访问 `18060` 端口
- 已准备好账号配置文件

## 1. 准备配置

```bash
mkdir -p docker/config docker/data docker/images
cp configs/accounts.enterprise.example.json docker/config/accounts.json
```

按实际情况编辑 `docker/config/accounts.json`，确保每个账号都有独立的 `cookie_path`。

## 2. 启动服务

```bash
docker compose -f docker/docker-compose.yml up -d
```

如需查看日志：

```bash
docker compose -f docker/docker-compose.yml logs -f
```

## 3. 验证服务

```bash
curl http://127.0.0.1:18060/health
curl http://127.0.0.1:18060/api/v1/accounts
```

## 4. MCP 地址

容器启动后，默认 MCP 地址为：

```text
http://127.0.0.1:18060/mcp
```

如果给 OpenClaw 或局域网其他机器使用，替换为宿主机 IP：

```text
http://<host-ip>:18060/mcp
```

## 5. 常用运维命令

启动：

```bash
docker compose -f docker/docker-compose.yml up -d
```

停止：

```bash
docker compose -f docker/docker-compose.yml down
```

重建：

```bash
docker compose -f docker/docker-compose.yml down
docker compose -f docker/docker-compose.yml build --no-cache
docker compose -f docker/docker-compose.yml up -d
```

查看日志：

```bash
docker compose -f docker/docker-compose.yml logs --tail=200
```

## 6. 浏览器问题处理

如果容器内浏览器初始化失败：

- 优先检查环境变量 `ROD_BROWSER_BIN`
- ARM64 场景优先使用仓库中的内置 Chromium Dockerfile
- 通过容器内命令确认浏览器是否存在

示例：

```bash
docker exec -it <container_name> /usr/bin/chromium --version
```

如果你使用的是面向 OpenClaw 的源码打包版本，请参考 [openclaw_deployment.md](./openclaw_deployment.md) 中的内置浏览器方案。

## 7. 并发建议

建议通过账号配置文件控制：

- 全局并发：`8` 到 `12` 起步
- 单账号并发：`2` 到 `3` 起步

遇到掉登录或发布异常时，优先降低单账号并发。

## 8. 适合生产的检查项

- `GET /health` 正常
- `GET /api/v1/accounts` 能看到所有账号
- 能成功获取二维码并登录指定账号
- OpenClaw 能稳定访问 `/mcp`
- 发布、搜索、评论链路至少做一次烟雾测试
