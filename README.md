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

## 生产构建

```powershell
cd web
npm ci
npm run build
cd ../server
go run .
```

Go 服务会从 `web/dist` 提供前端静态文件。生产环境应在反向代理处启用 HTTPS，并将 Gin 设置为 release 模式。

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

后端测试：

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
