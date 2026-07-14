package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashScopeEmbeddingRequest(t *testing.T) {
	var requests []dashScopeRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload dashScopeRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		requests = append(requests, payload)
		embeddings := make([]map[string]any, len(payload.Input.Contents))
		for index := range embeddings {
			embeddings[index] = map[string]any{"index": index, "type": "text", "embedding": make([]float32, 1152)}
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"output": map[string]any{"embeddings": embeddings}})
	}))
	defer server.Close()

	client, err := NewDashScopeClient("test-key", server.URL, "tongyi-embedding-vision-plus-2026-03-06", 1152, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.EmbedText(context.Background(), "舞台上的选手"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.EmbedImages(context.Background(), [][]byte{{1, 2}, {3, 4}}); err != nil {
		t.Fatal(err)
	}

	if len(requests) != 2 {
		t.Fatalf("request count = %d", len(requests))
	}
	for _, request := range requests {
		if request.Model != "tongyi-embedding-vision-plus-2026-03-06" || request.Parameters.Dimension != 1152 || request.Parameters.ResLevel != 1 || request.Parameters.OutputType != "dense" {
			t.Fatalf("unexpected request: %+v", request)
		}
	}
	if len(requests[1].Input.Contents) != 2 || !strings.HasPrefix(requests[1].Input.Contents[0]["image"], "data:image/jpeg;base64,") {
		t.Fatalf("images must use independent contents: %+v", requests[1].Input.Contents)
	}
}

func TestNormalizeImageToJPEG(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			source.Set(x, y, color.NRGBA{R: 255, A: uint8(x * 4)})
		}
	}
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatal(err)
	}
	output, err := NormalizeImage(input.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(output) < 2 || output[0] != 0xff || output[1] != 0xd8 {
		t.Fatal("output is not JPEG")
	}
	if len(output) > MaxNormalizedImageBytes {
		t.Fatal("normalized image exceeds limit")
	}
}

func TestNormalizeImageRejectsInvalidAndOversizedInput(t *testing.T) {
	if _, err := NormalizeImage([]byte("not-image")); err == nil || !strings.Contains(err.Error(), "不支持") {
		t.Fatalf("invalid image error = %v, want unsupported format", err)
	}
	if _, err := NormalizeImage(make([]byte, MaxQueryImageBytes+1)); err == nil || !strings.Contains(err.Error(), "10MB") {
		t.Fatalf("oversized image error = %v, want size error", err)
	}
}
