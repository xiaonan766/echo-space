# 个人中心视频、评论和弹幕加载接口

## 需求背景

Web 端个人中心的评论管理、弹幕管理和视频筛选下拉需要兼容原 Java 接口：

- `/web/ucenter/loadAllVideo`
- `/interact/ucenter/loadComment`
- `/interact/ucenter/loadDanmu`

其中评论管理和弹幕管理会随着互动数据增长出现深分页问题，本次将这两个管理列表从 `pageNo + OFFSET` 改为基于 `comment_id/danmu_id` 的游标分页。

## 实现概要

- 新增 `UcenterContentService`，集中处理个人中心内容管理接口的登录用户校验、`videoId` 校验、分页大小限制和游标编解码。
- `loadAllVideo` 从 `video_info` 按当前用户查询全部已发布视频，并按 `create_time desc` 返回，继续作为筛选下拉使用。
- `loadComment` 按当前用户作为视频作者查询评论列表，支持可选 `videoId` 筛选，按 `comment_id desc` 游标分页。
- `loadDanmu` 按当前用户拥有的视频查询弹幕列表，支持可选 `videoId` 筛选，按 `danmu_id desc` 游标分页。
- 游标使用 Base64URL JSON，包含 `kind`、`direction`、`anchorId`、`videoId`，避免评论/弹幕游标混用和筛选条件变化后串用。
- 前端评论管理和弹幕管理关闭通用 Table 的数字分页，改为“上一页 / 下一页 + 每页条数”的本页游标分页条。

## 核心改动文件

- `internal/domain/pagination.go`
- `internal/domain/video.go`
- `internal/domain/interact.go`
- `internal/repository/video_repository.go`
- `internal/repository/interact_repository.go`
- `internal/service/web/ucenter_content_service.go`
- `internal/http/handler/web/ucenter_content_handler.go`
- `internal/http/router/web.go`
- `internal/service/web/ucenter_content_service_test.go`
- `internal/http/router/router_test.go`
- `echo-space-frontend-web/src/views/ucenter/CommentList.vue`
- `echo-space-frontend-web/src/views/ucenter/DanmuList.vue`

## 接口变化

### 加载当前用户全部视频

- 路径：`GET/POST /web/ucenter/loadAllVideo`
- 登录要求：需要 web 登录态
- 返回：视频数组，包含 `videoId`、`videoName`、`videoCover`、`createTime`、`danmuCount`、`commentCount`

### 加载当前用户视频评论

- 路径：`GET/POST /interact/ucenter/loadComment`
- 登录要求：需要 web 登录态
- 参数：
  - `pageSize`：每页数量，默认 `15`，最大 `100`
  - `videoId`：可选，筛选指定视频
  - `cursor`：可选，上一页或下一页游标；第一页不传
- 返回：`CursorPaginationResult<UcenterCommentItem>`
  - `list`
  - `pageSize`
  - `nextCursor`
  - `prevCursor`
  - `hasNext`
  - `hasPrev`

### 加载当前用户视频弹幕

- 路径：`GET/POST /interact/ucenter/loadDanmu`
- 登录要求：需要 web 登录态
- 参数：
  - `pageSize`：每页数量，默认 `15`，最大 `100`
  - `videoId`：可选，筛选指定视频
  - `cursor`：可选，上一页或下一页游标；第一页不传
- 返回：`CursorPaginationResult<UcenterDanmuItem>`
  - `list`
  - `pageSize`
  - `nextCursor`
  - `prevCursor`
  - `hasNext`
  - `hasPrev`

## 分页规则

- 评论按 `comment_id desc` 返回。
- 弹幕按 `danmu_id desc` 返回。
- 下一页查询边界：`id < anchorId`。
- 上一页查询边界：`id > anchorId`，数据库按升序取最近一页后，服务层反转为倒序返回。
- 每次查询 `pageSize + 1` 条，用多出的一条判断是否还有更多数据。
- 不再返回 `totalCount/pageNo/pageTotal`，前端不再支持跳转到指定页。

## 数据库、缓存、MQ、ES 变化

- 不新增表，不修改表结构，不自动执行索引变更。
- 不涉及 Redis、RabbitMQ、Elasticsearch。
- 建议由用户确认后手动补充以下索引：

```sql
CREATE INDEX idx_video_comment_owner_id ON video_comment (video_user_id, comment_id);
CREATE INDEX idx_video_comment_owner_video_id ON video_comment (video_user_id, video_id, comment_id);
CREATE INDEX idx_video_info_user_video ON video_info (user_id, video_id);
CREATE INDEX idx_video_danmu_video_id_id ON video_danmu (video_id, danmu_id);
```

## 验证方式

- 运行 `go test ./...`。
- 运行 `npm run build`。
- 登录 web 端进入个人中心：
  - 评论管理能正常加载第一页、下一页、上一页。
  - 弹幕管理能正常加载第一页、下一页、上一页。
  - 切换视频筛选后回到第一页。
  - 切换每页条数后回到第一页。
  - 第一页禁用“上一页”，最后一页禁用“下一页”。

## 遗留风险

- 游标分页不再支持跳到指定页，这是为避免深分页扫描做出的交互调整。
- 建议索引未在代码中自动创建，需要根据实际数据库规模和执行窗口手动确认后执行。
