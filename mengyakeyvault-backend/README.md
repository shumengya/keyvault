# 萌芽密码管理器 - 后端

## Docker 部署

### 使用 Docker Compose 部署

1. **构建并启动服务**
   ```bash
   docker-compose up -d --build
   ```

2. **查看日志**
   ```bash
   docker-compose logs -f
   ```

3. **停止服务**
   ```bash
   docker-compose down
   ```

4. **重启服务**
   ```bash
   docker-compose restart
   ```

### 配置说明

- **端口映射**: 容器内 8080 端口映射到主机 6464 端口
- **数据持久化**: 数据存储在 `/shumengya/docker/mengyakeyvault-backend/data/` 目录
- **API 地址**: 通过反向代理访问 `https://keyvault.api.shumengya.top`

### 数据文件

数据文件位置：`/shumengya/docker/mengyakeyvault-backend/data/data.json`

### 本地开发

```bash
go mod tidy
go run main.go
```

服务将在 `http://localhost:8080` 启动
