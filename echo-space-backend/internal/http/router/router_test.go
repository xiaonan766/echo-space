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

func TestLoadDanmuRouteDoesNotRequireLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/interact/danmu/loadDanmu", strings.NewReader("fileId=bad&videoId=bad"))
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
	if result.Code == response.CodeLoginTimeout {
		t.Fatalf("response code = %d, want non-login response", result.Code)
	}
	if result.Code != response.CodeBusinessFail {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeBusinessFail)
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

func TestUserActionRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/interact/userAction/doAction", strings.NewReader("videoId=Abc123Def4&actionType=2"))
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

func TestLoadCommentRouteDoesNotRequireLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/interact/comment/loadComment", strings.NewReader("videoId=bad"))
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
	if result.Code == response.CodeLoginTimeout {
		t.Fatalf("response code = %d, want non-login response", result.Code)
	}
	if result.Code != response.CodeBusinessFail {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeBusinessFail)
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

func TestWebUhomeFocusRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/web/uhome/focus", strings.NewReader("focusUserId=1000000002"))
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

func TestWebDynamicLoadCurrentUserInfoRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/web/dynamic/loadCurrentUserInfo", nil)
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

func TestWebDynamicLoadFollowUsersRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/web/dynamic/loadFollowUsers", nil)
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

func TestWebDynamicLoadFeedRouteRequiresLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/web/dynamic/loadFeed", strings.NewReader("pageSize=10"))
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

func TestWebUhomeGetUserInfoRouteDoesNotRequireLogin(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.Mode = "test"
	engine := New(Dependencies{Config: cfg})

	request := httptest.NewRequest(http.MethodPost, "/web/uhome/getUserInfo", strings.NewReader("userId=bad"))
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
	if result.Code == response.CodeLoginTimeout {
		t.Fatalf("response code = %d, want non-login response", result.Code)
	}
	if result.Code != response.CodeBusinessFail {
		t.Fatalf("response code = %d, want %d", result.Code, response.CodeBusinessFail)
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
