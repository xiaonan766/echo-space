# 图库 Milvus 搜索结果解析修复

## 背景

图库文搜图中，`太阳` 的查询向量可以在 Milvus 中命中目标图片，原始相似度约为 `0.1955`，高于默认阈值 `0.15`，但 Web 接口仍返回空列表。

## 原因

Milvus Go SDK 的 Search 结果中，主键 `file_id` 主要位于 `ResultSet.IDs`，输出字段需要通过 `ResultSet.GetColumn(...)` 读取。原实现使用 `Unmarshal` 反序列化到结构体，导致 `file_id` 和 `image_id` 为空，后续 MySQL 回查找不到稿件，最终结果被误过滤。

## 修复

- `internal/infra/search/gallery_vector_index.go` 中的图库搜索结果解析改为逐行读取：
  - `file_id` 优先从 `GetColumn("file_id")` 读取，缺失时回退到 `ResultSet.IDs`。
  - `image_id` 从 `GetColumn("image_id")` 读取。
- 保留 Go 层按 `image_id` 去重、取最高分和最低分阈值过滤逻辑。

## 验证

- 使用 Redis 中 `太阳` 查询向量运行诊断，修复后命中：
  - `file_id=HUWyCMTJnvQflTPS5Zw9`
  - `image_id=mRJMHGnI9Z`
  - `score=0.19551226`
- 执行 `go test ./...` 通过。
