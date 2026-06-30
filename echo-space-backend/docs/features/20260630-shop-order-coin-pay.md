# 购物订单硬币支付

## 需求背景

购物模块已经支持用户创建周边抢购订单，并通过 Redis 预扣库存、RabbitMQ 异步锁定 MySQL 库存。本次补齐待支付订单的硬币支付接口，让用户可以使用站内硬币余额完成支付。

## 实现概要

- 新增 `POST /web/shop/order/pay`，接口需要 web 登录。
- 用户只能支付自己的订单，且订单必须是待支付、未支付状态。
- 支付金额按 `1 元 = 100 硬币` 换算为扣减硬币数。
- MySQL 事务内完成扣硬币、订单改为已支付、SKU 锁定库存转为已售库存、写入硬币流水和库存流水。
- 支付成功后标记 Redis 库存占用记录为 `PAID`，Redis 标记失败只记录日志，不回滚已完成的数据库支付。

## 核心改动文件

- `internal/domain/shop_order.go`
- `internal/repository/shop_order_repository.go`
- `internal/service/web/shop_order_service.go`
- `internal/http/handler/web/shop_order_handler.go`
- `internal/http/router/web.go`
- `internal/infra/cache/shop_stock_store.go`

## 接口变化

请求：

```http
POST /web/shop/order/pay
Content-Type: application/x-www-form-urlencoded

orderNo=20260630120000000000000000
```

成功返回订单详情，并额外带上 `currentCoinCount`，用于前端刷新用户硬币数。

## 数据库、缓存与 MQ 变化

- 使用现有 `shop_coin_flow` 记录硬币支付流水。
- 使用现有 `shop_stock_flow` 记录支付成功后的库存转销量流水，`change_type=3`。
- 更新 `shop_order.order_status=3`、`shop_order.pay_status=1`、`shop_order.pay_time`。
- 更新 `shop_sku.locked_stock` 和 `shop_sku.sold_stock`。
- Redis reservation 支付成功后标记为 `PAID`，不回补 Redis 库存。
- 本次支付接口不新增 MQ 消息。

## 验证方式

- `go test ./...`
- 未登录调用 `/web/shop/order/pay` 应返回 `901`。
- 待支付订单支付成功后，检查订单状态、用户硬币、SKU 库存、硬币流水、库存流水均正确变化。
- 重复调用同一个已支付订单不会重复扣硬币。
- 硬币不足时订单、库存、流水不变化。
- 订单过期时接口返回业务错误，并释放已锁定库存。

## 遗留风险

- 本阶段不包含退款、发货、收货和真实第三方支付。
- Redis reservation 标记失败不会影响数据库支付结果，后续可通过补偿任务进一步清理残留 reservation。
