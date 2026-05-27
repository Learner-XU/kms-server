# KMS 知识管理系统

基于 Gitea 的本地知识管理系统，参考 Obsidian + Roam Research 设计理念。支持 macOS / Linux / Docker 多平台部署。

## 架构

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   kms-web    │────▶│  kms-server  │────▶│    Gitea     │
│  Next.js 16  │     │   Go (Gin)   │     │  Git 存储后端  │
│  :3456       │     │   :8000      │     │  :3000       │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
                     ┌──────┴───────┐
                     │    MySQL     │
                     │  索引/用户    │
                     │  :3306       │
                     └──────────────┘
```

## 功能

- 📝 Markdown 笔记管理（创建/编辑/删除）
- 🔗 双向链接（`[[wikilink]]` 语法）
- 🔍 全文搜索（MySQL LIKE 索引）
- 🕸️ 知识图谱（笔记关联可视化）
- 📅 日记视图
- 👥 多用户 JWT 认证
- 📂 文件树浏览（自动展开）
- 🔐 路由守卫（未登录强制跳转）

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Next.js 16 + React 19 + TypeScript + Tailwind CSS 4 |
| 后端 | Go 1.22+ + Gin + golang-jwt |
| 存储 | Gitea (Git) + MySQL 5.7+ |
| 认证 | JWT (access token 2h + refresh token 7d) |

---

## 部署方式

| 方式 | 适用场景 | 说明 |
|------|---------|------|
| **Docker Compose** | 快速体验 / 生产部署 | 一条命令拉起全栈 |
| **本地开发** | 开发调试 | macOS / Linux 直接运行 |
| **二进制部署** | 服务器 / 无 Docker 环境 | 交叉编译，上传即用 |

---

## 方式一：Docker Compose 一键部署（推荐）

前置条件：Docker >= 20.10, Docker Compose >= 2.0

```bash
# 1. 克隆两个仓库（同级目录）
git clone git@github.com:Learner-XU/kms-server.git
git clone git@github.com:Learner-XU/kms-web.git
cd kms-server

# 2. 配置环境变量
cp .env.deploy.example .env
# 编辑 .env，填写 GITEA_TOKEN 和 JWT_SECRET

# 3. 先启动 Gitea + MySQL
docker compose up -d mysql gitea
# 访问 http://localhost:3000 完成 Gitea 初始化
#   → 注册管理员账户
#   → 创建仓库（如 xuzong/knowledge-vault）
#   → Settings → Applications → Generate New Token
#   → 将 Token 填入 .env 的 GITEA_TOKEN

# 4. 启动全栈
docker compose up -d
```

| 服务 | 地址 | 说明 |
|------|------|------|
| **KMS 前端** | http://localhost:3456 | 注册账户后使用 |
| **KMS API** | http://localhost:8000 | 后端接口 |
| **Gitea** | http://localhost:3000 | Git 存储后端 |
| **MySQL** | localhost:3306 | 数据库 |

常用命令：
```bash
docker compose logs -f kms-server  # 查看日志
docker compose restart kms-server  # 重启服务
docker compose down                # 停止所有服务
docker compose down -v             # 停止并删除数据（危险！）
docker compose build --no-cache    # 重新构建镜像
```

---

## 方式二：本地开发

### 前置依赖

| 软件 | 最低版本 | macOS 安装 | Linux 安装 |
|------|---------|-----------|-----------|
| Go | >= 1.22 | `brew install go` | [golang.org/dl](https://go.dev/dl/) |
| Node.js | >= 18 | `brew install node` | [nodejs.org](https://nodejs.org/) |
| MySQL | >= 5.7 | `brew install mysql` | `apt install mysql-server` |
| Gitea | >= 1.20 | `brew install gitea` | [dl.gitea.io](https://dl.gitea.io/gitea/) |

### 第一步：启动 Gitea

```bash
# macOS
brew install gitea
gitea web

# Linux
wget -O gitea https://dl.gitea.io/gitea/1.22.0/gitea-1.22.0-linux-amd64
chmod +x gitea
./gitea web
```

访问 http://localhost:3000，完成初始化：
1. 注册管理员账户
2. 创建仓库（如 `xuzong/knowledge-vault`）
3. **Settings → Applications → Generate New Token**，保存 Token

### 第二步：配置 MySQL

```bash
# 启动 MySQL
brew services start mysql          # macOS
sudo systemctl start mysql         # Linux

# 创建数据库
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS kms DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

> KMS 启动时自动建表（notes、links、users），无需手动建表。

### 第三步：启动后端

```bash
git clone git@github.com:Learner-XU/kms-server.git
cd kms-server

# 配置环境变量
cat > .env << 'EOF'
PORT=8000
GITEA_URL=http://localhost:3000
GITEA_TOKEN=你的Gitea_Token
GITEA_REPO=你的用户名/仓库名
MYSQL_DSN=root:密码@tcp(127.0.0.1:3306)/kms?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci
JWT_SECRET=你的JWT密钥
EOF

# 运行
go run ./cmd/server/
# 看到 "KMS server starting addr=:8000" 即成功
```

验证：
```bash
curl http://localhost:8000/health
# {"status":"ok"}
```

### 第四步：启动前端

```bash
git clone git@github.com:Learner-XU/kms-web.git
cd kms-web
npm install
npx next dev -p 3456 -H 0.0.0.0
```

### 第五步：开始使用

访问 **http://localhost:3456** → 注册账户 → 开始创建笔记！

---

## 方式三：二进制部署

适合服务器环境，不需要 Docker，上传编译好的二进制即可运行。

### 交叉编译

```bash
cd kms-server

# 编译所有平台
make build-all

# 或只编译目标平台
make build-linux              # linux/amd64 + linux/arm64
make build-darwin             # darwin/amd64 + darwin/arm64

# 查看产物
ls build/
# kms-server-darwin-amd64   (11MB)
# kms-server-darwin-arm64   (9.8MB)
# kms-server-linux-amd64    (10MB)
# kms-server-linux-arm64    (9.6MB)
```

### 部署到服务器

```bash
# 上传二进制
scp build/kms-server-linux-amd64 user@server:/opt/kms/kms-server

# 上传前端构建产物（本地先 npm run build）
scp -r kms-web/.next/standalone/* user@server:/opt/kms-web/
scp -r kms-web/.next/static user@server:/opt/kms-web/.next/
scp -r kms-web/public user@server:/opt/kms-web/

# 配置 .env 并启动
ssh user@server
cd /opt/kms
chmod +x kms-server
./kms-server
```

### 进程管理（systemd）

```ini
# /etc/systemd/system/kms-server.service
[Unit]
Description=KMS Server
After=mysql.service

[Service]
Type=simple
WorkingDirectory=/opt/kms
ExecStart=/opt/kms/kms-server
Restart=always
RestartSec=5
EnvironmentFile=/opt/kms/.env

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable kms-server
sudo systemctl start kms-server
```

---

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PORT` | 否 | `8000` | 后端服务端口 |
| `GITEA_URL` | 是 | - | Gitea 服务地址 |
| `GITEA_TOKEN` | 是 | - | Gitea API Token |
| `GITEA_REPO` | 是 | - | 笔记存储仓库（`用户名/仓库名`） |
| `MYSQL_DSN` | 是 | - | MySQL 连接字符串 |
| `JWT_SECRET` | **强烈建议** | 内置默认值 | JWT 签名密钥 |
| `WEBHOOK_SECRET` | 否 | - | Gitea Webhook 签名密钥 |
| `KMS_API_URL` | 否 | `http://localhost:8000` | 前端→后端地址（Docker 内部通信） |

---

## API 接口

### 认证（公开）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/register` | 注册（返回 token） |
| POST | `/api/auth/login` | 登录（返回 token） |
| POST | `/api/auth/refresh` | 刷新 token |

### 认证（需登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/auth/me` | 获取当前用户信息 |

### 笔记（需登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/notes` | 列出笔记（`?dir=` 筛选目录） |
| POST | `/api/notes` | 创建笔记 |
| GET | `/api/notes/*path` | 获取单篇笔记 |
| PUT | `/api/notes/*path` | 更新笔记 |
| DELETE | `/api/notes/*path` | 删除笔记 |

### 搜索与图谱（需登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/search?q=关键词` | 全文搜索 |
| GET | `/api/backlinks/:id` | 反向链接查询 |
| GET | `/api/graph` | 获取知识图谱数据 |
| GET | `/api/graph/orphans` | 获取孤立笔记 |

### 系统

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查（无需认证） |
| POST | `/webhooks/gitea` | Gitea Webhook 回调 |

---

## Nginx 反向代理

```nginx
server {
    listen 80;
    server_name kms.example.com;

    location / {
        proxy_pass http://127.0.0.1:3456;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 项目结构

```
kms-server/
├── cmd/server/main.go              # 入口
├── internal/
│   ├── auth/                       # 认证（注册/登录/JWT）
│   ├── note/                       # 笔记 CRUD
│   ├── search/                     # 搜索索引（MySQL）
│   ├── graph/                      # 知识图谱
│   ├── gitea/                      # Gitea API 客户端
│   ├── sync/                       # 同步 / Webhook
│   ├── middleware/                  # CORS / JWT 中间件
│   └── config/                     # 配置加载（.env）
├── pkg/                            # 工具包
├── Makefile                        # 交叉编译
├── Dockerfile                      # 多阶段构建
├── docker-compose.yml              # 全栈编排
└── .env.deploy.example             # 部署配置模板

kms-web/
├── src/
│   ├── app/
│   │   ├── page.tsx                # 主页（四栏布局）
│   │   ├── login/page.tsx          # 登录页
│   │   ├── register/page.tsx       # 注册页
│   │   └── layout.tsx              # 全局布局 + AuthGuard
│   ├── components/
│   │   ├── LeftNav.tsx             # 左侧导航栏
│   │   ├── FileBrowser.tsx         # 文件树浏览器
│   │   ├── MainEditor.tsx          # 笔记编辑器
│   │   ├── RightSidebar.tsx        # 右侧边栏
│   │   ├── GraphView.tsx           # 知识图谱
│   │   ├── DiaryView.tsx           # 日记视图
│   │   ├── NewNoteDialog.tsx       # 新建笔记对话框
│   │   └── AuthGuard.tsx           # 路由守卫
│   └── lib/
│       ├── api.ts                  # API 客户端 + Token 管理
│       └── store.ts                # Zustand 全局状态
├── Dockerfile                      # 多阶段构建
└── next.config.ts                  # API 代理 + standalone
```

---

## 常见问题

**Q: 启动报 `address already in use`**
```bash
lsof -ti:8000 | xargs kill -9
lsof -ti:3456 | xargs kill -9
```

**Q: MySQL 连接失败**
```bash
brew services restart mysql         # macOS
sudo systemctl restart mysql        # Linux
mysql -u root -p -e "SELECT 1"     # 测试连接
```

**Q: 笔记搜索无结果**
搜索使用 LIKE 匹配，关键词至少 2 个字符。确认后端启动时有 `ReindexAll` 日志。

**Q: 前端显示 401**
检查 JWT_SECRET 是否和后端 `.env` 一致，清除浏览器 localStorage 后重新登录。

**Q: 局域网其他设备无法访问**
确保后端监听 `0.0.0.0`（默认已配置），防火墙允许 3456 和 8000 端口。

**Q: Docker 内部通信不通**
确认 docker-compose.yml 中 `KMS_API_URL` 指向 `http://kms-server:8000`（容器名而非 localhost）。

## License

MIT
