# xiaohongshuritter 企业级部署说明

`xiaohongshuritter` 在原有小红书 MCP 能力基础上，补齐了企业级运行所需的多租户、多账号、多并发与多平台部署能力。

## 企业级能力概览

- 多租户隔离：使用 `tenant_id` 区分不同企业、品牌或业务线
- 多账号托管：使用 `account_id` 区分同租户下的多个运营账号
- 并发控制：支持全局并发和单账号并发限制
- 账号级配置：每个账号可单独设置 Cookie、浏览器路径、代理和 Headless 模式
- 双入口接入：同一服务同时提供 MCP 与 HTTP API
- 多平台部署：支持 Windows、macOS、Linux、Docker、OpenClaw

## 目录建议

推荐的目录结构如下：

```text
xiaohongshuritter/
├─ configs/
│  ├─ accounts.enterprise.example.json
│  └─ accounts.json
├─ data/
│  └─ <tenant>/<account>/cookies.json
├─ docker/
├─ docs/
└─ ...
```

## 账号配置文件

复制示例配置：

```bash
cp configs/accounts.enterprise.example.json configs/accounts.json
```

示例：

```json
{
  "default_tenant": "brand-a",
  "default_account": "main",
  "global_max_concurrency": 12,
  "acquire_timeout_seconds": 180,
  "tenants": [
    {
      "id": "brand-a",
      "name": "Brand A",
      "default_account": "main",
      "accounts": [
        {
          "id": "main",
          "name": "Main Publisher",
          "cookie_path": "./data/brand-a/main/cookies.json",
          "max_concurrency": 3
        },
        {
          "id": "backup",
          "name": "Backup Publisher",
          "cookie_path": "./data/brand-a/backup/cookies.json",
          "max_concurrency": 2
        }
      ]
    }
  ]
}
```

## 字段说明

- `default_tenant`: 默认租户
- `default_account`: 默认账号
- `global_max_concurrency`: 全局最大并发
- `acquire_timeout_seconds`: 获取并发令牌的等待时长
- `tenants[].id`: 租户标识
- `tenants[].default_account`: 当前租户默认账号
- `accounts[].id`: 账号标识
- `accounts[].cookie_path`: 当前账号 Cookie 文件路径
- `accounts[].max_concurrency`: 当前账号最大并发
- `accounts[].browser_bin`: 当前账号专用浏览器路径
- `accounts[].proxy`: 当前账号专用代理

## 账号路由规则

### MCP 工具调用

企业版 MCP 工具支持附带：

- `tenant_id`
- `account_id`

如果不传，将回退到配置文件中的默认账号。

### HTTP API 调用

HTTP API 按以下优先级解析账号：

1. JSON Body 中的 `tenant_id` / `account_id`
2. Header 中的 `X-XHS-Tenant` / `X-XHS-Account`
3. Query 中的 `tenant_id` / `account_id`

## 并发控制建议

生产环境建议从保守值开始压测：

- `global_max_concurrency`: `8` 到 `20`
- 单账号 `max_concurrency`: `2` 到 `4`

如果出现以下问题，应优先降低并发：

- 同账号频繁掉登录
- 发布页面加载变慢
- 浏览器会话异常增多
- OpenClaw 调用高峰期超时增加

## 运行时可观测性

可通过以下接口查看当前账号与并发状态：

- HTTP: `GET /api/v1/accounts`
- MCP Tool: `list_accounts`

返回内容包括：

- 配置文件路径
- 租户和账号列表
- 每个账号的最大并发
- 当前 `in-flight` 数量

## 部署路径

根据场景选择：

- Docker 部署：[docker_deployment.md](./docker_deployment.md)
- 源码部署：[source_deployment.md](./source_deployment.md)
- OpenClaw 部署：[openclaw_deployment.md](./openclaw_deployment.md)
- macOS M4 场景：[macos_m4_openclaw.md](./macos_m4_openclaw.md)
- Windows 场景：[windows_enterprise.md](./windows_enterprise.md)

## 生产最佳实践

- 不同账号必须使用独立 Cookie 文件
- 不要让同一账号在多个网页端同时登录
- OpenClaw 与其他 MCP 客户端必须显式传账号参数
- 写操作要接入内部审核、限流和日志审计
- 高峰期前先做账号容量与并发压测
