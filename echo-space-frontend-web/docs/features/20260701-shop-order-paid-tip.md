# 订单详情已支付提示修复

## 需求背景

用户在订单详情页查看已支付订单时，状态标签显示“已支付”，但顶部结果提示仍显示“本次抢购未成功，Redis 预扣库存会自动回补。”，容易造成订单状态误解。

## 实现概要

- 将订单详情页顶部结果提示收敛为 `activeOrderTip` 计算属性。
- 新增 `isPaidOrder` 判断，优先识别 `orderStatus === 3` 或 `payStatus === 1` 的已支付订单。
- 只有真正的抢购失败状态 `orderStatus === 2` 才展示抢购失败和 Redis 预扣库存回补提示。
- 未知状态展示中性刷新提示，避免误导用户。

## 核心改动文件

- `src/views/shop/OrderCenter.vue`

## 接口变化

无。继续使用原有接口：

- `/web/shop/order/detail`
- `/web/shop/order/loadOrder`
- `/web/shop/order/pay`
- `/web/shop/order/cancel`

## 数据库、缓存、MQ、ES 变化

无。本次只调整前端展示逻辑，不涉及数据库、Redis、RabbitMQ 或 Elasticsearch。

## 验证方式

- 运行 `npm run build` 验证前端构建。
- 人工验证订单详情页：
  - `orderStatus=3` 或 `payStatus=1` 显示“订单已支付成功，商品权益已确认。”
  - `orderStatus=2` 显示抢购失败和 Redis 回补提示。

## 遗留风险

未新增自动化测试。该页面当前主要通过接口返回状态驱动展示，后续若订单状态枚举扩展，需要同步补充 `getOrderResultTip` 的状态文案。
