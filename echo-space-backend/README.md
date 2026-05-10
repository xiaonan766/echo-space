# echo-space-backend

Echo Space backend service.

This project is currently a monolithic Go service. Microservices are not part of the current development stage.

## Stack

- Go
- Gin
- GORM
- MySQL
- Redis
- Elasticsearch
- RabbitMQ

## Current Scope

- Gin HTTP server startup
- Unified response format compatible with the previous Spring Boot `ResponseVO`
- Redis client initialization
- Admin route group

## Environment

The service uses environment variables and provides local defaults:

- `SERVER_HOST`: empty by default
- `SERVER_PORT`: `7070`
- `GIN_MODE`: `debug`
- `REDIS_ADDR`: `localhost:6379`
- `REDIS_PASSWORD`: empty by default
- `REDIS_DB`: `0`
- `REDIS_POOL_SIZE`: `10`
- `MYSQL_DSN`: `root:root@tcp(localhost:3306)/echo_space?charset=utf8mb4&parseTime=True&loc=Local`
- `MYSQL_MAX_IDLE_CONNS`: `5`
- `MYSQL_MAX_OPEN_CONNS`: `20`
- `MYSQL_CONN_MAX_LIFETIME`: `1h`
- `MYSQL_AUTO_MIGRATE`: `false`

## Run

```bash
go run ./cmd/api
```
