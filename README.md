# 🍦 frp-panel — frp 管理面板

一个用 Go 编写的单二进制 [frp](https://github.com/fatedier/frp) 管理面板,内置奶油色风格的 WebUI。首次打开自动检测平台、按需拉取最新版 frps/frpc(带进度条),选择服务端或客户端角色后即沿用配置。

> 配置文件(`panel.json`、`frps.toml` / `frpc.toml`)与下载的 frp 二进制都存放在**程序同级目录**。

## ✨ 功能

- **首次向导**:检测系统 / 架构(可手动切换)→ 选择 frps / frpc 角色 → 从 GitHub 拉取最新 frp(实时进度条 + SHA256 校验)→ 设置面板密码。
- **可视化配置**:表单编辑常用项,或直接编辑原始 TOML;客户端支持代理(tcp/udp/http/https/stcp 等)的增删改。
- **进程控制**:一键启动 / 停止 / 重启;客户端支持热重载;实时显示 PID 与运行时长。
- **实时日志**:通过 SSE 流式查看 frp 进程输出。
- **运行监控**:服务端展示累计流量、当前连接、在线客户端与各代理状态;客户端展示各代理连接状态(数据来自 frp 自带的 webServer 管理 API)。
- **在线更新**:检测 frp 新版本,一键升级(自动停止 → 替换二进制 → 恢复运行,带进度条)。
- **面板安全**:bcrypt 密码、会话 Cookie(HttpOnly + SameSite=Strict)、CSRF 头校验。

## 🚀 使用

### 直接运行

把编译好的 `frp-panel`(Windows 为 `frp-panel.exe`)放到任意目录,运行后浏览器打开提示的地址(默认 `http://localhost:8088`),按向导完成初始化即可。

```bash
./frp-panel            # 默认监听 :8088,数据存于程序所在目录
./frp-panel -addr :9000           # 自定义监听地址
./frp-panel -dir /opt/frp-panel   # 自定义数据目录(默认与程序同级)
```

后续启动会自动沿用配置;若开启「开机自启」,面板启动时会自动拉起 frp。

## 🛠 从源码构建

前置:**Go ≥ 1.23**、**Node ≥ 18**。前端构建产物 `web/dist` 会通过 `//go:embed` 打包进二进制,因此需先构建前端再编译 Go。

```bash
# 1) 构建前端
cd web && npm install && npm run build && cd ..

# 2) 编译单二进制
go build -o frp-panel .          # 当前平台
# 交叉编译示例:
GOOS=linux  GOARCH=amd64 go build -o frp-panel       .
GOOS=windows GOARCH=amd64 go build -o frp-panel.exe  .
```

或使用 `make`:

```bash
make build      # 构建前端 + 编译当前平台
make dev        # 提示:两个终端分别跑 `go run . -addr :8088` 和 `cd web && npm run dev`
```

开发模式下,Vite(`npm run dev`,端口 5173)会把 `/api` 代理到本地 Go 后端(`:8088`)。

## 📁 数据文件(均位于程序同级目录)

| 文件 | 说明 |
| --- | --- |
| `panel.json` | 面板元信息:角色、frp 版本/平台、密码哈希、自启开关(含密钥,权限 600) |
| `frps.toml` / `frpc.toml` | frp 实际读取的配置,**唯一真源** |
| `frps` / `frpc`(`.exe`) | 向导拉取的 frp 可执行文件 |

## 🔒 安全建议

面板用于控制网络穿透,请勿将管理端口直接暴露到公网。如需远程访问,建议置于反向代理之后并启用 TLS。面板会话 Cookie 默认未设置 `Secure`(以便在纯 HTTP 下工作),HTTPS 部署时可自行通过反代加固。

## 🧱 技术栈

- 后端:Go(标准库 `net/http`,`pelletier/go-toml`,`x/crypto/bcrypt`),零外部进程依赖外的轻量实现。
- 前端:Vue 3 + Vite + vue-router,纯 CSS 奶油色主题与动画,构建产物内嵌二进制。
- frp:[fatedier/frp](https://github.com/fatedier/frp),运行时按平台拉取。
