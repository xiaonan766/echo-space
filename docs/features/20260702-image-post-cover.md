# 图片投稿固定封面

## 需求背景

图片投稿需要和视频投稿一样拥有独立封面，图库列表展示固定 16:9 封面，不能再默认使用正文第一张图片作为封面。

## 实现概要

- web 图片投稿表单新增 `封面` 项，复用现有封面选择和裁剪组件。
- 图片封面裁剪比例固定为 16:9，正文图片仍为 1-9 张，多图数量不包含封面。
- 图片投稿提交时如果封面是本地 `File`，先调用现有图片上传接口，提交上传后的资源路径。
- 后端继续使用 `video_info_post.video_cover` 保存封面，不新增数据库字段。
- 后端图片投稿校验要求封面路径安全、扩展名为图片格式，并且资源文件真实存在。
- 图库列表继续使用 `imageCover` 展示卡片封面，并固定 16:9 容器，详情页只展示正文图片列表。

## 核心改动文件

- `echo-space-frontend-web/src/views/ucenter/postvideo/Post.vue`
- `echo-space-frontend-web/src/components/ImageCoverSelect.vue`
- `echo-space-frontend-web/src/views/gallery/GalleryImageItem.vue`
- `echo-space-backend/internal/service/web/video_post_service.go`
- `echo-space-backend/internal/service/web/video_post_service_test.go`

## 接口与数据变化

- `/web/ucenter/postVideo` 图片投稿的 `videoCover` 变为必填，由独立封面上传结果提供。
- 不新增数据库字段，继续写入 `video_info_post.video_cover`。
- 不再在后端或前端把 `imageList[0].sourceName` 自动设置为封面。

## 验证方式

- 后端运行 `go test ./...`。
- web 前端运行 `npm run build`。
- 手动验证图片投稿：未上传封面不能提交，封面裁剪为 16:9，图库列表展示封面图，详情页正文图片不混入封面。

## 遗留风险

- 已有历史图片稿件如果封面数据不符合 16:9，只会在图库卡片中按 16:9 容器展示，不做历史数据重裁剪。
- 图片详情页本次仍只展示正文图片，不接评论、点赞、收藏等互动链路。
