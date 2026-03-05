# 在吗 (Zaima) 后端服务

"在吗" 是一款旨在连接空巢老人与在外打拼子女的银发社交与亲情陪伴应用。
后端服务基于 Go + Gin + GORM + PostgreSQL + Redis 构建，提供 高并发实时通信、RESTful API 以及对接第三方服务的支撑。

## 目录结构
- `cmd/api/`: 入口 `main.go`
- `configs/`: 配置文件目录，提供 `config.yaml.example` 模板
- `internal/`: 核心业务代码
  - `config/`: 配置解析
  - `handler/`: API 路由处理逻辑
  - `model/`: GORM 数据模型
  - `pkg/`: 通用工具 (database, response, utils 等)
  - `ws/`: WebSocket 服务层 (聊天核心)
- `docs/`: 接口文档及技术说明、代码审查报告等
- `test/`: 单元/集成/E2E 系统测试及安全测试
- `Dockerfile` & `docker-compose.yml`: 容器化部署

## 开发与部署

### 配置要求
1. 复制配置模板：`cp configs/config.yaml.example configs/config.yaml`
2. 配置 PostgreSQL 和 Redis 地址、密码等
3. (生产环境) 配置 `ZAIMA_DATABASE_PASSWORD` 等环境变量以覆盖 yaml 中明文凭证

### 运行方式
```bash
# 安装依赖
go mod tidy

# 启动服务
go run cmd/api/main.go --config configs/config.yaml

# 运行全量测试
go test -v ./test/...
```

### Docker 部署
```bash
# 需提前创建并配置 .env 文件给 docker-compose 获取数据库密码等参数
docker-compose up -d --build
```
