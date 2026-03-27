# OpenClaw MCP 全量使用手册

这份文档用于给 OpenClaw 学习和调用 `xiaohongshuritter` MCP，覆盖当前仓库里已经实现的全部 MCP 工具、关键 HTTP 接口、参数规范、标准 SOP 和错误处理规则。

适用范围：
- OpenClaw 通过 MCP 调用小红书能力
- 运维和开发同学核对当前已经开放的工具与接口
- 多租户、多账号、多并发场景下的标准调用约束

## 1. 服务连接方式

### MCP 连接

- Name: `xiaohongshuritter`
- Transport: `streamable_http`
- URL: `http://<server-ip>:18060/mcp`

本机部署时也可使用：

```text
http://127.0.0.1:18060/mcp
```

### 关键 HTTP 接口

- 健康检查：`GET /health`
- 账号列表：`GET /api/v1/accounts`
- 登录状态：`GET /api/v1/login/status`
- 登录二维码：`GET /api/v1/login/qrcode`

## 2. OpenClaw 必须遵守的规则

1. 除 `list_accounts` 外，所有 MCP 业务工具都应显式传入 `tenant_id` 和 `account_id`。
2. 不要依赖默认账号，不要省略 `tenant_id` 和 `account_id`。
3. 任何写操作前，必须先调用 `check_login_status`。
4. 如果未登录，必须先调用 `get_login_qrcode`，等待人工扫码，再轮询 `check_login_status`。
5. 图文发布使用 `publish_content`，视频发布使用 `publish_with_video`。
6. `visibility` 推荐直接使用中文：`公开可见`、`仅自己可见`、`仅互关好友可见`。
7. 系统兼容英文别名 `public`、`self-only`、`friends-only`，但仍优先建议使用中文值。
8. `reply_comment_in_feed` 至少要提供 `comment_id` 或 `user_id` 其中一个。
9. 如果返回 `invalid tenant/account`，先重新调用 `list_accounts`，不要猜账号。
10. 如果返回 `not logged in`，不要继续写操作，立即回到二维码登录流程。
11. 如果返回并发耗尽，应该退避重试或换账号，不要高频重试同一账号。
12. 当前 MCP 不直接支持 `note_url` 或 `note_id` 作为标准入参；涉及笔记定位时，当前主要使用 `feed_id` 和 `xsec_token`。

## 3. 当前已开放的 MCP 工具总览

当前 MCP 共开放 `14` 个工具。

### 3.1 只读工具

| 工具名 | 作用 | 关键参数 |
| --- | --- | --- |
| `list_accounts` | 列出当前可用租户、账号、并发占用 | 无 |
| `check_login_status` | 检查指定账号登录状态 | `tenant_id`, `account_id` |
| `get_login_qrcode` | 获取指定账号登录二维码 | `tenant_id`, `account_id` |
| `list_feeds` | 获取推荐流 | `tenant_id`, `account_id` |
| `search_feeds` | 搜索笔记 | `tenant_id`, `account_id`, `keyword`, `filters` |
| `get_feed_detail` | 获取笔记详情及评论 | `tenant_id`, `account_id`, `feed_id`, `xsec_token` |
| `user_profile` | 获取用户主页信息 | `tenant_id`, `account_id`, `user_id`, `xsec_token` |

### 3.2 写操作工具

| 工具名 | 作用 | 关键参数 |
| --- | --- | --- |
| `publish_content` | 发布图文 | `tenant_id`, `account_id`, `title`, `content`, `images` |
| `publish_with_video` | 发布视频 | `tenant_id`, `account_id`, `title`, `content`, `video` |
| `post_comment_to_feed` | 发表评论 | `tenant_id`, `account_id`, `feed_id`, `xsec_token`, `content` |
| `reply_comment_in_feed` | 回复评论 | `tenant_id`, `account_id`, `feed_id`, `xsec_token`, `content`, `comment_id` 或 `user_id` |
| `like_feed` | 点赞或取消点赞 | `tenant_id`, `account_id`, `feed_id`, `xsec_token`, `unlike` |
| `favorite_feed` | 收藏或取消收藏 | `tenant_id`, `account_id`, `feed_id`, `xsec_token`, `unfavorite` |
| `delete_cookies` | 删除当前账号 Cookie | `tenant_id`, `account_id` |

## 4. MCP 工具详细说明

### 4.1 `list_accounts`

用途：列出当前系统中配置好的全部租户与账号，并返回运行中的并发占用。

示例：

```json
{}
```

典型用途：
- OpenClaw 首次接入前探测账号
- 排查 `invalid tenant/account`
- 选择可用账号执行任务

### 4.2 `check_login_status`

用途：检查指定账号是否已登录。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main"
}
```

### 4.3 `get_login_qrcode`

用途：获取指定账号的二维码登录信息。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main"
}
```

说明：
- 返回结果里通常包含二维码图片数据
- OpenClaw 应将二维码交给人工扫码
- 扫码后轮询 `check_login_status`

### 4.4 `delete_cookies`

用途：删除当前账号的 Cookie，用于强制重新登录。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main"
}
```

使用建议：
- 只在 Cookie 失效或账号状态异常时使用
- 删除后应重新走二维码登录流程

### 4.5 `list_feeds`

用途：获取推荐流内容。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main"
}
```

### 4.6 `search_feeds`

用途：按关键词搜索笔记。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "keyword": "品牌运营",
  "filters": {
    "sort_by": "general",
    "note_type": "",
    "publish_time": "",
    "search_scope": "",
    "location": ""
  }
}
```

说明：
- `keyword` 为必填
- `filters` 为可选
- 可先用 `search_feeds` 找目标笔记，再用 `get_feed_detail`

### 4.7 `get_feed_detail`

用途：获取笔记详情，可选加载评论。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "目标笔记ID",
  "xsec_token": "目标笔记xsec_token",
  "load_all_comments": true,
  "limit": 20,
  "click_more_replies": true,
  "reply_limit": 10,
  "scroll_speed": "normal"
}
```

说明：
- `feed_id` 和 `xsec_token` 是当前标准入参
- `load_all_comments` 为 `true` 时会进一步展开评论
- `scroll_speed` 支持 `slow`、`normal`、`fast`

### 4.8 `user_profile`

用途：根据用户信息获取用户主页数据。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "user_id": "目标用户ID",
  "xsec_token": "目标用户xsec_token"
}
```

### 4.9 `publish_content`

用途：发布图文笔记。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "title": "企业内容发布示例",
  "content": "用于验证 OpenClaw 到 MCP 的图文发布链路。",
  "images": ["/app/images/demo-1.jpg", "/app/images/demo-2.jpg"],
  "tags": ["企业发布", "内容运营"],
  "visibility": "公开可见",
  "is_original": false,
  "products": ["品牌词", "商品关键词"]
}
```

参数说明：
- `images` 支持 URL 或绝对路径
- `schedule_at` 支持 RFC3339 时间
- `visibility` 推荐使用中文值
- `products` 为可选商品关键词

### 4.10 `publish_with_video`

用途：发布视频笔记。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "title": "企业视频内容示例",
  "content": "用于验证视频发布链路。",
  "video": "/app/videos/demo.mp4",
  "tags": ["企业发布", "视频内容"],
  "visibility": "公开可见",
  "products": ["品牌词"]
}
```

参数说明：
- `video` 必须是绝对路径
- `schedule_at` 支持预约时间

### 4.11 `post_comment_to_feed`

用途：对指定笔记发表评论。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "目标笔记ID",
  "xsec_token": "目标笔记xsec_token",
  "content": "这是一条评论示例"
}
```

### 4.12 `reply_comment_in_feed`

用途：回复指定评论。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "目标笔记ID",
  "xsec_token": "目标笔记xsec_token",
  "comment_id": "目标评论ID",
  "content": "这是一条回复示例"
}
```

注意：
- `comment_id` 或 `user_id` 至少要有一个

### 4.13 `like_feed`

用途：点赞或取消点赞。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "目标笔记ID",
  "xsec_token": "目标笔记xsec_token",
  "unlike": false
}
```

说明：
- `unlike=false` 表示点赞
- `unlike=true` 表示取消点赞

### 4.14 `favorite_feed`

用途：收藏或取消收藏。

示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "目标笔记ID",
  "xsec_token": "目标笔记xsec_token",
  "unfavorite": false
}
```

说明：
- `unfavorite=false` 表示收藏
- `unfavorite=true` 表示取消收藏

## 5. 当前 HTTP 接口总览

下列接口属于当前服务的 HTTP 层，适合运维、调试、外部系统或 OpenClaw 接入前的健康检查。

| Method | Path | 作用 |
| --- | --- | --- |
| `GET` | `/health` | 健康检查 |
| `GET` | `/api/v1/login/status` | 检查登录状态 |
| `GET` | `/api/v1/login/qrcode` | 获取登录二维码 |
| `DELETE` | `/api/v1/login/cookies` | 删除 Cookie |
| `GET` | `/api/v1/accounts` | 账号列表与并发状态 |
| `POST` | `/api/v1/publish` | 发布图文 |
| `POST` | `/api/v1/publish_video` | 发布视频 |
| `GET / POST` | `/api/v1/feeds/search` | 搜索笔记 |
| `GET` | `/api/v1/feeds/list` | 推荐流 |
| `POST` | `/api/v1/feeds/detail` | 笔记详情 |
| `POST` | `/api/v1/user/profile` | 用户主页 |
| `GET` | `/api/v1/user/me` | 当前登录用户信息 |
| `POST` | `/api/v1/feeds/comment` | 发表评论 |
| `POST` | `/api/v1/feeds/comment/reply` | 回复评论 |

说明：
- `GET /api/v1/user/me` 当前只在 HTTP 层开放，未单独暴露为 MCP 工具
- 大多数 HTTP 业务接口也支持 `tenant_id` / `account_id` 路由

## 6. 标准调用 SOP

### 6.1 首次接入 SOP

1. 调用 `list_accounts`
2. 确认目标 `tenant_id`
3. 确认目标 `account_id`
4. 确认 OpenClaw 后续任务都使用同一组租户和账号

### 6.2 登录 SOP

1. 调用 `check_login_status`
2. 如果已登录，继续业务流程
3. 如果未登录，调用 `get_login_qrcode`
4. 人工扫码
5. 每隔 3 到 5 秒轮询 `check_login_status`
6. 登录成功后再继续

### 6.3 图文发布 SOP

1. `list_accounts`
2. `check_login_status`
3. 如未登录则 `get_login_qrcode`
4. 调用 `publish_content`
5. 若失败，先检查素材路径、登录态、可见范围和并发状态

### 6.4 视频发布 SOP

1. `list_accounts`
2. `check_login_status`
3. 如未登录则 `get_login_qrcode`
4. 调用 `publish_with_video`
5. 若失败，先检查视频路径、登录态、浏览器状态

### 6.5 搜索与互动 SOP

1. `list_accounts`
2. `check_login_status`
3. `search_feeds` 或 `list_feeds`
4. 需要详情时调用 `get_feed_detail`
5. 需要评论时调用 `post_comment_to_feed`
6. 需要回复时调用 `reply_comment_in_feed`
7. 需要点赞或收藏时调用 `like_feed` / `favorite_feed`

## 7. 账号路由规则

当前系统为多租户、多账号设计，账号路由建议遵循以下原则：

1. MCP 场景下优先显式传入 `tenant_id` 和 `account_id`
2. HTTP 场景可通过 JSON Body、Header、Query 指定
3. 不要让 OpenClaw 依赖默认账号
4. 不同业务任务不要混用账号上下文

推荐标准入参：

```json
{
  "tenant_id": "default",
  "account_id": "main"
}
```

## 8. 常见错误与处理

### `invalid tenant/account`

处理方式：
- 重新调用 `list_accounts`
- 直接使用返回结果中的真实 `tenant_id` / `account_id`
- 不要猜测默认账号

### `not logged in`

处理方式：
- 立即停止写操作
- 调用 `get_login_qrcode`
- 扫码后重新 `check_login_status`

### `Account concurrency exhausted`

处理方式：
- 等待当前任务完成
- 对同账号做退避重试
- 必要时切换账号

### 浏览器初始化失败

处理方式：
- 优先检查浏览器路径或容器内浏览器可用性
- OpenClaw 生产交付推荐使用稳定交付包
- 必要时重建服务并重试登录流程

### `missing comment_id or user_id`

说明：
- `reply_comment_in_feed` 缺少必要回复目标

处理方式：
- 至少补充 `comment_id` 或 `user_id` 其中一个

## 9. OpenClaw 不要做的事

- 不要跳过 `list_accounts` 直接猜账号
- 不要在未登录时直接发布
- 不要省略 `tenant_id` 和 `account_id`
- 不要在同一账号报并发耗尽后高频重试
- 不要把 `public` 当成唯一推荐值，优先用 `公开可见`
- 不要把 HTTP 接口和 MCP 工具名混用
- 不要假设 `note_url` / `note_id` 已经是当前标准 MCP 参数

## 10. 可直接给 OpenClaw 的学习提示词

下面这段可以直接作为 OpenClaw 的学习材料：

```text
你正在调用 xiaohongshuritter MCP。这是一个多租户、多账号的小红书 MCP 服务。

固定规则：
1. 除 list_accounts 外，所有业务调用都显式传 tenant_id 和 account_id。
2. 所有写操作前先 check_login_status。
3. 如果未登录，先 get_login_qrcode，再等待人工扫码，再轮询 check_login_status。
4. 图文发布用 publish_content，视频发布用 publish_with_video。
5. 发布、评论、点赞、收藏前都要确认账号作用域正确。
6. visibility 优先使用中文：公开可见、仅自己可见、仅互关好友可见。
7. reply_comment_in_feed 至少传 comment_id 或 user_id。
8. 遇到 invalid tenant/account 时先 list_accounts，不要猜。
9. 遇到 not logged in 时停止写操作，回到二维码登录流程。
10. 遇到并发耗尽时，等待、退避或换账号，不要高频重试。
11. 当前与笔记详情相关的标准参数是 feed_id 和 xsec_token，不要假设 note_url 或 note_id 已直接支持。
12. 不要混淆 HTTP 接口路径和 MCP 工具名。
```

## 11. 相关文档

- [API.md](./API.md)
- [API_ENTERPRISE.md](./API_ENTERPRISE.md)
- [openclaw_deployment.md](./openclaw_deployment.md)
- [source_deployment.md](./source_deployment.md)
- [enterprise_deployment.md](./enterprise_deployment.md)
