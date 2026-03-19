# 后端设计文档

## 1. 设计理念

### 1.1 核心原则

- **分层架构**：清晰的职责划分，便于维护和测试
- **异步优先**：耗时操作异步处理，避免阻塞
- **松耦合**：服务间通过消息队列通信，独立扩展
- **幂等性**：所有操作可重复执行，结果一致

### 1.2 技术选择

| 选择 | 理由 |
|-----|------|
| Go | 高并发、编译部署简单、内存效率高 |
| Gin | 高性能路由、中间件灵活 |
| Asynq | Go 原生任务队列、轻量高效 |
| PostgreSQL | JSONB 支持、事务可靠 |

---

## 2. 架构分层

```
┌─────────────────────────────────────────────────────────┐
│                      API 层 (Gin)                        │
│  • 接收 HTTP 请求                                        │
│  • 参数验证                                              │
│  • 调用 Service                                          │
│  • 返回响应                                              │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    Service 层                            │
│  • 业务逻辑处理                                          │
│  • 调用外部 API (YouTube, Claude)                        │
│  • 协调多个 Repository                                   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                   Repository 层                          │
│  • 数据库 CRUD                                           │
│  • SQL 查询封装                                          │
│  • 事务管理                                              │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    Worker 层 (Asynq)                     │
│  • 异步任务处理                                          │
│  • 任务重试机制                                          │
│  • 任务链调度                                            │
└─────────────────────────────────────────────────────────┘
```

**目录映射：**

```
internal/
├── api/handlers/    → API 层
├── service/         → Service 层
├── repository/      → Repository 层
└── worker/          → Worker 层
```

---

## 3. 核心流程

### 3.1 任务处理流程

```
用户创建任务 (POST /tasks)
       │
       ▼
用户启动任务 (POST /tasks/:id/start)
       │
       ├──► 更新任务状态为 running
       │
       ▼
入队 Asynq 任务 [discovery:run]
       │
       ▼
┌──────────────────────────────────────┐
│         Discovery Worker              │
│                                       │
│  1. 调用 YouTube API 搜索视频         │
│  2. 去重检查 (youtube_id 唯一)        │
│  3. 时长过滤 (5min - 3h)              │
│  4. 保存视频到数据库                  │
│  5. 为每个视频创建分析任务            │
└───────────────┬──────────────────────┘
                │
                ▼
入队 Asynq 任务 [analysis:analyze] × N
                │
                ▼
┌──────────────────────────────────────┐
│          Analysis Worker              │
│                                       │
│  1. 下载视频片段 (yt-dlp)             │
│  2. 调用 ML 服务                      │
│     • 人脸检测 → visual_score         │
│     • 说话人分离 → audio_score        │
│  3. 元数据分析 → meta_score           │
│  4. 计算置信度                        │
│     score = 0.4×visual + 0.4×audio    │
│            + 0.2×meta                 │
│  5. 判定：score >= 0.6 为对话         │
│  6. 更新视频分析结果                  │
└───────────────┬──────────────────────┘
                │
                │ if is_dialogue = true
                ▼
入队 Asynq 任务 [tagging:tag]
                │
                ▼
┌──────────────────────────────────────┐
│           Tagging Worker              │
│                                       │
│  1. 调用 Claude API 生成标签          │
│  2. 自动规则标签                      │
│  3. 保存标签到数据库                  │
└──────────────────────────────────────┘
```

### 3.2 数据流向

```
YouTube API ──► videos 表 ──► ML 分析 ──► 更新 videos 表
                                │
                                ├──► video_tags 表
                                └──► dialogue_segments 表
```

---

## 4. 关键设计

### 4.1 任务状态机

```
pending ──start──► running ──complete──► completed
                      │
                      ├──pause──► paused ──resume──► running
                      │
                      └──error──► failed
```

### 4.2 视频分析状态机

```
pending ──► analyzing ──► completed
                │
                └──► failed
```

### 4.3 多模态融合

为什么用多模态？单一信号不可靠：
- 标题可能骗点击
- 画面可能有多人但只有一人说话
- 音频可能检测不到画外音

融合策略：
```
视觉 (40%) : 平均 2 张脸 → 1.0 分
音频 (40%) : 2 个说话人 → 1.0 分
元数据 (20%) : 标题含 "interview" → 0.2 分

最终分数 >= 0.6 判定为双人对话
```

### 4.4 幂等性设计

```sql
-- 视频去重
INSERT INTO videos (youtube_id, ...)
ON CONFLICT (youtube_id) DO NOTHING;

-- 标签去重
INSERT INTO video_tags (video_id, tag_name, ...)
ON CONFLICT (video_id, tag_name) DO UPDATE ...;
```

### 4.5 实时通知

```
Worker 完成任务
      │
      ▼
WebSocket Hub 广播
      │
      ▼
前端接收并更新 UI
```

消息类型：
- `task_started` / `task_completed` / `task_failed`
- `video_analyzed` / `video_tagged`
- `discovery_complete`

---

## 5. 扩展点

| 扩展需求 | 实现方式 |
|---------|---------|
| 新增任务类型 | 在 `TaskType` 添加枚举，实现对应 discovery 方法 |
| 新增分析模态 | 实现 Analyzer 接口，调整权重 |
| 新增标签类型 | 在 `TagType` 添加枚举 |
| 水平扩展 | 增加 Worker 实例数或调整 `WORKER_CONCURRENCY` |

---

## 6. 错误处理

```go
// 任务失败时记录错误并通知
if err != nil {
    errMsg := err.Error()
    repo.UpdateStatus(ctx, taskID, TaskStatusFailed, &errMsg)
    hub.Broadcast("task_failed", map[string]interface{}{
        "task_id": taskID,
        "error":   errMsg,
    })
}
```

任务队列自动重试失败的任务（默认 3 次）。

---

## 7. 配置项

| 配置 | 说明 | 影响 |
|-----|------|------|
| `WORKER_CONCURRENCY` | Worker 并发数 | 处理速度 vs 资源占用 |
| `ML_SERVICE_ADDR` | ML 服务地址 | 留空则跳过 ML 分析 |
| `YOUTUBE_API_KEY` | YouTube 配额 | 每日可发现视频数量 |

---

## 8. 文件索引

| 功能 | 文件 |
|-----|------|
| API 路由 | `internal/api/router.go` |
| 任务处理 | `internal/api/handlers/task.go` |
| YouTube 调用 | `internal/service/youtube.go` |
| 视频发现 | `internal/service/discovery.go` |
| 视频分析 | `internal/service/analyzer.go` |
| 标签生成 | `internal/service/tagger.go` |
| 异步任务 | `internal/worker/*.go` |
| 数据模型 | `internal/models/*.go` |
| 数据访问 | `internal/repository/*.go` |
