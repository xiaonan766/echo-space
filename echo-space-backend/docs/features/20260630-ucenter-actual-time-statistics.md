# 个人中心统计接口

## 需求背景

Web 端个人中心需要展示当前用户的视频播放、评论、弹幕、点赞、收藏、硬币和粉丝统计数据。同时，趋势图需要按统计类型获取最近 7 天的增量数据。

本功能补齐 Go 后端中的两个统计接口：

- `/web/ucenter/getActualTimeStatisticsInfo`
- `/web/ucenter/getWeekStatisticsInfo`

## 实现概要

- `StatisticsInfo` 对应 `statistics_info` 表，保存用户每日不同统计类型的增量数据。
- `getActualTimeStatisticsInfo` 查询昨天的各类型增量，并从 `video_info`、`user_focus` 汇总当前总量。
- `getWeekStatisticsInfo` 按登录用户和 `dataType` 查询昨天往前 7 天的数据，按日期升序返回。
- 最近 7 天中没有统计记录的日期会补齐为 `statisticsCount=0`，避免前端趋势图出现空洞。
- 两个接口都注册在 `/web/ucenter` 登录分组下，需要 web 登录态。

## 接口变化

### 获取实时统计

- 前端代理地址：`http://localhost:3000/api/web/ucenter/getActualTimeStatisticsInfo`
- 后端路由：`GET/POST /web/ucenter/getActualTimeStatisticsInfo`
- 登录要求：需要 web 登录态，未登录返回 `901`

返回示例：

```json
{
  "preDayData": {
    "0": 0,
    "1": 0,
    "2": 0,
    "3": 0,
    "4": 0,
    "5": 0,
    "6": 0
  },
  "totalCountInfo": {
    "userCount": 0,
    "playCount": 0,
    "commentCount": 0,
    "danmuCount": 0,
    "likeCount": 0,
    "collectCount": 0,
    "coinCount": 0
  }
}
```

### 获取最近 7 天统计

- 前端代理地址：`http://localhost:3000/api/web/ucenter/getWeekStatisticsInfo`
- 后端路由：`GET/POST /web/ucenter/getWeekStatisticsInfo`
- 登录要求：需要 web 登录态，未登录返回 `901`
- 请求参数：
  - `dataType`：必填，取值 `0..6`

统计类型：

| dataType | 含义 |
| --- | --- |
| 0 | 播放 |
| 1 | 粉丝 |
| 2 | 点赞 |
| 3 | 收藏 |
| 4 | 硬币 |
| 5 | 评论 |
| 6 | 弹幕 |

返回示例：

```json
[
  {
    "statisticsDate": "2026-06-23",
    "userId": "1000000001",
    "dateType": 0,
    "statisticsCount": 0
  }
]
```

## 数据库和缓存变化

- 不新增表，不修改表结构。
- 读取 `statistics_info`、`video_info`、`user_focus`。
- 不新增 Redis、RabbitMQ、Elasticsearch 逻辑。

## 验证方式

- `go test ./...`
- 登录 web 端后进入个人中心首页，确认统计卡片和趋势图能正常展示。
- 调用 `getWeekStatisticsInfo` 并传入 `dataType=0`，确认返回 7 条日期升序数据。

## 遗留风险

- `statistics_info` 依赖离线统计任务写入；如果任务未写入，接口会按日期补 `0`。
- 当前总量直接从 MySQL 汇总，若用户视频量极大，后续可考虑引入统计汇总表或缓存。
