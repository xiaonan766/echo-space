# 用户主页收藏列表接口

## 需求背景

Web 端用户主页的“收藏”页需要加载指定用户收藏过的视频。原 Java 接口为 `/interact/uhome/loadUserCollection`，本次在 Go 后端补齐同名能力，保持前端 `Api.uHomeLoadCollection` 不变。

## 实现概要

- 新增 `UserCollectionItem` 返回结构，包含收藏动作信息、收藏时间、视频封面、视频标题、作者信息和视频统计字段。
- 在 `InteractRepository` 中从 `user_action` 查询 `action_type=3` 的视频收藏记录，并左关联 `video_info`、`user_info` 获取视频和作者信息。
- 在 `UcenterContentService` 中新增 `LoadUserCollection`，负责校验 `userId`、归一化 `pageNo/pageSize`，并返回普通页码分页结构。
- 在 `/interact/uhome` 下注册 `GET/POST /loadUserCollection`，该接口允许游客访问。

## 接口变化

- 前端代理地址：`http://localhost:3000/api/interact/uhome/loadUserCollection`
- 后端路由：`GET/POST /interact/uhome/loadUserCollection`
- 登录要求：不需要登录
- 请求参数：
  - `userId`：必填，10 位用户 ID
  - `pageNo`：可选，默认 `1`
  - `pageSize`：可选，默认 `15`，最大 `50`
- 返回结构：`PaginationResult<UserCollectionItem>`
  - `totalCount`
  - `pageSize`
  - `pageNo`
  - `pageTotal`
  - `list`

## 数据库和缓存变化

- 不新增表，不修改表结构。
- 读取 `user_action`、`video_info`、`user_info`。
- 不新增 Redis、RabbitMQ、Elasticsearch 逻辑。
- 建议数据量增大后手动补充索引：

```sql
CREATE INDEX idx_user_action_collection_page
ON user_action (user_id, action_type, action_time, action_id);
```

## 验证方式

- `go test ./...`
- 打开 Web 端用户主页收藏页，确认能加载收藏视频列表。
- 未登录访问收藏页时不返回 `901`。
- 自己的收藏页点击“取消收藏”后，刷新列表确认该视频消失。

## 遗留风险

- 当前接口按页码分页，数据量极大时深分页仍可能变慢；后续可以改为游标分页。
- 如果被收藏的视频之后被删除，接口仍保留收藏动作记录并返回空视频标题，前端会显示“已失效视频”。
