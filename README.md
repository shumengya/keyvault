# 萌芽密码管理器

个人密码管理器。前端页面编译进 Go 后端的二进制里,运行一个文件、用浏览器打开就能用,不需要再单独部署前端。数据存在本地 SQLite 文件里。

## 功能

- 两个访问密码:必须两个都输对才能进入,输入顺序不限
- 登录有效期默认 1 天,到期要重新登录
- 登录有效期、两个访问密码都能在页面右上角「设置」里修改
- 添加、编辑、删除密码记录,关键词搜索
- 单文件运行,数据是一个 SQLite 文件(`data/mengyakeyvault.db`)

## 快速开始

Windows 下双击或运行 `start.cmd`:没构建过会先自动构建,然后启动服务,数据目录用 `backend\data`。启动后用浏览器打开 `http://localhost:8080`。

> 不要直接运行 `bin\mengyakeyvault.exe`:不设 `MENGYA_DATA_DIR` 时,它的数据目录是**运行时所在目录**下的 `data`,从 `bin\` 里启动会悄悄新建一个空库,看起来就像数据丢了(实际原来的库没动)。启动日志第一行会打印实际使用的数据目录,用错了一眼就能看出来。

也可以手动构建(见下文「构建」)后自己运行二进制,这时记得用 `MENGYA_DATA_DIR` 指定数据目录。

**首次启动**会随机生成两个访问密码,**只在控制台(或 Docker 日志)里打印一次**,请立刻记下:

```
首次启动,已随机生成两个访问密码(登录时两个都要输入,顺序不限)。
  密码 1: ...
  密码 2: ...
```

登录后可以在「设置」里改成自己好记的。也可以在首次启动前用环境变量指定(见下表)。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `MENGYA_ADDR` | `:8080` | 监听地址,例如 `127.0.0.1:9000` 只允许本机访问 |
| `MENGYA_DATA_DIR` | `data` | 数据目录(相对运行时所在目录) |
| `MENGYA_PASSWORD_1` / `MENGYA_PASSWORD_2` | 随机生成 | 首次启动时的两个访问密码,必须同时设置且不能相同,每个最长 72 字节。**只在库里还没有密码时生效**,之后在设置里改的密码不会被它覆盖 |
| `MENGYA_TRUSTED_PROXIES` | 空 | 前面有 nginx 等反向代理时,填代理的 IP,多个用逗号分隔。见下方「放在反向代理后面」 |
| `GIN_MODE` | `release` | 设成 `debug` 会输出调试日志 |

## 放在反向代理后面

默认**不信任** `X-Forwarded-For`,登录限流(同一来源 15 分钟内最多尝试 5 次)按直连地址计数。

如果前面有 nginx,而不设置 `MENGYA_TRUSTED_PROXIES`,所有访客在服务端看来都来自 nginx 那一个地址,会共用一个计数:有人乱试几次,你自己也会被锁 15 分钟。所以要把代理的 IP 填进去,例如:

```
MENGYA_TRUSTED_PROXIES=127.0.0.1
```

nginx 里需要带上这两个头:

```nginx
proxy_set_header X-Forwarded-For $remote_addr;
proxy_set_header X-Forwarded-Proto $scheme;
```

走 HTTPS 时,`X-Forwarded-Proto: https` 会让会话 Cookie 带上 `Secure` 标记。

## 开发

需要 Go 1.26+ 和 Node.js 18+。

```bash
# 终端 1:后端,监听 8080
start_backend.cmd          # 等价于 cd backend && go run .

# 终端 2:前端开发服务器,监听 3000,/api 自动代理到 8080
start_frontend.cmd         # 等价于 cd frontend && npm start
```

开发时打开 `http://localhost:3000`。

后端测试:

```bash
cd backend
go test ./...
```

## 构建

```bash
build.cmd           # Windows 可执行文件  -> bin\mengyakeyvault.exe
build.cmd linux     # Linux amd64         -> bin\mengyakeyvault-linux-amd64
```

脚本分两步:先 `npm run build` 把前端输出到 `backend/web/dist`,再 `go build`,Go 会通过 `go:embed` 把这个目录打进二进制。SQLite 用的是纯 Go 驱动(`modernc.org/sqlite`),不需要 CGO,所以可以在 Windows 上直接交叉编译 Linux 版本。

> 如果跳过前端构建直接 `go build`,能编译通过,但运行时访问页面会看到“前端尚未内嵌”的提示,启动日志里也会有警告。

## Docker

前端和后端在同一个镜像里,只有一个容器:

```bash
docker compose up -d --build
docker compose logs -f        # 首次启动在这里找随机生成的两个密码
```

默认映射 `6464 -> 8080`,数据目录是宿主机的 `/shumengya/docker/mengyakeyvault-backend/data`(沿用旧部署的路径,方便直接迁移)。需要时在 `docker-compose.yml` 里打开 `MENGYA_PASSWORD_1/2` 和 `MENGYA_TRUSTED_PROXIES` 的注释。

## 从旧版迁移

旧版是前后端分开部署、数据存在 `data.json`。迁移步骤:

1. 停掉并删除旧的后端容器(新容器占用同一个 `6464` 端口)
2. 保持数据目录不变,`data.json` 留在里面
3. 启动新版。首次启动会把 `data.json` 里的全部记录导入 SQLite,**保留原来的 id**,日志里会显示“已从旧版 data.json 导入 N 条记录”
4. `data.json` 本身不会被修改或删除,确认没问题后可以自己归档。导入只发生一次,之后即使你把数据删光再重启,也不会把旧数据导回来
5. 旧版的固定密码不再有效(它写在旧文档里,等于公开),访问密码以新版首次启动时生成或指定的为准

旧版前端的 `dist-desktop`(从桌面端跨域调用远程接口)不再可用:登录靠同源 Cookie,接口不再允许跨域。

## 数据与备份

- 数据文件:`<MENGYA_DATA_DIR>/mengyakeyvault.db`,备份时停掉服务后复制这一个文件即可
- 条目里的密码是**明文**存储的(和旧版一致),请保护好数据目录的访问权限
- 两个访问密码用 bcrypt 哈希保存,登录会话令牌只存 SHA-256,数据库泄露也读不出这些明文

## 忘记访问密码

没有“找回”功能,但本机能操作数据文件就能重置:

1. 停掉服务
2. 用任意 SQLite 工具打开 `mengyakeyvault.db`,执行:
   ```sql
   DELETE FROM settings WHERE key IN ('passwordHashFirst', 'passwordHashSecond');
   ```
3. 重新启动。库里没有密码时会按首次启动处理:用 `MENGYA_PASSWORD_1/2` 或随机生成并打印。同时所有已登录的会话都会失效

## 接口

页面和接口同源。除登录相关接口外都需要会话 Cookie,否则返回 401。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/login` | 请求体 `{"first":"...","second":"..."}`,两个密码顺序不限 |
| POST | `/api/logout` | 退出登录 |
| GET | `/api/session` | 返回 `{"authenticated": true/false}`,始终 200 |
| GET | `/api/entries?keyword=xxx` | 条目列表,可按关键词搜索 |
| POST | `/api/entries` | 新增条目 |
| PUT | `/api/entries` | 更新条目(请求体带 `id`) |
| DELETE | `/api/entries/:id` | 删除条目 |
| GET | `/api/settings` | 当前登录有效期(小时) |
| PUT | `/api/settings/session` | 修改登录有效期,`{"hours": 24}`,范围 1 到 8760,只影响之后的新登录 |
| PUT | `/api/settings/passwords` | 修改两个访问密码,需带上当前的两个密码;成功后其他设备的会话失效 |

条目字段:`officialName`(必填)、`account`、`password`、`username`、`phone`、`email`、`website`、`tags`、`logo`。

## 项目结构

```
mengyakeyvault/
├── backend/            Go 后端
│   ├── main.go         路由与条目接口
│   ├── auth.go         登录、会话、限流、访问密码
│   ├── settings.go     设置接口
│   ├── store.go        SQLite 存储与旧数据导入
│   ├── store_auth.go   会话与设置的存储
│   ├── web.go          内嵌前端的静态文件服务
│   └── web/            前端构建产物输出到 web/dist(不入库)
├── frontend/           React 前端
├── build.cmd           一键构建:前端 + 后端 -> bin/
├── Dockerfile          多阶段构建,产出单个镜像
└── docker-compose.yml
```
