# 萌芽密码管理器

一个基于 Go 后端和 React 前端的轻量密码管理器应用，支持本地 JSON 存储与关键词搜索。

## 功能特性

- 密码保护访问（默认密码：shumengya520）
- JSON 文件存储（无需数据库）
- 关键词搜索
- 添加、编辑、删除密码记录
- 响应式布局

## 项目结构

```
mengyakeyvault/
├── mengyakeyvault-backend/    # Go 后端
│   ├── main.go                # 主程序
│   ├── go.mod                 # Go 模块文件
│   └── data/                  # 数据目录
├── mengyakeyvault-frontend/   # React 前端
│   ├── src/                   # 源代码
│   ├── public/                # 公共文件
│   └── package.json           # 依赖配置
├── start_backend.bat          # 启动后端脚本
├── start_frontend.bat         # 启动前端脚本
└── build_frontend.bat         # 构建前端脚本
```

## 环境要求

- Go 1.21+
- Node.js 18+ / npm

## 快速开始

### 1. 启动后端

```bash
# Windows
start_backend.bat

# 或手动启动
cd mengyakeyvault-backend
go mod tidy
go run main.go
```

后端将启动在 `http://localhost:8080`。

### 2. 启动前端

```bash
# Windows
start_frontend.bat

# 或手动启动
cd mengyakeyvault-frontend
npm install
npm start
```

前端将启动在 `http://localhost:3000`。

### 3. 访问应用

打开浏览器访问 `http://localhost:3000`，输入默认密码 `shumengya520` 即可使用。

## 数据格式

密码记录包含以下字段：

- **账号类型**：网站、软件
- **账号**：登录账号
- **密码**：登录密码
- **用户名**：用户名
- **手机号**：手机号码
- **邮箱**：邮箱地址
- **网站地址**：网站 URL
- **软件名称**：软件名称
- **标签**：分类标签

## API 接口

- `POST /api/verify` - 验证密码
- `GET /api/entries?keyword=xxx` - 获取密码列表（支持关键词搜索）
- `POST /api/entries` - 添加密码记录
- `PUT /api/entries` - 更新密码记录
- `DELETE /api/entries/:id` - 删除密码记录

## 技术栈

### 后端
- Go 1.21+
- Gin
- JSON 文件存储

### 前端
- React 18
- Axios
- CSS3

## Docker 部署

### 后端部署

1. 进入后端目录
   ```bash
   cd mengyakeyvault-backend
   ```

2. 使用 Docker Compose 启动
   ```bash
   docker-compose up -d --build
   ```

3. 查看日志
   ```bash
   docker-compose logs -f
   ```

### 部署配置

- 容器端口: 8080
- 主机端口: 6464
- 数据持久化路径: `/shumengya/docker/mengyakeyvault-backend/data/`
- API 域名: `https://keyvault.api.shumengya.top`

### 生产环境前端配置

前端在生产环境构建时会自动使用 `https://keyvault.api.shumengya.top/api` 作为 API 地址。

构建生产版本:
```bash
cd mengyakeyvault-frontend
npm run build
```

## 注意事项

- 密码验证成功后会在浏览器本地存储中缓存
- Docker 部署数据存储在 `/shumengya/docker/mengyakeyvault-backend/data/data.json`
- 建议定期备份数据文件
- 确保 favicon.ico 和 logo.png 已放置在 `mengyakeyvault-frontend/public/` 目录下

## 常见问题

### 数据文件在哪里

- 本地开发: `mengyakeyvault-backend/data/data.json`
- Docker 部署: `/shumengya/docker/mengyakeyvault-backend/data/data.json`

### 如何修改默认密码

修改后端常量 `DefaultPassword` 后重启服务即可。

