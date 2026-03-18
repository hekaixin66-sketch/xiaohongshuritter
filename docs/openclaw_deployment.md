# OpenClaw 部署指南

本文档用于将 `xiaohongshuritter` 以 MCP 服务形式接入 OpenClaw。

## 适用场景

- 希望让 OpenClaw 稳定调用小红书 MCP
- 希望以“多企业、多账号、多并发”的方式运行
- 希望将部署和运维步骤交付给执行团队

## 推荐交付方式

给 OpenClaw 的最佳交付方式不是直接丢源码仓库地址，而是交付一个可控的轻量部署包：

- 包内自带固定源码快照
- 包内自带部署脚本
- 包内自带运行手册
- 可以锁定浏览器、镜像、配置和版本

这样可以避免 OpenClaw 直接拉取上游源码时出现：

- 拉到错误分支
- 拉到未验证提交
- 浏览器依赖不一致
- 镜像或脚本发生漂移

## OpenClaw 连接参数

在 OpenClaw 中新增 MCP Server：

- Name: `xiaohongshuritter`
- Transport: `streamable_http`
- URL: `http://<server-ip>:18060/mcp`

如果 OpenClaw 与服务部署在同一台机器，也可以使用：

```text
http://127.0.0.1:18060/mcp
```

## 调用规范

企业级场景必须显式传入：

- `tenant_id`
- `account_id`

不要依赖默认账号，否则在多账号环境中容易出现串号。

## 推荐 SOP

### A. 预检查

1. 调用 `list_accounts`
2. 确认目标 `tenant_id` / `account_id` 存在
3. 调用 `check_login_status`

### B. 未登录处理

1. 调用 `get_login_qrcode`
2. 人工扫码
3. 每隔 3 到 5 秒轮询 `check_login_status`
4. 登录成功后再进入业务调用

### C. 业务调用

始终使用同一组 `tenant_id` / `account_id` 执行：

- `search_feeds`
- `list_feeds`
- `publish_content`
- `publish_with_video`
- `get_feed_detail`
- `post_comment_to_feed`
- `reply_comment_in_feed`
- `like_feed`
- `favorite_feed`

## 调用示例

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
  "title": "视频内容上门服务",
  "content": "重庆主城上门，支持图文、视频与账号隔离发布",
  "images": ["/app/images/demo.jpg"],
  "tags": ["内容运营", "企业发布"],
  "visibility": "公开可见"
}
```

## 运行与巡检

推荐检查项：

- `/health` 是否为 200
- `/api/v1/accounts` 是否能列出账号
- 指定账号是否能生成二维码
- OpenClaw 是否能正常访问 `/mcp`
- 单账号并发是否长期占满

## 常见故障

### invalid tenant/account

- 先调用 `list_accounts`
- 确认 `tenant_id` / `account_id` 拼写正确
- 不要依赖默认账号回退

### not logged in

- 重新走二维码登录流程
- 确认账号未在其他网页端同时登录

### account concurrency exhausted

- 降低任务并发
- 增加账号池
- 增加重试和退避

### 容器内浏览器初始化失败

- 优先使用带内置 Chromium 的打包版本
- 重建镜像并确认浏览器路径存在
- 进入容器检查 `/usr/bin/chromium --version`

## 建议

- 交付给 OpenClaw 的版本应带固定 `VERSION` 与 `SOURCE_INFO`
- 每次升级都保留版本号和 SHA256
- 运行手册建议与部署包一起分发
- 生产环境中所有写操作都应做日志留存
