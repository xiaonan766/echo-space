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

## Run

```bash
go run ./cmd/api
```
