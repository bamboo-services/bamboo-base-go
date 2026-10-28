package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/header"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/middleware"
	"github.com/gin-gonic/gin"
)

func parseHeaderTokens(value string) map[string]struct{} {
	tokens := make(map[string]struct{})
	for _, part := range strings.Split(value, ",") {
		token := strings.TrimSpace(part)
		if token != "" {
			tokens[token] = struct{}{}
		}
	}
	return tokens
}

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectHandler  bool
	}{
		{
			name:           "GET_Request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectHandler:  true,
		},
		{
			name:           "OPTIONS_Request",
			method:         http.MethodOptions,
			expectedStatus: http.StatusNoContent,
			expectHandler:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(middleware.CORS())

			handlerCalled := false
			engine.Any("/test", func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("unexpected status code: got %d, want %d", w.Code, tt.expectedStatus)
			}

			if handlerCalled != tt.expectHandler {
				t.Fatalf("unexpected handler invocation: got %v, want %v", handlerCalled, tt.expectHandler)
			}

			exposeHeader := w.Header().Get("Access-Control-Expose-Headers")
			tokens := parseHeaderTokens(exposeHeader)

			expectedUUID := header.HeaderRequestUUID.String()
			if _, ok := tokens[expectedUUID]; !ok {
				t.Fatalf("Access-Control-Expose-Headers missing expected token %q, got: %q", expectedUUID, exposeHeader)
			}

			if _, ok := tokens["X-Request-Id"]; ok {
				t.Fatalf("Access-Control-Expose-Headers should not contain misleading 'X-Request-Id', got: %q", exposeHeader)
			}

			if _, ok := tokens["Content-Length"]; !ok {
				t.Fatalf("Access-Control-Expose-Headers missing 'Content-Length', got: %q", exposeHeader)
			}
		})
	}
}
