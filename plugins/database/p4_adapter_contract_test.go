package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	database "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"gorm.io/gorm"
)

func TestP4StartRejectsInvalidAdapterValues(t *testing.T) {
	var nilAdapter *fakeAdapter
	var nilDialector *fakeDialector
	for _, tc := range []struct {
		name     string
		adapter  dialect.Adapter
		nilCause bool
	}{
		{"typed nil adapter", nilAdapter, true},
		{"nil dialector", &fakeAdapter{name: "p4"}, false},
		{"typed nil dialector", &fakeAdapter{name: "p4", dialector: nilDialector}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, factory := database.New(database.WithAdapter(tc.adapter))
			p := factory()
			err := p.Start(context.Background())
			if err == nil || err.Code.Code != dialect.AdapterInvalid.Code {
				t.Fatalf("invalid value: %v", err)
			}
			if tc.nilCause && !errors.Is(err, dialect.ErrNilAdapter) {
				t.Fatal("lost nil adapter cause")
			}
			if stopErr := p.Stop(context.Background()); stopErr != nil {
				t.Fatalf("failed start stop: %v", stopErr)
			}
		})
	}
}

func TestP4StartPreservesAdapterConfigurationError(t *testing.T) {
	cause := errors.New("configuration sentinel")
	inner := apperr.New(dialect.AdapterConfigInvalid, "启动配置非法", cause)
	_, factory := database.New(database.WithAdapter(&fakeAdapter{name: "p4", openErr: inner}))
	err := factory().Start(context.Background())
	if err == nil || err.Code.Code != 50036 || !errors.Is(err, inner) || !errors.Is(err, cause) {
		t.Fatalf("lost structured cause: %v", err)
	}
}

var _ func(dialect.Adapter) (gorm.Dialector, *apperr.Error) = dialect.Adapter.Open

func TestP4StartPreservesPingAndCleanupCauses(t *testing.T) {
	a, driver := newFakeAdapter("p4")
	pingCause := errors.New("ping cause")
	closeCause := errors.New("close cause")
	driver.pingErr = pingCause
	driver.closeErr = closeCause
	_, factory := database.New(database.WithAdapter(a))
	err := factory().Start(context.Background())
	if err == nil || err.Code.Code != apperr.FrameworkMultipleFailures.Code || !errors.Is(err, pingCause) || !errors.Is(err, closeCause) {
		t.Fatalf("startup or cleanup cause lost or unexpected aggregate code: %v", err)
	}
}
