package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
)

func TestPostVideoRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/web/ucenter/postVideo", strings.NewReader(""))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var result response.VO
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeLoginTimeout {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeLoginTimeout)
	}
}
