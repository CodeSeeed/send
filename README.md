# Send

Send 是一个由管理员上传、通过链接公开分享文件的轻量级应用。后端使用 Go、Gin、GORM 和 SQLite，前端使用 Vue 3、TypeScript、Vite 与 Element Plus。

## 功能

- 单管理员注册、登录、改密与退出
- 文件扩展名、MIME 类型和大小校验
- 分享密码、有效期和最大下载次数
- 分享链接与二维码
- 可选生成 8 位文件收件码，在公开主页输入后提取文件
- PDF、图片、文本和 JSON 在线预览
- 管理后台搜索、分页、删除文件与调整上传大小
- IP 限流、短期一次性下载凭证和过期文件清理

## 开发环境

- Go 1.25 或兼容版本
- Node.js 20+ 与 npm

启动后端：

```powershell
cd server
go run .
```

后端从 `server/config/config.yaml` 读取配置，默认监听 `8081`。首次启动会自动创建 `server/data` 和 `server/uploads`。

启动前端开发服务器：

```powershell
cd web
npm ci
npm run dev
```

Vite 默认监听 `5174`，并将 `/api` 代理到 `http://localhost:8081`。公开主页用于输入收件码；首次打开 `/#/admin/login` 时创建管理员，之后 `/#/upload` 上传页和管理后台都要求登录。

上传时可以选择“生成文件收件码”。收件码由 8 位易读大写字母和数字组成，不区分输入大小写；它只负责定位文件，不会绕过文件访问密码、有效期或下载次数限制。

## 生产构建与部署

### 构建

前端产物与后端二进制需要分别构建。后端使用需要 CGO 的 SQLite 驱动（`go-sqlite3`），因此**构建机器上必须安装 C 编译器**：Linux 上使用系统 gcc；Windows 上可使用 MSYS2 的 mingw64（如 `C:\msys64\mingw64\bin`），构建前把它的目录加入 PATH。

```bash
# 1. 构建前端（输出到 web/dist）
cd web
npm ci
npm run build

# 2. 构建后端二进制
cd ../server
CGO_ENABLED=1 go build -o send-server .
```

### 运行

二进制以 `server/` 为工作目录运行——配置文件、数据库和上传目录的路径都相对于工作目录：

```bash
cd server
GIN_MODE=release ./send-server
```

- 首次启动会自动创建 `server/data` 与 `server/uploads`，并在启动时清理数据库与磁盘不一致的文件（孤儿文件、中断上传产生的残缺记录）。
- `GIN_MODE=release` 让 Gin 进入 release 模式（关闭调试输出）；Windows 下可用 `$env:GIN_MODE="release"`。
- 服务监听 `server/config/config.yaml` 中配置的端口（默认 `8081`），并直接提供 `web/dist` 的静态页面，与 API 同源。

### 反向代理 + HTTPS（推荐）

生产环境建议把 Go 服务放在反向代理之后，由代理终结 TLS。此时需要在 `server/config/config.yaml` 中配置：

- `trusted_proxies`：填入反向代理的 IP（如 `127.0.0.1`、`::1` 或内网网关）。为空时 Gin 不信任任何代理，直接与公网通信的客户端无法伪造 `X-Forwarded-For` 绕过限流。
- `allowed_origins`：同源部署时留空即可；仅当前端与 API 跨域时才填写前端来源。
- 反向代理必须**覆盖**转发头，不能追加客户端提交的原始值。

nginx 示例（`client_max_body_size` 必须大于上传大小上限，否则大文件上传会被代理以 413 拒绝）：

```nginx
server {
    listen 443 ssl;
    server_name send.example.com;

    # SSL 证书配置（此处省略）

    # 上传大小上限：应不小于管理后台中配置的最大上传大小
    client_max_body_size 50m;

    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $remote_addr;    # 覆盖，而非追加
    proxy_set_header X-Forwarded-Proto $scheme;

    # 大文件上传需要更长的代理超时
    proxy_read_timeout 60s;
    proxy_send_timeout 60s;

    location / {
        proxy_pass http://127.0.0.1:8081;
    }
}
```

### 数据与备份

所有持久化数据都在 `server/data`（SQLite 数据库）与 `server/uploads`（上传的文件）两个目录，备份时两者都要保留。迁移到新机器时，将这两个目录连同 `server/config/config.yaml` 一起拷贝即可。

### 升级

1. 拉取新代码，重新执行上面「构建」的两步；
2. 用新二进制重启服务，数据库结构变更由启动时的自动迁移完成；
3. 启动时会自动清理孤儿文件，无需手动干预。

## 网络与安全配置

`server/config/config.yaml` 中的重要配置：

- `allowed_origins`：仅在前后端跨域部署时填写前端来源；同源部署可留空。
- `trusted_proxies`：默认不信任任何代理。只有确实位于受控反向代理之后时才填写代理 IP。
- `rate_limit_localhost`：为 `true` 时 localhost 也参与限流；默认 `false`。

反向代理必须覆盖客户端转发头，不能追加来自公网请求的原始值。例如 nginx：

```nginx
proxy_set_header X-Forwarded-For $remote_addr;
proxy_set_header X-Forwarded-Proto $scheme;
```

## 验证

后端测试（需要 C 编译器，见「生产构建与部署」）：

```powershell
cd server
go test ./...
```

前端类型检查与构建：

```powershell
cd web
npm run build
```

## 目录

- `server/controller`：HTTP 参数处理与文件响应
- `server/service`：认证、文件和系统设置业务逻辑
- `server/middleware`：认证、CORS、限流、安全头和静态文件
- `server/model`：SQLite 数据模型
- `web/src/views`：上传、分享、登录和管理页面
- `web/src/api`：前端 API 封装

## 许可证

[MIT](LICENSE)
