# 周边收藏柜模块

## 需求背景

个人主页需要展示用户已经购买的周边，形成“周边柜”。购买记录属于隐私数据，因此默认不公开，用户本人可以控制整个周边柜是否展示，也可以隐藏单个已购规格。

## 实现概要

- 收藏柜数据复用购物订单，不新增购买流水表。
- 仅统计已支付成功订单：`shop_order.order_status=3` 且 `shop_order.pay_status=1`。
- 列表按 `sku_id` 聚合，返回商品图、商品名、规格、拥有数量、订单次数和最近购买时间。
- 用户本人访问时始终能看到全部已购周边，并能看到每个 SKU 的隐藏状态。
- 非本人访问时，如果收藏柜未公开则返回空列表；公开后只返回未单独隐藏的 SKU。

## 接口变化

- `GET/POST /web/uhome/loadPeripheralCabinet`
  - 游客可访问，可选登录。
  - 参数：`userId,pageNo,pageSize`。
  - 返回：`owner,cabinetVisible,totalCount,pageSize,pageNo,pageTotal,list`。
- `POST /web/uhome/updatePeripheralCabinetVisible`
  - 需要 web 登录态。
  - 参数：`visible=0/1`。
- `POST /web/uhome/updatePeripheralCabinetItemVisible`
  - 需要 web 登录态。
  - 参数：`skuId,visible=0/1`。
  - 只能操作本人已经支付购买过的 SKU。

## 数据库变化

执行 `echo-space-backend/docs/sql/shop_peripheral_cabinet.sql`：

- `user_info.shop_cabinet_visible`：控制整个周边柜是否公开，默认 `0`。
- `shop_cabinet_hidden_item`：记录用户隐藏的单个 SKU。

本次不新增 Redis、RabbitMQ、Elasticsearch 逻辑。

## 前端变化

- 个人主页导航新增“周边柜”。
- 新增 `/user/:userId/peripheralCabinet` 页面。
- 页面支持商品卡片展示、点击跳转商品详情、本人公开开关和单品隐藏/展示。

## 验证方式

- 后端：`go test ./...`
- 前端：`npm run build`
- 手动验证：
  - 默认未公开时，游客看到“空间主人隐藏了周边收藏柜”。
  - 本人能看到全部已购 SKU，并可以打开公开展示。
  - 本人隐藏单个 SKU 后，游客列表不再显示该 SKU。
  - 同一个 SKU 多次购买时，只显示一张卡片且数量累加。

## 修复记录

- 修复周边柜首次加载时短暂误显示“空间主人隐藏了周边收藏柜”的问题。
- 修复接口加载失败时被误判为隐藏状态的问题，改为显示加载失败空状态。
- 修复公开开关请求失败回滚时可能再次触发变更的问题。
- 修复单品隐藏/展示连续点击可能重复提交的问题。

## 遗留风险

- 收藏柜目前直接从订单表聚合，数据量很大时可再引入缓存或单独汇总表。
- 商品名、规格名和封面使用订单快照聚合展示；如果同一 SKU 历史快照不同，展示值可能来自其中一个订单快照。
