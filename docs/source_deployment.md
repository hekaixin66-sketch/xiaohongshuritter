# 源码部署指南

本文档说明如何通过源码方式部署 `xiaohongshuritter`。

## 适用场景

- 需要二次开发
- 需要调试浏览器行为或发布流程
- 需要直接控制 Go 运行环境
- Windows 本地运维或企业内网环境

## 前置要求

- Go `1.24+`
- 本机已安装 Chrome 或 Chromium
- 能访问小红书网页端
- 已准备企业级账号配置文件

## 1. 准备配置

```bash
cp configs/accounts.enterprise.example.json configs/accounts.json
```

编辑 `configs/accounts.json`，按实际账号填写：

- `tenant_id`
- `account_id`
- `cookie_path`
- `max_concurrency`

## 2. 设置环境变量

### macOS / Linux

```bash
export XHS_ACCOUNT_CONFIG=configs/accounts.json
export XHS_MAX_CONCURRENCY=12
export XHS_ACCOUNT_MAX_CONCURRENCY=3
export XHS_ACQUIRE_TIMEOUT=180s
export ROD_BROWSER_BIN=/usr/bin/chromium
```

### Windows PowerShell

```powershell
$env:XHS_ACCOUNT_CONFIG="configs/accounts.json"
$env:XHS_MAX_CONCURRENCY="12"
$env:XHS_ACCOUNT_MAX_CONCURRENCY="3"
$env:XHS_ACQUIRE_TIMEOUT="180s"
$env:ROD_BROWSER_BIN="C:\Program Files\Google\Chrome\Application\chrome.exe"
```

## 3. 启动服务

```bash
go run .
```

如需显式指定浏览器路径或端口：

```bash
go run . -bin /path/to/chrome -port :18060
```

## 4. 验证服务

```bash
curl http://127.0.0.1:18060/health
curl http://127.0.0.1:18060/api/v1/accounts
```

## 5. 登录指定账号

通过 Header 指定目标账号：

```bash
curl -H "X-XHS-Tenant: default" \
     -H "X-XHS-Account: main" \
     http://127.0.0.1:18060/api/v1/login/qrcode
```

也可以通过 MCP 工具调用时传：

- `tenant_id`
- `account_id`

## 6. 常见浏览器路径

### Windows

```text
C:\Program Files\Google\Chrome\Application\chrome.exe
```

### macOS

```text
/Applications/Google Chrome.app/Contents/MacOS/Google Chrome
```

### Linux

```text
/usr/bin/google-chrome
/usr/bin/chromium
/usr/bin/chromium-browser
```

## 7. 常见问题

### 浏览器启动失败

- 检查 `ROD_BROWSER_BIN` 是否正确
- 确认浏览器可由当前用户启动
- 检查是否缺少无头运行依赖

### 登录后很快掉线

- 确认同一账号没有在其他网页端同时登录
- 降低同账号并发
- 保证 Cookie 文件独立

### 发布失败

- 检查账号登录状态
- 检查标题、图片、视频、可见范围参数是否合法
- 检查是否被平台风控或页面结构变化影响

## 8. 建议

- 开发联调优先使用源码部署
- 生产环境优先使用 Docker 或 OpenClaw 专用交付包
- 如果是 Apple Silicon + OpenClaw 场景，优先参考 [macos_m4_openclaw.md](./macos_m4_openclaw.md)
