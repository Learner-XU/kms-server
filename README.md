# KMS 知识管理系统

基于 Gitea 的本地知识管理系统，参考 Obsidian + Roam Research 设计理念。

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
| 存储 | Gitea (Git) + MySQL 8+ |
| 认证 | JWT (access token 2h + refresh token 7d) |

---

## 快速部署

### 前置依赖

| 软件 | 最低版本 | 安装方式 (macOS) |
|------|---------|-----------------|
| Go | >= 1.22 | `brew install go` |
| Node.js | >= 18 | `brew install node` |
| MySQL | >= 8.0 | `brew install mysql` |
| Gitea | >= 1.20 | `brew install gitea` |

### 第一步：启动 Gitea

```bash
# 安装并启动 Gitea
brew install gitea
gitea web --config /opt/homebrew/etc/gitea/app.ini
# 访问 http://localhost:3000 完成初始化设置
# 创建管理员账户（记住用户名和 Token）
```

在 Gitea Web UI 中：
1. 完成初始安装（数据库选 SQLite 或 MySQL 均可）
2. 创建管理员账户
3. 创建一个仓库用于存储笔记（如 `xuzong/knowledge-vault`）
4. 进入 **Settings → Applications → Generate New Token**，勾选全部权限，保存 Token

### 第二步：配置 MySQL

```bash
# 启动 MySQL
brew services start mysql

# 创建数据库
mysql -u root -p << 'SQL'
CREATE DATABASE IF NOT EXISTS kms DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;
SQL
```

> KMS 会自动建表（notes、links、users），无需手动建表。

### 第三步：启动后端 (kms-server)

```bash
# 克隆仓库
git clone git@github.com:Learner-XU/kms-server.git
cd kms-server

# 配置环境变量
cat > .env << 'EOF'
PORT=8000
GITEA_URL=http://localhost:3000
GITEA_TOKEN=你的Gitea_Token
GITEA_REPO=你的用户名/你的仓库名
MYSQL_DSN=root:你的密码@tcp(127.0.0.1:3306)/kms?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci
JWT_SECRET=你的JWT密钥_任意复杂字符串
EOF

# 安装依赖并运行
go mod download
go run ./cmd/server/
```

看到 `KMS server starting addr=:8000` 即启动成功。

验证：
```bash
curl http://localhost:8000/health
# {"status":"ok"}
```

### 第四步：启动前端 (kms-web)

```bash
# 克隆仓库
git clone git@github.com:Learner-XU/kms-web.git
cd kms-web

# 安装依赖
npm install

# 启动开发服务器
npx next dev -p 3456 -H 0.0.0.0
```

访问 **http://localhost:3456**，自动跳转注册/登录页。

### 第五步：注册用户并开始使用

1. 打开 http://localhost:3456/register
2. 注册账户（用户名 3-64 字符，密码至少 6 位）
3. 自动登录进入主界面
4. 开始创建笔记！

---

## 环境变量说明

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PORT` | 否 | `8000` | 后端服务端口 |
| `GITEA_URL` | 是 | - | Gitea 服务地址 |
| `GITEA_TOKEN` | 是 | - | Gitea API Token |
| `GITEA_REPO` | 是 | - | 笔记存储仓库（格式：`用户名/仓库名`） |
| `MYSQL_DSN` | 是 | - | MySQL 连接字符串 |
| `JWT_SECRET` | **强烈建议设置** | 内置默认值 | JWT 签名密钥 |
| `WEBHOOK_SECRET` | 否 | - | Gitea Webhook 签名密钥（可选） |

## API 接口

### 认证（公开接口）

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
| GET | `/api/notes` | 列出笔记（可选 `?dir=` 筛选目录） |
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

## Docker 一键部署（推荐）

全栈部署：Gitea + MySQL + kms-server + kms-web。

```bash
# 前置条件：Docker >= 20.10, Docker Compose >= 2.0

# 1. 克隆两个仓库（同级目录）
git clone git@github.com:Learner-XU/kms-server.git
git clone git@github.com:Learner-XU/kms-web.git
cd kms-server

# 2. 配置环境变量
cp .env.deploy.example .env
# 编辑 .env，填写 GITEA_TOKEN

# 3. 先启动 Gitea + MySQL
docker compose up -d mysql gitea
# 访问 http://localhost:3000 完成 Gitea 初始化
# 创建仓库 → 生成 API Token → 填入 .env

# 4. 启动全栈
docker compose up -d
```

| 服务 | 地址 |
|------|------|
| **KMS 前端** | http://localhost:3456 |
| **KMS API** | http://localhost:8000 |
| **Gitea** | http://localhost:3000 |

常用命令：
```bash
docker compose logs -f kms-server  # 查看日志
docker compose restart kms-server  # 重启服务
docker compose down                # 停止
docker compose build --no-cache    # 重新构建
```

---

## 本地部署（不用 Docker）

### 交叉编译（多平台）

```bash
cd kms-server
make build-all   # 编译 darwin/linux × arm64/amd64 四个平台
ls build/        # kms-server-darwin-arm64, kms-server-linux-amd64 等
```

### 手动部署

### 后端编译

```bash
cd kms-server
go build -o kms-server ./cmd/server/
./kms-server
```

### 前端构建

```bash
cd kms-web
npm run build
npx next start -p 3456 -H 0.0.0.0
```

### 建议搭配

- **反向代理**：Nginx/Caddy 统一入口，HTTPS
- **进程管理**：systemd / supervisord / pm2
- **MySQL**：生产环境建议独立部署，配置备份

### Nginx 参考配置

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
├── cmd/server/main.go          # 入口
├── internal/
│   ├── auth/                   # 认证（注册/登录/JWT）
│   │   ├── model.go
│   │   ├── jwt.go
│   │   ├── service.go
│   │   └── handler.go
│   ├── note/                   # 笔记 CRUD
│   ├── search/                 # 搜索索引
│   ├── graph/                  # 知识图谱
│   ├── gitea/                  # Gitea API 客户端
│   ├── sync/                   # 同步/Webhook
│   ├── middleware/              # CORS / JWT 中间件
│   └── config/                 # 配置加载
└── pkg/                        # 工具包（Markdown解析/ID生成）

kms-web/
├── src/
│   ├── app/
│   │   ├── page.tsx            # 主页（四栏布局）
│   │   ├── login/page.tsx      # 登录页
│   │   ├── register/page.tsx   # 注册页
│   │   └── layout.tsx          # 全局布局 + AuthGuard
│   ├── components/
│   │   ├── LeftNav.tsx         # 左侧导航栏
│   │   ├── FileBrowser.tsx     # 文件树浏览器
│   │   ├── MainEditor.tsx      # 笔记编辑器
│   │   ├── RightSidebar.tsx    # 右侧边栏
│   │   ├── GraphView.tsx       # 知识图谱
│   │   ├── DiaryView.tsx       # 日记视图
│   │   ├── NewNoteDialog.tsx   # 新建笔记对话框
│   │   └── AuthGuard.tsx       # 路由守卫
│   └── lib/
│       ├── api.ts              # API 客户端 + Token 管理
│       └── store.ts            # Zustand 全局状态
└── next.config.ts              # Next.js 配置（API 代理）
```

---

## 常见问题

**Q: 启动报 `address already in use`**
```bash
lsof -ti:8000 | xargs kill -9  # 杀掉占用 8000 端口的进程
lsof -ti:3456 | xargs kill -9  # 杀掉占用 3456 端口的进程
```

**Q: MySQL 连接失败**
```bash
brew services restart mysql
mysql -u root -p -e "SELECT 1"
```

**Q: 笔记搜索无结果**
搜索使用 LIKE 匹配，关键词至少 2 个字符。确认后端启动时有 `ReindexAll` 日志。

**Q: 前端显示 401**
检查 JWT_SECRET 是否和后端 `.env` 一致，清除浏览器 localStorage 后重新登录。

**Q: 局域网其他设备无法访问**
前端已配置 `allowedDevOrigins` 和 `0.0.0.0` 监听。确保防火墙允许 3456 和 8000 端口。

## License

MIT
