# 个人中心实时统计接口

## 需求背景

Web 端个人中心首页需要展示当前用户的视频总播放、评论、弹幕、点赞、收藏、硬币和粉丝数量，同时展示前一天各统计类型的增量数据。原 Java 接口为 `/ucenter/getActualTimeStatisticsInfo`，本次在 Go 后端补齐同名能力。

## 实现概要

- 新增 `StatisticsInfo` 模型，对应 `statistics_info` 表。
- 新增统计 repository，按用户和统计日期查询前一天增量，并从 `video_info` 汇总当前用户视频总数据。
- 粉丝数从 `user_focus` 表按 `focus_user_id` 统计。
- 新增 web 端 service，负责获取前一天日期、补齐缺省统计项和返回前端需要的结构。
- 新增 web handler，并在 `/web/ucenter` 登录分组下注册 `GET/POST /getActualTimeStatisticsInfo`。

## 接口变化

- 前端代理地址：`http://localhost:3000/api/web/ucenter/getActualTimeStatisticsInfo`
- 后端路由：`GET/POST /web/ucenter/getActualTimeStatisticsInfo`
- 登录要求：需要 web 登录态，未登录返回 `901`
- 返回数据：

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

## 数据库和缓存变化

- 不新增表，不修改表结构。
- 读取 `statistics_info`、`video_info`、`user_focus`。
- 本接口不新增 Redis、RabbitMQ、Elasticsearch 逻辑。

## 验证方式

- `go test ./...`
- 手动验证时登录 web 端后进入个人中心首页，检查统计卡片能正常展示。

## 遗留风险

- `statistics_info` 的前一天增量依赖离线统计任务写入；如果任务未写入，接口会返回各项增量为 `0`。
- 当前总数直接从 MySQL 汇总，若用户视频量极大，后续可考虑引入统计汇总表或缓存。
