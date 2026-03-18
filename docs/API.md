# HTTP API 文档

本文档描述 `xiaohongshuritter` 当前对外提供的 HTTP API。

## Base URL

```text
http://localhost:18060
```

## 通用响应格式

成功响应：

```json
{
  "success": true,
  "data": {},
  "message": "ok"
}
```

错误响应：

```json
{
  "error": "error message",
  "code": "ERROR_CODE",
  "details": "details"
}
```

## 接口总览

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health` | 健康检查 |
| GET | `/api/v1/login/status` | 检查登录状态 |
| GET | `/api/v1/login/qrcode` | 获取登录二维码 |
| DELETE | `/api/v1/login/cookies` | 删除当前账号 Cookie |
| GET | `/api/v1/accounts` | 获取账号列表与并发状态 |
| POST | `/api/v1/publish` | 发布图文 |
| POST | `/api/v1/publish_video` | 发布视频 |
| GET / POST | `/api/v1/feeds/search` | 搜索笔记 |
| GET | `/api/v1/feeds/list` | 获取推荐流 |
| POST | `/api/v1/feeds/detail` | 获取笔记详情 |
| POST | `/api/v1/user/profile` | 获取指定用户主页 |
| GET | `/api/v1/user/me` | 获取当前登录用户 |
| POST | `/api/v1/feeds/comment` | 发表评论 |
| POST | `/api/v1/feeds/comment/reply` | 回复评论 |

## 1. 健康检查

```http
GET /health
```

示例响应：

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "service": "xiaohongshuritter",
    "account": "system",
    "timestamp": "now"
  },
  "message": "ok"
}
```

## 2. 登录相关

### 检查登录状态

```http
GET /api/v1/login/status
```

### 获取登录二维码

```http
GET /api/v1/login/qrcode
```

示例响应：

```json
{
  "success": true,
  "data": {
    "timeout": "4m0s",
    "is_logged_in": false,
    "img": "data:image/png;base64,..."
  },
  "message": "ok"
}
```

### 删除 Cookie

```http
DELETE /api/v1/login/cookies
```

## 3. 账号与运行时状态

### 获取账号列表

```http
GET /api/v1/accounts
```

示例响应：

```json
{
  "success": true,
  "data": {
    "config_path": "configs/accounts.json",
    "accounts": [
      {
        "tenant_id": "default",
        "account_id": "main",
        "cookie_path": "./data/default/main/cookies.json",
        "max_concurrency": 3,
        "current_in_flight": 0
      }
    ],
    "count": 1
  },
  "message": "ok"
}
```

## 4. 发布接口

### 发布图文

```http
POST /api/v1/publish
Content-Type: application/json
```

请求示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "title": "企业内容发布示例",
  "content": "支持图文、视频与账号隔离发布",
  "images": [
    "/app/images/demo-1.jpg",
    "/app/images/demo-2.jpg"
  ],
  "tags": ["内容运营", "企业发布"],
  "visibility": "公开可见"
}
```

说明：

- `images` 支持远程 URL 或本地绝对路径
- `visibility` 支持 `公开可见`、`仅自己可见`、`仅互关好友可见`
- 企业环境建议显式传入 `tenant_id` 与 `account_id`

### 发布视频

```http
POST /api/v1/publish_video
Content-Type: application/json
```

请求示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "title": "视频内容案例",
  "content": "展示企业内容发布完整流程",
  "video": "/app/videos/demo.mp4",
  "tags": ["视频内容"],
  "visibility": "公开可见"
}
```

## 5. 内容搜索与详情

### 搜索笔记

```http
POST /api/v1/feeds/search
Content-Type: application/json
```

请求示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "keyword": "品牌运营",
  "filters": {
    "sort_by": "general"
  }
}
```

### 获取推荐流

```http
GET /api/v1/feeds/list
```

### 获取笔记详情

```http
POST /api/v1/feeds/detail
Content-Type: application/json
```

请求示例：

```json
{
  "tenant_id": "default",
  "account_id": "main",
  "feed_id": "xxxx",
  "xsec_token": "xxxx"
}
```

## 6. 用户与互动

### 获取用户主页

```http
POST /api/v1/user/profile
```

### 获取当前登录用户

```http
GET /api/v1/user/me
```

### 发表评论

```http
POST /api/v1/feeds/comment
```

### 回复评论

```http
POST /api/v1/feeds/comment/reply
```

## 7. 企业级账号路由

账号路由优先级如下：

1. JSON Body: `tenant_id`, `account_id`
2. Header: `X-XHS-Tenant`, `X-XHS-Account`
3. Query: `tenant_id`, `account_id`

如果不传，则回退到默认账号。

Header 示例：

```http
X-XHS-Tenant: default
X-XHS-Account: main
```

## 8. 相关文档

- 企业级说明：[enterprise_deployment.md](./enterprise_deployment.md)
- Docker 部署：[docker_deployment.md](./docker_deployment.md)
- 源码部署：[source_deployment.md](./source_deployment.md)
- OpenClaw 部署：[openclaw_deployment.md](./openclaw_deployment.md)
- 企业 API 补充：[API_ENTERPRISE.md](./API_ENTERPRISE.md)
