# 企业级 API 扩展说明

本文档说明 `xiaohongshuritter` 在企业级多账号场景下新增的 API 与 MCP 能力。

## 账号路由

可通过三种方式指定目标账号，优先级从高到低如下：

1. JSON Body: `tenant_id`, `account_id`
2. HTTP Header: `X-XHS-Tenant`, `X-XHS-Account`
3. Query 参数: `tenant_id`, `account_id`

如果都未提供，则服务回退到 `configs/accounts.json` 中的默认租户与默认账号。

## 新增接口

### `GET /api/v1/accounts`

返回当前已加载的账号列表与运行时并发状态。

示例响应：

```json
{
  "success": true,
  "data": {
    "config_path": "configs/accounts.json",
    "accounts": [
      {
        "tenant_id": "brand-a",
        "tenant_name": "Brand A",
        "account_id": "main",
        "account_name": "Main Account",
        "cookie_path": "./data/brand-a/main/cookies.json",
        "max_concurrency": 3,
        "current_in_flight": 0,
        "default_tenant": true,
        "default_account": true
      }
    ],
    "count": 1
  },
  "message": "ok"
}
```

## MCP 扩展

### 新增工具

- `list_accounts`

### 已扩展工具参数

以下 MCP 工具支持附加：

- `tenant_id`
- `account_id`

适用于：

- `check_login_status`
- `get_login_qrcode`
- `publish_content`
- `publish_with_video`
- `search_feeds`
- `list_feeds`
- `get_feed_detail`
- `post_comment_to_feed`
- `reply_comment_in_feed`

## MCP 调用示例

```json
{
  "name": "search_feeds",
  "arguments": {
    "tenant_id": "brand-a",
    "account_id": "main",
    "keyword": "品牌营销"
  }
}
```

## 建议

- 企业环境中不要依赖默认账号回退
- 所有写操作都显式传入账号参数
- OpenClaw 场景中先调用 `list_accounts` 再执行业务操作
