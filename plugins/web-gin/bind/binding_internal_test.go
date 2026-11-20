package bind

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/gin-gonic/gin"
)

func TestHandleBindError_TypedNilFallbackWithoutBinder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	var nilAppErr *apperr.Error
	b := &Binding[struct{}]{Context: ctx}
	b.handleBindError(nilAppErr)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 ParameterError fallback when typed-nil *apperr.Error is passed, got %d (%s)", w.Code, w.Body.String())
	}
	if !ctx.IsAborted() {
		t.Fatal("expected gin.Context to be aborted")
	}
}
