# OpenClaw MCP 使用文档

这份文档用于让 OpenClaw 正确接入并调用 `xiaohongshuritter` MCP。

适用目标：
- 让 OpenClaw 学会正确的工具调用顺序
- 避免多租户、多账号场景下串号、误发、漏传参数
- 降低登录失败、发布失败、并发耗尽等常见问题

## 给 OpenClaw 学习的核心规则

把下面这组规则作为 OpenClaw 的固定使用约束：

1. 这是一个多租户、多账号的小红书 MCP，除 `list_accounts` 外，所有业务调用都应显式传入 `tenant_id` 和 `account_id`。
2. 不要假设默认账号，不要省略 `tenant_id` 和 `account_id`。
3. 在任何发布、评论、点赞、收藏前，先调用 `check_login_status`。
4. 如果账号未登录，必须先调用 `get_login_qrcode`，等待人工扫码，再轮询 `check_login_status`，确认登录成功后再继续。
5. 写操作前优先确认目标账号正确，避免跨租户误操作。
6. 图文发布使用 `publish_content`，视频发布使用 `publish_with_video`。
7. `visibility` 推荐直接使用中文值：`公开可见`、`仅自己可见`、`仅互关好友可见`。
8. 系统也兼容常见英文别名：`public`、`self-only`、`friends-only`，但优先使用中文，减少歧义。
9. `reply_comment_in_feed` 至少要提供 `comment_id` 或 `user_id` 其中一个，否则会失败。
10. 如果返回 `invalid tenant/account`，先重新调用 `list_accounts`，不要盲猜账号。
11. 如果返回 `not logged in`，不要继续发布，立即切回二维码登录流程。
12. 如果返回并发耗尽，应该等待、退避重试，或者换账号，不要对同一账号高频重试。

## MCP 连接信息

- Transport: `streamable_http`
- URL: `http://<server-ip>:18060/mcp`
- 本机部署时也可使用：`http://127.0.0.1:18060/mcp`

## 工具清单

### 只读工具

- `list_accounts`
- `check_login_status`
- `get_login_qrcode`
- `list_feeds`
- `search_feeds`
- `get_feed_detail`
- `user_profile`

### 写操作工具

- `publish_content`
- `publish_with_video`
- `post_comment_to_feed`
- `reply_comment_in_feed`
- `like_feed`
- `favorite_feed`
- `delete_cookies`

## 标准 SOP

### SOP 1：首次执行前确认账号

1. 调用 `list_accounts`
2. 找到目标 `tenant_id`
3. 找到目标 `account_id`
4. 后续所有操作都使用同一组 `tenant_id` / `account_id`

### SOP 2：登录检查

1. 调用 `check_login_status`
2. 如果已登录，进入业务流程
3. 如果未登录，进入二维码登录流程

### SOP 3：二维码登录流程

1. 调用 `get_login_qrcode`
2. 将二维码交给人工扫码
3. 每隔 3 到 5 秒调用一次 `check_login_status`
4. 直到返回已登录
5. 登录成功后再执行发布、评论、点赞等写操作

### SOP 4：发布图文

1. `list_accounts`
2. `check_login_status`
3. 如未登录则执行二维码登录
4. 调用 `publish_content`
5. 若失败，先检查素材路径、可见范围、账号登录态

### SOP 5：发布视频

1. `list_accounts`
2. `check_login_status`
3. 如未登录则执行二维码登录
4. 调用 `publish_with_video`
5. 若失败，先检查视频路径、账号登录态、浏览器状态

### SOP 6：搜索后互动

1. `search_feeds` 或 `list_feeds`
2. 如需拿详情，调用 `get_feed_detail`
3. 如需评论，调用 `post_comment_to_feed`
4. 如需回复评论，调用 `reply_comment_in_feed`
5. 如需点赞或收藏，调用 `like_feed` / `favorite_feed`

## 参数规范

### 账户路由参数

除 `list_accounts` 外，推荐所有调用都显式传：

```json
{
  "tenant_id": "default",
  "account_id": "main"
}
```

### 图文发布

工具：`publish_content`

常用参数：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "title": "企业内容发布示例",
  "content": "用于验证 OpenClaw 到 MCP 的标准发布链路。",
  "images": ["/app/images/demo-1.jpg", "/app/images/demo-2.jpg"],
  "tags": ["企业发布", "内容运营"],
  "visibility": "公开可见",
  "products": ["品牌词", "商品关键词"]
}
```

说明：
- `images` 支持图片 URL 或容器/主机内可访问的绝对路径
- `visibility` 推荐使用中文值
- `schedule_at` 支持 RFC3339 时间
- `is_original` 可用于声明原创

### 视频发布

工具：`publish_with_video`

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

说明：
- `video` 要传绝对路径
- `schedule_at` 也支持 RFC3339

### 搜索

工具：`search_feeds`

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

### 笔记详情

工具：`get_feed_detail`

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

### 评论

工具：`post_comment_to_feed`

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "目标笔记ID",
  "xsec_token": "目标笔记xsec_token",
  "content": "这是一条评论示例"
}
```

### 回复评论

工具：`reply_comment_in_feed`

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

### 点赞与收藏

点赞工具：`like_feed`

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "目标笔记ID",
  "xsec_token": "目标笔记xsec_token",
  "unlike": false
}
```

收藏工具：`favorite_feed`

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "目标笔记ID",
  "xsec_token": "目标笔记xsec_token",
  "unfavorite": false
}
```

## OpenClaw 推荐调用模板

### 模板 1：安全发布模板

1. 先 `list_accounts`
2. 再 `check_login_status`
3. 未登录则 `get_login_qrcode`
4. 登录成功后才允许 `publish_content` 或 `publish_with_video`
5. 发布失败时先排查：
   - 参数是否缺少 `tenant_id` / `account_id`
   - 素材路径是否存在
   - `visibility` 是否正确
   - 当前账号是否仍在线

### 模板 2：搜索互动模板

1. `list_accounts`
2. `check_login_status`
3. `search_feeds`
4. 需要详情时 `get_feed_detail`
5. 需要评论时 `post_comment_to_feed`
6. 需要回复时 `reply_comment_in_feed`

## 错误处理规则

### `invalid tenant/account`

- 先调用 `list_accounts`
- 使用返回结果中的真实 `tenant_id` / `account_id`
- 不要猜测默认账号

### `not logged in`

- 立即停止写操作
- 切换到 `get_login_qrcode`
- 扫码后重新 `check_login_status`

### `Account concurrency exhausted`

- 说明该账号当前并发已满
- 先等待当前任务完成
- 做退避重试
- 或切换到其他账号

### 浏览器初始化失败

- 检查容器内浏览器是否存在
- ARM64 / OpenClaw 推荐使用 `package/openclaw-source-lite`
- 必要时执行 `./scripts/rebuild.sh`

## 不建议 OpenClaw 做的事

- 不要跳过 `list_accounts` 直接猜账号
- 不要在未登录时直接发布
- 不要省略 `tenant_id` 和 `account_id`
- 不要在同一账号报并发耗尽后立刻高频重试
- 不要把英文 `public` 当成唯一标准值，优先使用 `公开可见`

## 推荐给 OpenClaw 的学习提示词

下面这段可以直接作为 OpenClaw 的学习材料：

```text
你正在调用 xiaohongshuritter MCP。这是一个多租户、多账号的小红书 MCP 系统。

调用规则：
1. 除 list_accounts 外，所有工具都显式传 tenant_id 和 account_id。
2. 任何写操作前都先检查登录状态。
3. 若未登录，先 get_login_qrcode，再等待人工扫码，再轮询 check_login_status。
4. 图文发布用 publish_content，视频发布用 publish_with_video。
5. visibility 优先使用中文：公开可见、仅自己可见、仅互关好友可见。
6. 遇到 invalid tenant/account 时，重新 list_accounts，不要猜。
7. 遇到 not logged in 时，不继续发布，立即转入二维码登录流程。
8. 遇到 account concurrency exhausted 时，等待或换账号，不要高频重试。
9. reply_comment_in_feed 至少要带 comment_id 或 user_id。
10. 所有业务流程都围绕同一组 tenant_id/account_id 执行，避免串号。
```
