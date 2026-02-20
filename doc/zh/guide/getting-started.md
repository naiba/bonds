# 快速开始

## 方式一：Docker（推荐）

使用 Docker 是运行 Bonds 最简单的方式：

```bash
# 下载 docker-compose.yml
curl -O https://raw.githubusercontent.com/naiba/bonds/main/docker-compose.yml

# 启动服务
docker compose up -d
```

打开 **http://localhost:8080**，注册账号即可使用。

### 自定义配置

编辑 `docker-compose.yml` 设置 JWT 密钥和其他选项：

```yaml
environment:
  - JWT_SECRET=你的密钥         # ⚠️ 生产环境务必修改！
  - APP_URL=https://bonds.example.com
  - APP_ENV=production
```

### 持久化存储

默认的 `docker-compose.yml` 已挂载数据卷，SQLite 数据库和上传文件在容器重启后仍然保留。

## 方式二：下载预编译版本

从 [GitHub Releases](https://github.com/naiba/bonds/releases) 下载最新版本：

```bash
# 设置必要的环境变量
export JWT_SECRET=你的密钥
export APP_ENV=production

# 运行服务器
./bonds-server
```

服务器在 **http://localhost:8080** 启动，自带前端界面和 SQLite 数据库。

### 数据目录

默认情况下，Bonds 在工作目录中存储数据：

| 路径 | 用途 |
|------|------|
| `bonds.db` | SQLite 数据库 |
| `uploads/` | 上传的文件（照片、文档） |
| `data/bonds.bleve/` | 全文搜索索引 |
| `data/backups/` | 自动备份 |

## 方式三：从源码构建

**环境要求**：Go 1.25+、[Bun](https://bun.sh) 1.x

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# 安装依赖
make setup

# 构建单二进制文件（内嵌前端）
make build-all

# 运行
export JWT_SECRET=你的密钥
./server/bin/bonds-server
```

## 登录后的第一步

1. **创建 Vault** — Vault 是联系人的隔离容器。你可以创建「家人」和「工作」两个 Vault。
2. **添加联系人** — 在 Vault 中创建联系人，添加详细信息、照片、笔记。
3. **设置提醒** — 不再忘记生日或重要日期。配置邮件或 Telegram 通知。
4. **邀请他人** — 通过邮件邀请家人共享 Vault，并设置适当的权限级别。

## 系统要求

- **CPU**：任何现代 64 位处理器
- **内存**：空闲时约 50 MB，随使用量增长
- **磁盘**：极少；取决于上传文件大小
- **操作系统**：Linux（amd64、arm64）、macOS、Windows
- **数据库**：SQLite（内置）或 PostgreSQL 14+
