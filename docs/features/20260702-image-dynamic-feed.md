# 图片稿件进入粉丝动态

## 需求背景

图片稿件审核通过后，需要像视频一样出现在粉丝的动态页中，点击图片动态进入图库详情页。

## 实现概要

- 新增图片动态事件类型 `DynamicEventTypeImage=2`，图片审核通过后创建动态事件和动态 outbox 消息。
- 图片仍只保存在投稿表和附件表中，不写入公开视频表、不生成下载文件、不写入搜索索引。
- 动态 feed 接口返回混合内容字段：`contentType`、`contentId`、`contentCover`、`contentName`。
- `user_dynamic_feed` 增加 `event_type` 字段区分视频和图片，Redis 动态缓存中图片成员使用 `image:<imageId>`。
- 动态页按内容类型展示“发布了视频”或“发布了图片”，图片动态跳转 `/gallery/:imageId`。

## 核心改动文件

- `echo-space-backend/internal/domain/dynamic.go`
- `echo-space-backend/internal/repository/dynamic_repository.go`
- `echo-space-backend/internal/repository/video_post_repository.go`
- `echo-space-backend/internal/service/web/dynamic_service.go`
- `echo-space-backend/internal/infra/cache/dynamic_cache.go`
- `echo-space-frontend-web/src/views/dynamic/Dynamic.vue`

## 接口与数据变化

- `/web/dynamic/loadFeed` 列表项新增：
  - `contentType`：`0=视频，1=图片`
  - `contentId`：视频 ID 或图片稿件 ID
  - `contentCover`：内容封面
  - `contentName`：内容标题
- 新增 SQL：`echo-space-backend/docs/sql/dynamic_feed_event_type.sql`。
- 不回填历史已审核图片，只处理上线后新审核通过的图片稿件。

## 验证方式

- 后端运行 `go test ./...`。
- web 前端运行 `npm run build`。
- 手动验证：关注作者后审核图片稿件，粉丝动态页出现“发布了图片”，点击进入图库详情页；视频动态仍正常跳转视频详情页。

## 遗留风险

- 上线前需要先执行 `user_dynamic_feed.event_type` 的 SQL，否则新动态写入会缺字段失败。
- 图片动态本次只做曝光和跳转，不接评论、点赞、收藏、投币等互动链路。
