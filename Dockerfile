# 前端嵌入后端二进制,所以只有一个镜像、一个容器。
# 构建上下文是仓库根目录:docker build -t mengyakeyvault .

# 第一阶段:构建前端。package.json 的 build 脚本会输出到 ../backend/web/dist
FROM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 第二阶段:把前端产物放进后端目录,go build 时一并内嵌
FROM golang:1.26-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/backend/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/mengyakeyvault .

# 第三阶段:只留二进制。tzdata 让 TZ=Asia/Shanghai 这类时区设置生效
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /root
COPY --from=backend /out/mengyakeyvault /usr/local/bin/mengyakeyvault
ENV MENGYA_DATA_DIR=/root/data
VOLUME /root/data
EXPOSE 8080
CMD ["mengyakeyvault"]
