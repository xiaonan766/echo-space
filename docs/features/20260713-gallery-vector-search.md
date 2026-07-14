# 图库 Milvus + 阿里云百炼多模态搜索

## 需求背景

图库原先只有普通分页浏览，顶部搜索仍复用视频 Elasticsearch，无法按图片画面语义检索。本次为 `/gallery` 新增独立向量搜索，支持游客以中文描述搜图、上传图片搜相似图片；一个稿件包含多张正文图时，任意一张满足阈值即可命中，稿件只返回一次并展示最相似附件。

## 实现概要

- 使用阿里云百炼 `tongyi-embedding-vision-plus-2026-03-06`，固定 1152 维、`res_level=1`、dense 输出。文本和每张图片分别请求独立向量，不生成融合向量。
- 每张审核通过、传输成功、未删除的正文图片在 Milvus 保存一条记录。使用 `image_id` Grouping Search，每组保留最高分附件。
- 查询图片只在内存中读取、解码并转成 JPEG，不写磁盘；原图最大 10MB，压缩结果最大 3MB。GIF 使用 Go 解码得到的首帧，透明区域填白。
- 查询向量在 Redis 缓存 10 分钟，后续分页使用不透明 `searchToken`，不重复请求百炼。
- 图片稿件审核后立即排队同步；应用启动和每 10 分钟进行一次单并发对账。Milvus 临时残留的稿件在返回前还会回查 MySQL 审核状态与附件状态。
- 百炼或 Milvus 未配置/不可用时，仅 `/web/gallery/search` 返回“图库搜索服务暂不可用”，普通图库和视频 ES 搜索不受影响。

## 核心改动

- 后端基础设施：`internal/infra/embedding`、`internal/infra/search/gallery_vector_index.go`、`internal/infra/cache/gallery_search_vector_store.go`。
- 后端业务：`internal/service/gallerysearch/service.go`，图库 repository、web handler/service/router，管理员审核同步入口和应用组装。
- Web 页面：`src/views/gallery/Gallery.vue`、`GalleryImageItem.vue`、`GalleryDetail.vue`、`views/layout/LayoutHeader.vue`，以及 `Api.js`、`Request.js`。图库页不再保留正文内搜索框，搜索入口统一收敛到顶部搜索栏；顶部搜索栏左侧提供“文 / 图”切换，文本模式走 URL query，图片模式通过页面级事件把用户选择的文件交给图库页发起检索。
- 部署：`deploy/milvus/docker-compose.yml`。

## 接口变化

新增游客接口 `POST /web/gallery/search`。

- 首次文搜图：`searchType=text`、`keyword`、`pageNo`、`pageSize`。
- 首次图搜图：multipart `searchType=image`、`file`、`pageNo`、`pageSize`。
- 后续分页：`searchToken`、`pageNo`、`pageSize`。
- 返回 `searchToken/searchType/pageNo/pageSize/hasMore/list`。列表额外包含 `matchedImage`，不返回无法准确计算的 ANN 总数。

## Milvus Schema

Collection 默认为 `echo_space_gallery_image_v1`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `file_id` | VARCHAR(64) | 主键，附件幂等 upsert |
| `image_id` | VARCHAR(64) | 稿件分组字段 |
| `content_version` | INT64 | 稿件最后更新时间戳 |
| `embedding_model` | VARCHAR(128) | 模型版本，用于判断是否重建 |
| `embedding` | FLOAT_VECTOR(1152) | 百炼独立图片向量 |

索引为 HNSW + COSINE，`M=16`、`efConstruction=200`，查询 `ef=64`，默认最低分 `0.15`。

## 配置与部署

1. 在根目录执行 `docker compose -f deploy/milvus/docker-compose.yml up -d`。
2. 设置 `DASHSCOPE_API_KEY`。密钥禁止写入 YAML 或 Git。
3. 可选配置：`MILVUS_ADDRESS`、`MILVUS_TOKEN`、`MILVUS_COLLECTION`、`DASHSCOPE_BASE_URL`、`GALLERY_SEARCH_MIN_SCORE`、`GALLERY_SEARCH_VECTOR_TTL`、`GALLERY_SEARCH_RECONCILE_INTERVAL`。
4. 启动后端。首次对账会读取历史审核通过正文图片并发送给百炼，然后写入 Milvus。

## 数据、缓存及外部服务变化

- MySQL：不新增表、不修改字段。
- Redis：新增 `echo-space:gallery:search-vector:<token>`，默认 TTL 10 分钟，只保存查询类型和 1152 维向量。
- RabbitMQ：不新增队列。
- Elasticsearch：图库搜索不使用 ES，现有视频索引逻辑不变。
- 数据出站：审核通过的图库正文图片和用户主动选择的查询图片，会以处理后的 JPEG Data URI 发送到阿里云百炼。本站不保存查询原图，但应在隐私政策中明确第三方处理。

## 验证方式

- 后端：`cd echo-space-backend && go test ./...`。
- Web：`cd echo-space-frontend-web && npm run build`。
- 手工验证中文文搜图、图搜图、多图任一命中、加载更多、刷新分享文本搜索、清除搜索、token 过期，以及停用百炼/Milvus 时普通图库仍可浏览。

## 风险与调优

- 百炼按调用/Token 或图片处理量计费，首次历史回填会产生集中费用；`content_version` 与模型字段可避免每轮重复向量化。
- `0.15` 只是首版阈值，需要根据真实图库标注样本统计误召回和漏召回后调整。只改阈值无需重建；更换模型或维度必须使用新 collection 全量回填。
- 当前不新增持久任务表，失败依靠日志和下轮对账修复；大规模图库可后续升级为 Outbox/MQ。
- Milvus 对精确总数不保证，因此使用 `hasMore`；极端陈旧向量会在 MySQL 回查时被过滤，随后由周期对账清理。
## 2026-07-14 修复记录

- 修复 Milvus Grouping Search 与 range search 不能同时使用的问题。
- 图库搜索保留 `image_id` 分组能力，最低分阈值改为在 Go 层过滤，不再传递 `radius/range_filter` 给 Milvus。
- 进一步改为先做普通 ANN 检索，再在 Go 层按 `image_id` 去重取最高分，规避 Milvus group search 在本地环境中的兼容性问题。
- 该修复不需要重建 collection，也不影响百炼模型、Redis 查询缓存或历史索引数据。
