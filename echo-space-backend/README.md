# echo-space-backend

Echo Space 后端工程目录。

当前阶段只完成后端空工程初始化和目录占位，不包含业务代码、不包含 Docker 配置。

## 技术栈规划

- Go
- Gin
- GORM
- MySQL
- Redis
- Elasticsearch
- RabbitMQ

## 目录规划

- `cmd/api`: API 服务入口预留目录
- `configs`: 配置文件预留目录
- `docs`: 后端文档预留目录
- `internal/app`: 应用装配预留目录
- `internal/config`: 配置加载预留目录
- `internal/domain`: 领域对象预留目录
- `internal/http/handler`: HTTP 处理器预留目录
- `internal/http/middleware`: HTTP 中间件预留目录
- `internal/http/response`: HTTP 响应结构预留目录
- `internal/http/router`: HTTP 路由预留目录
- `internal/infra/database`: MySQL/GORM 预留目录
- `internal/infra/cache`: Redis 预留目录
- `internal/infra/search`: Elasticsearch 预留目录
- `internal/infra/queue`: RabbitMQ 预留目录
- `internal/repository`: 数据访问层预留目录
- `internal/service`: 业务服务层预留目录
- `pkg`: 可复用公共包预留目录
