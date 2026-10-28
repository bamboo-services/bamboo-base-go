package header_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/header"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHeaderConstantsAndMethods(t *testing.T) {
	if header.HeaderAuthorization.String() != "Authorization" {
		t.Fatalf("unexpected string: %s", header.HeaderAuthorization.String())
	}
	if header.Header("").IsEmpty() != true {
		t.Fatalf("expected empty")
	}
	if header.HeaderAuthorization.IsEmpty() {
		t.Fatalf("expected non-empty")
	}
	if !header.HeaderRequestUUID.IsCustom() {
		t.Fatalf("expected custom header")
	}
	if header.HeaderAuthorization.IsCustom() {
		t.Fatalf("expected non-custom header")
	}
}

func TestGetAuthorization(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// nil check
	if _, err := header.GetAuthorization(nil); err == nil {
		t.Fatalf("expected error on nil")
	}

	// Missing header
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := header.GetAuthorization(c); err == nil {
		t.Fatalf("expected error on missing authorization header")
	}

	// Malformed header
	c.Request.Header.Set("Authorization", "Basic 123456")
	if _, err := header.GetAuthorization(c); err == nil {
		t.Fatalf("expected error on malformed format")
	}

	// Valid Bearer
	c.Request.Header.Set("Authorization", "Bearer test-token-xyz")
	token, err := header.GetAuthorization(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-token-xyz" {
		t.Fatalf("expected token test-token-xyz, got %s", token)
	}
}

func TestGetToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	if header.GetToken(nil, header.HeaderAuthorization) != "" {
		t.Fatalf("expected empty on nil ctx")
	}

	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer auth-token")
	c.Request.Header.Set("X-Refresh-Token", "refresh-token")

	if token := header.GetToken(c, header.HeaderAuthorization); token != "auth-token" {
		t.Fatalf("expected auth-token, got %s", token)
	}
	if token := header.GetToken(c, header.HeaderRefreshToken); token != "refresh-token" {
		t.Fatalf("expected refresh-token, got %s", token)
	}
	// Fallback to default
	if token := header.GetToken(c, header.HeaderAccept); token != "auth-token" {
		t.Fatalf("expected fallback to auth-token, got %s", token)
	}
}
