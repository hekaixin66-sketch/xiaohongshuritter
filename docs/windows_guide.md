# Windows 快速指南

如果你只是想在 Windows 上尽快把 `xiaohongshuritter` 跑起来，建议优先看这份快速指南。

更完整的企业版说明见 [windows_enterprise.md](./windows_enterprise.md)。

## 快速开始

### 1. 准备环境

- 安装 Go `1.24+`
- 安装 Chrome
- 确认 PowerShell 可正常执行 `go version`

## 2. 准备配置

```powershell
Copy-Item .\configs\accounts.enterprise.example.json .\configs\accounts.json
```

## 3. 设置环境变量

```powershell
$env:XHS_ACCOUNT_CONFIG="configs/accounts.json"
$env:XHS_MAX_CONCURRENCY="12"
$env:XHS_ACCOUNT_MAX_CONCURRENCY="3"
$env:XHS_ACQUIRE_TIMEOUT="180s"
$env:ROD_BROWSER_BIN="C:\Program Files\Google\Chrome\Application\chrome.exe"
```

## 4. 启动服务

```powershell
go run .
```

## 5. 验证

```powershell
Invoke-RestMethod http://127.0.0.1:18060/health
Invoke-RestMethod http://127.0.0.1:18060/api/v1/accounts
```

## 6. 如果遇到问题

- 浏览器启动失败：先检查 `ROD_BROWSER_BIN`
- 登录异常：检查是否同账号多端登录
- 发布异常：检查账号是否已登录、图片路径是否可访问

需要完整说明时，请转到 [windows_enterprise.md](./windows_enterprise.md)。
