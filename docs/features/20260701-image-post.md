# 图片投稿与稿件类型筛选

## 需求背景

用户投稿页需要在视频投稿之外支持图片投稿，管理后台稿件管理需要能按视频/图片筛选稿件。图片稿件本次只进入投稿与后台审核管理链路，不接入前台播放页、推荐流、搜索索引或动态流。

## 实现概要

- web 投稿页新增 `视频投稿` / `图片投稿` 顶部切换，默认视频投稿。
- 图片投稿支持一次上传 1-9 张图片，复用现有 `/file/uploadImage` 保存图片资源。
- `/web/ucenter/postVideo` 新增可选参数 `contentType`，默认 `0=视频`；图片投稿提交 `contentType=1` 和 `imageList`。
- 后端复用 `video_info_post` 保存稿件基础信息，复用 `video_info_file_post` 保存图片附件，图片附件的 `file_path` 存图片资源路径。
- admin 稿件管理新增稿件类型筛选，图片稿件详情展示图片预览，不初始化视频播放器。

## 核心改动文件

- `echo-space-backend/internal/domain/video_post.go`
- `echo-space-backend/internal/service/web/video_post_service.go`
- `echo-space-backend/internal/repository/video_post_repository.go`
- `echo-space-frontend-web/src/views/ucenter/postvideo/Post.vue`
- `echo-space-frontend-web/src/views/ucenter/postvideo/ImagePostUploader.vue`
- `echo-space-frontend-admin/src/views/content/VideoList.vue`
- `echo-space-frontend-admin/src/views/content/VideoDetail.vue`

## 接口与数据变化

- 新增 SQL：`echo-space-backend/docs/sql/image_post_content_type.sql`
- `video_info_post.content_type`：`0=视频`，`1=图片`。
- `/web/ucenter/postVideo`：
  - `contentType` 可选，默认 `0`。
  - 图片投稿使用 `imageList`，格式为 `[{"sourceName":"images/202607/xxx.png","fileName":"xxx"}]`。
- `/admin/videoInfo/loadVideoList`：
  - 新增 `contentType` 筛选参数。
  - 返回列表项新增 `contentType` 字段。

## 验证方式

- 后端运行 `go test ./...`。
- web 前端运行 `npm run build`。
- admin 前端运行 `npm run build`。

## 遗留风险

- 图片稿件审核通过后暂不出现在前台公开内容流，后续如果需要公开展示，需要新增图片详情页、公开列表查询、搜索/动态/互动策略。
- 数据库变更需要手动执行 SQL，项目未启用 GORM AutoMigrate。
