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

func TestPostDanmuRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/interact/danmu/postDanmu", strings.NewReader("text=hello"))
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

func TestPostCommentRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/interact/comment/postComment", strings.NewReader("content=hello&videoId=Abc123Def4"))
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

func TestWebAutoLoginWithoutTokenReturnsSuccess(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/web/account/autoLogin", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var result response.VO
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != response.CodeSuccess {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeSuccess)
	}
	if result.Data != nil {
		t.Fatalf("response data = %#v, want nil", result.Data)
	}
}

func TestWebGetUserCountInfoRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/web/account/getUserCountInfo", nil)
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

func TestAdminAuditVideoRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/admin/videoInfo/auditVideo", strings.NewReader(""))
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

func TestAdminRecommendVideoRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/admin/videoInfo/recommendVideo", strings.NewReader("videoId=Abc123Def4"))
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

func TestAdminLoadVideoPListRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/admin/videoInfo/loadVideoPList", strings.NewReader("videoId=Abc123Def4"))
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

func TestAdminVideoResourceRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodGet, "/admin/file/videoResource/dQ1JEV4n9ZVqfOA7qm1I/", nil)
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
