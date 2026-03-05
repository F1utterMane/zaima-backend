# "在吗" APP 后端服务 - 快速上手指南 (Getting Started)

本指南将帮助你从零在本地环境或者服务器上运行起 **"在吗"** 的后端服务。本项目基于 Go 1.25 构建，借助 Docker Compose 可实现一键拉起依赖。

---

## 1. 运行环境准备

在开始之前，请确保你的开发主机上已经安装了以下软件：
- **Go 编译器**: Go 1.25 或以上版本 (本地开发调试需要)
- **Docker**: 最新版的 Docker Desktop 或 Docker Engine
- **Docker Compose**: 用于一键启动 PostgreSQL 和 Redis
- **Git**: 版本控制工具

## 2. 克隆与配置文件初始化

1. 拉取代码到本地工作区：
   ```bash
   git clone <仓库地址> zaima-backend
   cd zaima-backend
   ```
2. 这个项目的所有服务端配置都收拢于 `configs/config.yaml`。里面包含基础服务连接、各类云端鉴权 Key (阿里 OSS、SMS。大模型 Key 等)。
3. 开发环境下，你**不需要**立马填满所有的 OSS 或 Key。核心代码里已经写了大量的本地“伪装/兜底”实现（比如发验证码直接返回、拿新闻给假数据）。

## 3. 一键拉起基础设施 (DB & Cache)

项目根目录提供了一个写好的 `docker-compose.yml`，包含了所需版本的数据库和缓存服务器。

```bash
# 后台一键拉起 PostgreSQL 15 与 Redis 7
docker-compose up -d postgres redis

# 检查它们是否都成功处于 Up (Healthy) 状态
docker-compose ps
```

## 4. 启动 Go 后端服务

基础设施就绪后，下载 Go 第三方依赖包，并直接启动 `main.go`。由于我们在 `database.go` 里加入了 `AutoMigrate`，项目首次启动时会自动在 PostgreSQL 中建立所需的 8 张表。

```bash
# 下载包
go mod tidy

# 运行服务 (默认读取 configs/config.yaml)
go run cmd/api/main.go
```

**成功的输出应该如下：**
```text
[database] PostgreSQL 连接成功，表结构已同步
[database] Redis 连接成功
[main] 🚀 在吗后端服务启动: http://localhost:8080
[main] 📋 API 文档: http://localhost:8080/health
```

## 5. 本地简单测试联调

由于开发环境 `config.yaml` 默认模式是 Debug，许多流程会被简化：

### 5.1 获取本地验证码
```bash
curl -X POST http://localhost:8080/api/v1/auth/sms-code \
-H "Content-Type: application/json" \
-d '{"phone":"13800138000"}'

# 响应会自动返回通过的验证码 (例如：458213)
```

### 5.2 注册并拿 Token
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
-H "Content-Type: application/json" \
-d '{
  "phone": "13800138000",
  "code": "458213",
  "role": 2
}'

# 响应
# {
#   "code": 0,
#   "data": { "is_new": true, "role": 2, "token": "eyJhb....", "user_id": 1 },
#   "message": "success"
# }
```

### 5.3 携带 Token 访问业务接口
随后调用其他接口（如更新用户资料），请把上面返回的 Token 放入 Headers：
```bash
curl -X PUT http://localhost:8080/api/v1/user/profile \
-H "Authorization: Bearer eyJhbGci..." \
-H "Content-Type: application/json" \
-d '{"nickname":"Jack", "city":"北京"}'
```

---

## 6. 面向生产环境 (Production Deployment)

向生产环境发布时，不再建议通过 `go run`。
你可以直接使用我们提供的内置 Dockerfile 来封包，它是多阶段构建的体积趋近于极小化（Alpine 基础镜像不到 20MB）。

```bash
# 1. 构建业务镜像
docker build -t zaima-api:v1.0.0 .

# 2. 修改 docker-compose.yml 里的 `api` 服务镜像，开启全链路编排。然后一行搞定：
docker-compose up -d
```
