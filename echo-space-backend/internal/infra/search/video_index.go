package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const (
	defaultVideoIndexName = "echo_space_video"
	videoSearchTimeout    = 3 * time.Second
)

type VideoIndex struct {
	client *elasticsearch.Client
	index  string
}

type VideoSearchInput struct {
	Keyword   string
	OrderType *int
	PageNo    int
	PageSize  int
	Highlight bool
}

type VideoSearchResult struct {
	TotalCount int64
	Hits       []VideoSearchHit
}

type VideoSearchHit struct {
	VideoID       string
	HighlightName string
}

func NewVideoIndex(cfg config.ElasticsearchConfig) (*VideoIndex, error) {
	indexName := strings.TrimSpace(cfg.IndexVideoName)
	if indexName == "" {
		indexName = defaultVideoIndexName
	}

	addresses := normalizeAddresses(cfg.Addresses)
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: addresses,
		Username:  strings.TrimSpace(cfg.Username),
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, err
	}

	return &VideoIndex{
		client: client,
		index:  indexName,
	}, nil
}

func (v *VideoIndex) EnsureVideoIndex(ctx context.Context) error {
	if v == nil || v.client == nil {
		return errors.New("video search index is not ready")
	}

	ctx, cancel := context.WithTimeout(ctx, videoSearchTimeout)
	defer cancel()

	response, err := v.client.Indices.Exists([]string{v.index}, v.client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	defer closeResponse(response)

	if response.StatusCode == 200 {
		return nil
	}
	if response.StatusCode != 404 {
		return responseError("check video index", response)
	}

	createResponse, err := v.client.Indices.Create(
		v.index,
		v.client.Indices.Create.WithContext(ctx),
		v.client.Indices.Create.WithBody(strings.NewReader(videoIndexMapping)),
	)
	if err != nil {
		return err
	}
	defer closeResponse(createResponse)
	if createResponse.IsError() {
		return responseError("create video index", createResponse)
	}
	return nil
}

func (v *VideoIndex) IndexVideo(ctx context.Context, document domain.VideoSearchDocument) error {
	if v == nil || v.client == nil {
		return errors.New("video search index is not ready")
	}
	document.VideoID = strings.TrimSpace(document.VideoID)
	if document.VideoID == "" {
		return errors.New("video id is empty")
	}

	content, err := json.Marshal(document)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, videoSearchTimeout)
	defer cancel()

	response, err := v.client.Index(
		v.index,
		bytes.NewReader(content),
		v.client.Index.WithContext(ctx),
		v.client.Index.WithDocumentID(document.VideoID),
	)
	if err != nil {
		return err
	}
	defer closeResponse(response)
	if response.IsError() {
		return responseError("index video document", response)
	}
	return nil
}

func (v *VideoIndex) Search(ctx context.Context, input VideoSearchInput) (VideoSearchResult, error) {
	if v == nil || v.client == nil {
		return VideoSearchResult{}, errors.New("video search index is not ready")
	}

	body, err := buildVideoSearchBody(input)
	if err != nil {
		return VideoSearchResult{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, videoSearchTimeout)
	defer cancel()

	response, err := v.client.Search(
		v.client.Search.WithContext(ctx),
		v.client.Search.WithIndex(v.index),
		v.client.Search.WithBody(bytes.NewReader(body)),
		v.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return VideoSearchResult{}, err
	}
	defer closeResponse(response)
	if response.IsError() {
		return VideoSearchResult{}, responseError("search video document", response)
	}

	var result elasticVideoSearchResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return VideoSearchResult{}, err
	}

	hits := make([]VideoSearchHit, 0, len(result.Hits.Hits))
	for _, item := range result.Hits.Hits {
		videoID := strings.TrimSpace(item.Source.VideoID)
		if videoID == "" {
			videoID = strings.TrimSpace(item.ID)
		}
		if videoID == "" {
			continue
		}
		hits = append(hits, VideoSearchHit{
			VideoID:       videoID,
			HighlightName: firstHighlight(item.Highlight["videoName"]),
		})
	}

	return VideoSearchResult{
		TotalCount: result.Hits.Total.Value,
		Hits:       hits,
	}, nil
}

func SearchOrderField(orderType int) (string, bool) {
	switch orderType {
	case 0:
		return "playCount", true
	case 1:
		return "createTime", true
	case 2:
		return "danmuCount", true
	case 3:
		return "collectCount", true
	default:
		return "", false
	}
}

func buildVideoSearchBody(input VideoSearchInput) ([]byte, error) {
	keyword := strings.TrimSpace(input.Keyword)
	if keyword == "" {
		return nil, errors.New("keyword is empty")
	}

	sortList := make([]map[string]map[string]string, 0, 2)
	if input.OrderType != nil {
		if field, ok := SearchOrderField(*input.OrderType); ok {
			sortList = append(sortList, map[string]map[string]string{
				field: map[string]string{"order": "desc"},
			})
		}
	}
	sortList = append(sortList, map[string]map[string]string{
		"_score": map[string]string{"order": "desc"},
	})

	body := map[string]any{
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  keyword,
				"fields": []string{"videoName", "tags"},
			},
		},
		"sort":             sortList,
		"track_total_hits": true,
	}
	if input.Highlight {
		body["highlight"] = map[string]any{
			"pre_tags":  []string{"<span class='highlight'>"},
			"post_tags": []string{"</span>"},
			"encoder":   "html",
			"fields": map[string]any{
				"videoName": map[string]any{},
			},
		}
	}
	return json.Marshal(body)
}

func normalizeAddresses(addresses []string) []string {
	result := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if item := strings.TrimSpace(address); item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		result = append(result, "http://localhost:9200")
	}
	return result
}

func firstHighlight(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func responseError(action string, response *esapi.Response) error {
	if response == nil {
		return fmt.Errorf("%s failed", action)
	}
	content, _ := io.ReadAll(response.Body)
	return fmt.Errorf("%s failed: status=%s body=%s", action, response.Status(), strings.TrimSpace(string(content)))
}

func closeResponse(response *esapi.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
}

type elasticVideoSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID        string              `json:"_id"`
			Source    videoSearchSource   `json:"_source"`
			Highlight map[string][]string `json:"highlight"`
		} `json:"hits"`
	} `json:"hits"`
}

type videoSearchSource struct {
	VideoID string `json:"videoId"`
}

const videoIndexMapping = `{
  "settings": {
    "analysis": {
      "analyzer": {
        "comma": {
          "type": "pattern",
          "pattern": ","
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "videoId": {
        "type": "keyword"
      },
      "userId": {
        "type": "keyword",
        "index": false
      },
      "videoCover": {
        "type": "keyword",
        "index": false
      },
      "videoName": {
        "type": "text",
        "analyzer": "ik_max_word"
      },
      "tags": {
        "type": "text",
        "analyzer": "comma"
      },
      "playCount": {
        "type": "integer"
      },
      "danmuCount": {
        "type": "integer"
      },
      "collectCount": {
        "type": "integer"
      },
      "createTime": {
        "type": "date",
        "format": "yyyy-MM-dd HH:mm:ss"
      }
    }
  }
}`
