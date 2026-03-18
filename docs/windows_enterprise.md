# Windows 企业版部署指南

本文档用于在 Windows 环境中以源码方式部署 `xiaohongshuritter`。

## 适用场景

- Windows 本地开发与联调
- 企业内网服务器或运维主机
- 需要直接指定本机 Chrome 路径

## 前置要求

- Windows 10 / 11
- Go `1.24+`
- Google Chrome 已安装
- PowerShell 可用

## 1. 准备配置

```powershell
cd E:\daimaku\xhsmcp1
Copy-Item .\configs\accounts.enterprise.example.json .\configs\accounts.json
```

编辑 `configs/accounts.json`，填写你的实际租户和账号信息。

## 2. 设置环境变量

```powershell
$env:XHS_ACCOUNT_CONFIG="configs/accounts.json"
$env:XHS_MAX_CONCURRENCY="12"
$env:XHS_ACCOUNT_MAX_CONCURRENCY="3"
$env:XHS_ACQUIRE_TIMEOUT="180s"
$env:ROD_BROWSER_BIN="C:\Program Files\Google\Chrome\Application\chrome.exe"
```

## 3. 启动服务

```powershell
go run .
```

## 4. 验证服务

```powershell
Invoke-RestMethod http://127.0.0.1:18060/health
Invoke-RestMethod http://127.0.0.1:18060/api/v1/accounts
```

## 5. 登录指定账号

HTTP Header 方式：

```powershell
Invoke-RestMethod `
  -Headers @{ "X-XHS-Tenant" = "default"; "X-XHS-Account" = "main" } `
  -Uri http://127.0.0.1:18060/api/v1/login/qrcode
```

MCP 方式则在工具参数中传：

- `tenant_id`
- `account_id`

## 6. 常见问题

### 浏览器无法启动

- 检查 `ROD_BROWSER_BIN` 是否指向真实的 Chrome 路径
- 确保当前 PowerShell 用户能启动该浏览器
- 如使用企业安全软件，检查是否拦截了浏览器子进程

### 登录状态异常

- 确认同一账号没有同时在其他网页端登录
- 确认 Cookie 文件路径可写
- 降低账号并发

### 发布异常

- 检查标题、内容、图片路径、视频路径是否合法
- 检查账号是否仍处于登录状态
- 检查小红书页面是否发生结构变化

## 7. 建议

- Windows 更适合开发和调试
- 生产环境优先 Docker 或 OpenClaw 专用部署包
- 多账号场景不要复用 Cookie 文件
