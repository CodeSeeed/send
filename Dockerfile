FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS server
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
# SQLite needs cgo, so CGO_ENABLED stays on. Compiling with GOMAXPROCS=2
# caps link-stage memory use so a 2-core build host (and the GitHub runner)
# won't OOM during compilation.
ENV GOMAXPROCS=2
RUN CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o /out/send-server .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates \
    && addgroup -S send && adduser -S send -G send
COPY --from=server /out/send-server /usr/local/bin/send-server
# 前端构建产物必须进镜像,否则 NoRoute 静态服务无页面可吐(404)
COPY --from=web /web/dist /app/web/dist
# 让后端默认静态目录指向镜像内的前端(容器无 config.yaml,走默认+env)
ENV SEND_STATIC_DIR=/app/web/dist
USER send
WORKDIR /app
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/send-server"]
