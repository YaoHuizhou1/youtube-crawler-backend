# YouTube 双人对话视频抓取系统 - 后端服务

Go 语言实现的后端 API 服务和任务处理 Worker。

## 技术栈

- **语言**: Go 1.22
- **Web 框架**: Gin
- **任务队列**: Asynq (基于 Redis)
- **数据库**: PostgreSQL 16
- **缓存**: Redis 7

## 项目结构

```
youtube-crawler-backend/
├── cmd/
│   ├── api/            # API 服务入口
│   │   └── main.go
│   └── worker/         # Worker 服务入口
│       └── main.go
├── internal/
│   ├── api/            # API 层
│   │   ├── handlers/   # 请求处理器
│   │   ├── middleware/ # 中间件
│   │   └── router.go   # 路由定义
│   ├── config/         # 配置管理
│   ├── models/         # 数据模型
│   ├── repository/     # 数据访问层
│   ├── service/        # 业务逻辑层
│   ├── worker/         # 任务处理器
│   ├── ml/             # ML 服务客户端
│   └── pkg/            # 通用工具包
├── migrations/         # 数据库迁移
├── proto/              # gRPC 定义
├── Dockerfile
├── go.mod
└── go.sum
```

## 环境要求

- Go 1.22+
- PostgreSQL 16+
- Redis 7+
- ML 服务 (可选)

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件配置数据库等连接信息
```

### 3. 运行数据库迁移

```bash
# 使用 psql 执行迁移
psql -U ytcrawler -d youtube_crawler -f migrations/001_init.up.sql
```

### 4. 启动服务

```bash
# 启动 API 服务
go run cmd/api/main.go

# 启动 Worker 服务 (另一个终端)
go run cmd/worker/main.go
```

## 环境变量

| 变量名 | 说明 | 默认值 |
|-------|------|--------|
| `DATABASE_URL` | PostgreSQL 连接字符串 | `postgres://ytcrawler:ytcrawler123@localhost:5432/youtube_crawler` |
| `REDIS_URL` | Redis 连接字符串 | `redis://localhost:6379` |
| `YOUTUBE_API_KEY` | YouTube Data API 密钥 | - |
| `ANTHROPIC_API_KEY` | Claude API 密钥 | - |
| `ML_SERVICE_ADDR` | ML 服务地址 | `localhost:50051` |
| `API_PORT` | API 服务端口 | `8080` |
| `GIN_MODE` | Gin 运行模式 | `debug` |
| `WORKER_CONCURRENCY` | Worker 并发数 | `10` |

## API 接口

### 任务管理

```
POST   /api/v1/tasks              创建任务
GET    /api/v1/tasks              获取任务列表
GET    /api/v1/tasks/:id          获取任务详情
PUT    /api/v1/tasks/:id          更新任务
DELETE /api/v1/tasks/:id          删除任务
POST   /api/v1/tasks/:id/start    启动任务
POST   /api/v1/tasks/:id/pause    暂停任务
POST   /api/v1/tasks/:id/stop     停止任务
```

### 视频管理

```
GET    /api/v1/videos             获取视频列表
GET    /api/v1/videos/:id         获取视频详情
GET    /api/v1/videos/:id/segments 获取对话时间段
PUT    /api/v1/videos/:id/review  人工复核
POST   /api/v1/videos/:id/tags    添加标签
POST   /api/v1/videos/export      导出数据
```

### 统计接口

```
GET    /api/v1/stats/overview     总体统计
GET    /api/v1/stats/tasks/:id    任务统计
GET    /api/v1/stats/tags         标签分布
```

### WebSocket

```
/ws/notifications                 实时通知推送
```

## Docker 构建

```bash
# 构建 API 镜像
docker build --target api -t youtube-crawler-api .

# 构建 Worker 镜像
docker build --target worker -t youtube-crawler-worker .
```

## 开发

### 生成 Proto 文件

```bash
protoc --go_out=. --go-grpc_out=. proto/ml_service.proto
```

### 运行测试

```bash
go test -v ./...
```

### 代码检查

```bash
golangci-lint run
```

## 架构说明

### 分层架构

1. **API 层** (`internal/api/`): 处理 HTTP 请求，路由分发
2. **服务层** (`internal/service/`): 业务逻辑处理
3. **数据层** (`internal/repository/`): 数据库访问
4. **Worker 层** (`internal/worker/`): 异步任务处理

### 核心服务

- **DiscoveryService**: 从 YouTube 发现视频
- **AnalyzerService**: 分析视频内容，判断是否为双人对话
- **TaggerService**: 生成内容标签和时间段标注

### 任务队列

使用 Asynq 实现异步任务处理：

- `discovery:run` - 视频发现任务
- `analysis:analyze` - 视频分析任务
- `tagging:tag` - 内容标签任务

## License

MIT
