# 访客端在线播放人数上报

## 需求背景

播放器需要周期性上报当前分 P 的播放心跳，并展示“X 人正在看”。本次补齐访客端 `reportVideoPlayOnline` 后端接口，保持和原 Java 逻辑一致。

## 实现概要

- 新增 `POST/GET /interact/online/reportVideoPlayOnline`，游客可访问。
- 请求参数为 `fileId` 和 `deviceId`。
- `fileId` 校验为 20 位字母数字。
- `deviceId` 为空时使用 `unknown:{clientIP}` 兜底。
- 使用 Redis TTL 心跳统计在线人数，Redis 异常时返回兜底值 `1`，不影响播放主流程。

## 核心改动文件

- `internal/infra/cache/video_online_store.go`
- `internal/service/web/video_online_service.go`
- `internal/http/handler/web/online_handler.go`
- `internal/http/router/web.go`

## 接口变化

```http
POST /interact/online/reportVideoPlayOnline
Content-Type: application/x-www-form-urlencoded

fileId=VM1M5t5IsNy9bLRWM1ji&deviceId=device-1
```

成功返回：

```json
{
  "status": "success",
  "code": 200,
  "info": "请求成功",
  "data": 1
}
```

## Redis 变化

- 用户心跳 key：`echo-space:video:online:user:{fileId}:{deviceId}`，TTL 8 秒。
- 在线人数 key：`echo-space:video:online:count:{fileId}`，TTL 10 秒。
- 使用 Lua 脚本原子处理首次上报、在线数自增、续期和读取，避免并发首报重复加人数。

## 验证方式

- `go test ./...`
- 未登录访问 `/interact/online/reportVideoPlayOnline` 不返回 `901`。
- 同一 `fileId + deviceId` 连续上报时，8 秒内不会重复增加在线人数。
- 不同 `deviceId` 上报同一 `fileId` 时，在线人数增加。

## 遗留风险

- 在线人数是弱一致统计，Redis 异常时会返回兜底值 `1`。
- 当前逻辑与原 Java 版一致，不扫描所有用户心跳 key 精确重算在线人数。
