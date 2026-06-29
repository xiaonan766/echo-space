package search

import (
	"encoding/json"
	"testing"
)

func TestSearchOrderField(t *testing.T) {
	tests := []struct {
		orderType int
		wantField string
		wantOK    bool
	}{
		{orderType: 0, wantField: "playCount", wantOK: true},
		{orderType: 1, wantField: "createTime", wantOK: true},
		{orderType: 2, wantField: "danmuCount", wantOK: true},
		{orderType: 3, wantField: "collectCount", wantOK: true},
		{orderType: 9, wantOK: false},
	}

	for _, tt := range tests {
		field, ok := SearchOrderField(tt.orderType)
		if field != tt.wantField || ok != tt.wantOK {
			t.Fatalf("SearchOrderField(%d) = (%q, %t), want (%q, %t)", tt.orderType, field, ok, tt.wantField, tt.wantOK)
		}
	}
}

func TestBuildVideoSearchBodyUsesFieldSortAndHighlight(t *testing.T) {
	orderType := 0
	content, err := buildVideoSearchBody(VideoSearchInput{
		Keyword:   "vue",
		OrderType: &orderType,
		Highlight: true,
	})
	if err != nil {
		t.Fatalf("buildVideoSearchBody returned error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(content, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	sortList, ok := body["sort"].([]any)
	if !ok || len(sortList) != 2 {
		t.Fatalf("sort = %#v, want two sort fields", body["sort"])
	}
	firstSort, ok := sortList[0].(map[string]any)
	if !ok {
		t.Fatalf("first sort = %#v, want map", sortList[0])
	}
	if _, ok := firstSort["playCount"]; !ok {
		t.Fatalf("first sort = %#v, want playCount sort", firstSort)
	}

	highlight, ok := body["highlight"].(map[string]any)
	if !ok {
		t.Fatalf("highlight = %#v, want map", body["highlight"])
	}
	if highlight["encoder"] != "html" {
		t.Fatalf("highlight encoder = %#v, want html", highlight["encoder"])
	}
}

func TestBuildVideoSearchBodyRejectsEmptyKeyword(t *testing.T) {
	if _, err := buildVideoSearchBody(VideoSearchInput{Keyword: "   "}); err == nil {
		t.Fatal("expected error for empty keyword")
	}
}
