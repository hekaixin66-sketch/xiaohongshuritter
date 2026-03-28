# OpenClaw 运行手册

这份手册用于 `package/openclaw-source-lite` 的交付与运维，适合 OpenClaw、Apple Silicon、ARM64 和浏览器下载受限环境。

## 1. 部署与启动

```bash
cd package/openclaw-source-lite
chmod +x scripts/*.sh
./scripts/install.sh
```

日常更新推荐：

```bash
./scripts/update.sh
```

如果 Docker 缓存脏了、Chromium 异常或镜像状态可疑：

```bash
./scripts/update.sh --full
```

安装完成后检查：

```bash
./scripts/health.sh
curl -s http://127.0.0.1:18060/api/v1/accounts
```

## 2. OpenClaw 接入方式

- Transport: `streamable_http`
- URL: `http://<服务器IP>:18060/mcp`
- 参考文件：`openclaw-mcp-example.json`

## 3. 调用规范

OpenClaw 每次业务调用都要显式传入：

- `tenant_id`
- `account_id`

不要依赖默认账号，避免跨租户串号或误发。

## 4. 标准调用顺序

### 第一步：确认账号

先调用 `list_accounts`，确认目标租户和账号是否存在。

如果是多账号批量任务，推荐先调用：

- `recommend_publish_accounts`

或者 HTTP：

- `GET /api/v1/scheduler/accounts`
- `POST /api/v1/scheduler/accounts`

优先选择：

- `available=true`
- `queue_depth` 更低
- `active_jobs` 更低
- `cooldown_until` 为空
- `cookie_present=true`

### 第二步：确认登录状态

调用 `check_login_status`，并带上 `tenant_id` 与 `account_id`。

### 第三步：未登录时获取二维码

如果账号未登录：

1. 调用 `get_login_qrcode`
2. 人工扫码
3. 每隔 3 到 5 秒重试 `check_login_status`
4. 登录成功后再执行发布或检索任务

### 第四步：执行业务工具

在同一组 `tenant_id` / `account_id` 下调用业务工具，例如：

- `search_feeds`
- `publish_content`
- `publish_with_video`
- `post_comment_to_feed`
- `reply_comment_in_feed`
- `like_feed`
- `favorite_feed`

对于发布类任务，推荐优先使用异步模式：

- `submit_publish_content_async`
- `submit_publish_video_async`
- `get_publish_job_status`

或者 HTTP：

- `POST /api/v1/publish_async`
- `POST /api/v1/publish_video_async`
- `GET /api/v1/publish/jobs/:job_id`

这样 OpenClaw agent 不需要长时间阻塞等待单次发布完成。

## 5. OpenClaw 示例参数

### 查询账号

```json
{}
```

### 检查登录状态

```json
{
  "tenant_id": "default",
  "account_id": "main"
}
```

### 获取登录二维码

```json
{
  "tenant_id": "default",
  "account_id": "main"
}
```

### 发布图文

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "title": "企业内容发布示例",
  "content": "用于验证 OpenClaw 到 MCP 的标准发布链路。",
  "images": ["/app/images/demo.jpg"],
  "tags": ["企业发布", "内容运营"],
  "visibility": "公开可见"
}
```

## 6. 日常运维命令

```bash
./scripts/start.sh
./scripts/stop.sh
./scripts/logs.sh
./scripts/health.sh
./scripts/smoke.sh
```

## 7. 巡检建议

- 每 5 分钟检查一次 `/health`
- 每 30 分钟检查一次 `/api/v1/accounts`
- 每批任务前检查一次 `/api/v1/scheduler/accounts`
- 发布失败时先确认账号登录态和并发占用

建议重点关注：

- `queue_depth`
- `active_jobs`
- `cooldown_until`
- `global_in_flight`
- `job_runtime.outstanding_jobs`

## 8. 常见故障处理

### 账号不存在或 tenant/account 无效

1. 先调用 `list_accounts`
2. 确认 `tenant_id` 和 `account_id`
3. 修正后重试

### 账号未登录

按第 4 节的二维码登录流程处理。

### 单账号并发耗尽

1. 等待当前任务完成
2. 对同账号任务做退避重试
3. 必要时扩充账号池或下调并发

### 浏览器初始化失败

先执行：

```bash
./scripts/rebuild.sh
```

然后检查：

```bash
docker exec -it xiaohongshuritter-arm64 /usr/bin/chromium --version
docker compose logs --tail=120
```

如果仍然失败，采集日志：

```bash
docker compose logs --tail=300 > xiaohongshuritter-browser.log
```

## 9. 交付给 OpenClaw 的最小要求

1. 使用本包的 `scripts/install.sh` 安装
2. 调用前先执行 `list_accounts`
3. 多账号调度前优先执行 `recommend_publish_accounts`
4. 发布任务优先走异步接口或异步 MCP 工具
5. 每次业务调用显式传入 `tenant_id` 和 `account_id`
6. 登录失败时按手册执行二维码登录流程，不要跳步
