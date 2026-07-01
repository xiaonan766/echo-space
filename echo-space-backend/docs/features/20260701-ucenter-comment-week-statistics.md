# 个人中心近 7 天评论趋势修复

## 需求背景

个人中心首页的评论总量可以正常显示，但点击“评论”后，`/web/ucenter/getWeekStatisticsInfo` 传入 `dataType=5` 返回的近 7 天评论量一直为 0。原因是周趋势接口只读取 `statistics_info`，而评论发布流程只写入 `video_comment` 并更新 `video_info.comment_count`，没有写入离线统计表。

## 实现概要

- `dataType=5` 评论趋势不再依赖 `statistics_info`。
- 后端按当前登录用户作为视频作者，从 `video_comment.post_time` 聚合最近 7 天顶级评论数量。
- 聚合口径与现有 `video_info.comment_count` 保持一致：只统计 `p_comment_id = 0` 的顶级评论。
- 其他统计类型仍沿用 `statistics_info`，避免扩大改动范围。

## 核心改动文件

- `internal/service/web/statistics_service.go`
- `internal/repository/statistics_repository.go`
- `internal/service/web/statistics_service_test.go`

## 接口变化

接口路径和请求参数不变：

- `GET/POST /web/ucenter/getWeekStatisticsInfo`
- `dataType=5`

返回结构不变，仍返回 7 条按日期升序排列的 `StatisticsInfo`。

## 数据库、缓存、MQ、ES 变化

- 不新增表，不修改表结构。
- `dataType=5` 新增读取 `video_comment`。
- 不涉及 Redis、RabbitMQ、Elasticsearch。

## 验证方式

- 运行 `go test ./...`。
- 登录 web 端进入个人中心首页，点击“评论”，确认近 7 天评论趋势会随着对应日期的顶级评论数变化。

## 遗留风险

评论趋势按顶级评论统计，不包含楼中楼回复；这是为了和当前评论总量 `video_info.comment_count` 的维护口径保持一致。其他类型仍依赖 `statistics_info`，如果离线统计任务没有写入，仍会补 0。
