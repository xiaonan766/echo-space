package search

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

const (
	galleryEmbeddingField = "embedding"
	galleryImageIDField   = "image_id"
)

type GalleryVectorRecord struct {
	FileID         string
	ImageID        string
	ContentVersion int64
	EmbeddingModel string
	Embedding      []float32
}

type GalleryVectorMetadata struct {
	FileID         string `milvus:"file_id"`
	ImageID        string `milvus:"image_id"`
	ContentVersion int64  `milvus:"content_version"`
	EmbeddingModel string `milvus:"embedding_model"`
}

type GalleryVectorHit struct {
	FileID  string
	ImageID string
	Score   float32
}

type GalleryVectorIndex struct {
	client     *milvusclient.Client
	collection string
	dimension  int
}

func NewGalleryVectorIndex(ctx context.Context, address, token, collection string, dimension int) (*GalleryVectorIndex, error) {
	if strings.TrimSpace(address) == "" || strings.TrimSpace(collection) == "" || dimension <= 0 {
		return nil, errors.New("milvus gallery search configuration is invalid")
	}
	client, err := milvusclient.New(ctx, &milvusclient.ClientConfig{Address: address, APIKey: token})
	if err != nil {
		return nil, fmt.Errorf("connect milvus: %w", err)
	}
	return &GalleryVectorIndex{client: client, collection: collection, dimension: dimension}, nil
}

func (i *GalleryVectorIndex) EnsureCollection(ctx context.Context) error {
	has, err := i.client.HasCollection(ctx, milvusclient.NewHasCollectionOption(i.collection))
	if err != nil {
		return fmt.Errorf("check gallery collection: %w", err)
	}
	if !has {
		schema := entity.NewSchema().WithDynamicFieldEnabled(false).
			WithField(entity.NewField().WithName("file_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(64).WithIsPrimaryKey(true)).
			WithField(entity.NewField().WithName(galleryImageIDField).WithDataType(entity.FieldTypeVarChar).WithMaxLength(64)).
			WithField(entity.NewField().WithName("content_version").WithDataType(entity.FieldTypeInt64)).
			WithField(entity.NewField().WithName("embedding_model").WithDataType(entity.FieldTypeVarChar).WithMaxLength(128)).
			WithField(entity.NewField().WithName(galleryEmbeddingField).WithDataType(entity.FieldTypeFloatVector).WithDim(int64(i.dimension)))
		vectorIndex := index.NewHNSWIndex(entity.COSINE, 16, 200)
		if err := i.client.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(i.collection, schema).
			WithIndexOptions(milvusclient.NewCreateIndexOption(i.collection, galleryEmbeddingField, vectorIndex))); err != nil {
			return fmt.Errorf("create gallery collection: %w", err)
		}
	} else {
		collection, err := i.client.DescribeCollection(ctx, milvusclient.NewDescribeCollectionOption(i.collection))
		if err != nil {
			return fmt.Errorf("describe gallery collection: %w", err)
		}
		if collection == nil || collection.Schema == nil {
			return errors.New("gallery collection schema is unavailable")
		}
		var embeddingField *entity.Field
		for _, field := range collection.Schema.Fields {
			if field != nil && field.Name == galleryEmbeddingField {
				embeddingField = field
				break
			}
		}
		if embeddingField == nil {
			return errors.New("gallery collection embedding field is missing")
		}
		dimension, dimErr := embeddingField.GetDim()
		if dimErr != nil || dimension != int64(i.dimension) {
			return fmt.Errorf("gallery collection embedding dimension mismatch: want %d", i.dimension)
		}
	}
	loadTask, err := i.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(i.collection))
	if err != nil {
		return fmt.Errorf("load gallery collection: %w", err)
	}
	if err := loadTask.Await(ctx); err != nil {
		return fmt.Errorf("await gallery collection load: %w", err)
	}
	return nil
}

func (i *GalleryVectorIndex) Upsert(ctx context.Context, records []GalleryVectorRecord) error {
	if len(records) == 0 {
		return nil
	}
	fileIDs := make([]string, len(records))
	imageIDs := make([]string, len(records))
	versions := make([]int64, len(records))
	models := make([]string, len(records))
	vectors := make([][]float32, len(records))
	for offset, record := range records {
		if len(record.Embedding) != i.dimension {
			return errors.New("gallery vector dimension mismatch")
		}
		fileIDs[offset], imageIDs[offset], versions[offset], models[offset], vectors[offset] = record.FileID, record.ImageID, record.ContentVersion, record.EmbeddingModel, record.Embedding
	}
	_, err := i.client.Upsert(ctx, milvusclient.NewColumnBasedInsertOption(i.collection).
		WithVarcharColumn("file_id", fileIDs).
		WithVarcharColumn(galleryImageIDField, imageIDs).
		WithInt64Column("content_version", versions).
		WithVarcharColumn("embedding_model", models).
		WithFloatVectorColumn(galleryEmbeddingField, i.dimension, vectors))
	return err
}

func (i *GalleryVectorIndex) DeleteFiles(ctx context.Context, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}
	_, err := i.client.Delete(ctx, milvusclient.NewDeleteOption(i.collection).WithStringIDs("file_id", fileIDs))
	return err
}

func (i *GalleryVectorIndex) MetadataByImageID(ctx context.Context, imageID string) ([]GalleryVectorMetadata, error) {
	result, err := i.client.Query(ctx, milvusclient.NewQueryOption(i.collection).
		WithFilter(fmt.Sprintf(`image_id == %q`, imageID)).
		WithOutputFields("file_id", galleryImageIDField, "content_version", "embedding_model"))
	if err != nil {
		return nil, err
	}
	var records []*GalleryVectorMetadata
	if err := result.Unmarshal(&records); err != nil {
		return nil, err
	}
	metadata := make([]GalleryVectorMetadata, 0, len(records))
	for _, record := range records {
		if record != nil {
			metadata = append(metadata, *record)
		}
	}
	return metadata, nil
}

func (i *GalleryVectorIndex) AllMetadata(ctx context.Context) ([]GalleryVectorMetadata, error) {
	const batchSize = 2048
	metadata := make([]GalleryVectorMetadata, 0)
	for offset := 0; ; offset += batchSize {
		result, err := i.client.Query(ctx, milvusclient.NewQueryOption(i.collection).
			WithFilter("file_id != \"\"").
			WithOutputFields("file_id", galleryImageIDField, "content_version", "embedding_model").
			WithOffset(offset).
			WithLimit(batchSize))
		if err != nil {
			return nil, err
		}
		var records []*GalleryVectorMetadata
		if err := result.Unmarshal(&records); err != nil {
			return nil, err
		}
		for _, record := range records {
			if record != nil {
				metadata = append(metadata, *record)
			}
		}
		if len(records) < batchSize {
			break
		}
	}
	return metadata, nil
}

func (i *GalleryVectorIndex) Search(ctx context.Context, vector []float32, offset, limit int, minScore float64) ([]GalleryVectorHit, error) {
	if len(vector) != i.dimension {
		return nil, errors.New("gallery query vector dimension mismatch")
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		return []GalleryVectorHit{}, nil
	}
	rawLimit := (offset + limit) * 8
	if rawLimit < limit*8 {
		rawLimit = limit * 8
	}
	if rawLimit < 100 {
		rawLimit = 100
	}
	if rawLimit > 1000 {
		rawLimit = 1000
	}
	annParam := index.NewCustomAnnParam()
	annParam.WithExtraParam("ef", 64)
	resultSets, err := i.client.Search(ctx, milvusclient.NewSearchOption(i.collection, rawLimit, []entity.Vector{entity.FloatVector(vector)}).
		WithANNSField(galleryEmbeddingField).
		WithOffset(0).
		WithOutputFields("file_id", galleryImageIDField).
		WithAnnParam(annParam))
	if err != nil {
		return nil, err
	}
	if len(resultSets) == 0 {
		return []GalleryVectorHit{}, nil
	}
	resultSet := resultSets[0]
	fileIDColumn := resultSet.GetColumn("file_id")
	imageIDColumn := resultSet.GetColumn(galleryImageIDField)
	hitsByImageID := make(map[string]GalleryVectorHit, resultSet.Len())
	for rowIndex := 0; rowIndex < resultSet.Len(); rowIndex++ {
		if rowIndex >= len(resultSet.Scores) {
			continue
		}
		score := resultSet.Scores[rowIndex]
		if score < float32(minScore) {
			continue
		}
		fileID := columnStringValue(fileIDColumn, rowIndex)
		if fileID == "" && resultSet.IDs != nil {
			fileID = columnStringValue(resultSet.IDs, rowIndex)
		}
		imageID := columnStringValue(imageIDColumn, rowIndex)
		if fileID == "" || imageID == "" {
			continue
		}
		hit, exists := hitsByImageID[imageID]
		if !exists || score > hit.Score {
			hitsByImageID[imageID] = GalleryVectorHit{
				FileID:  fileID,
				ImageID: imageID,
				Score:   score,
			}
		}
	}
	hits := make([]GalleryVectorHit, 0, len(hitsByImageID))
	for _, hit := range hitsByImageID {
		hits = append(hits, hit)
	}
	sort.SliceStable(hits, func(left, right int) bool { return hits[left].Score > hits[right].Score })
	pageOffset := offset
	if pageOffset >= len(hits) {
		return []GalleryVectorHit{}, nil
	}
	end := pageOffset + limit
	if end > len(hits) {
		end = len(hits)
	}
	hits = hits[pageOffset:end]
	return hits, nil
}

func columnStringValue(column interface {
	GetAsString(int) (string, error)
}, rowIndex int) string {
	if column == nil {
		return ""
	}
	value, err := column.GetAsString(rowIndex)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func (i *GalleryVectorIndex) Close(ctx context.Context) error {
	if i == nil || i.client == nil {
		return nil
	}
	return i.client.Close(ctx)
}
