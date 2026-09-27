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
USER send
WORKDIR /app
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/send-server"]
