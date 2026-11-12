package dialect

import (
	"errors"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"gorm.io/gorm"
	"testing"
)

var (
	_ func(Driver) (Adapter, *apperr.Error)         = Lookup
	_ func(Adapter) (gorm.Dialector, *apperr.Error) = Adapter.Open
	_ func(Driver) *apperr.Error                    = validateDriver
)

func TestP4RegistryErrors(t *testing.T) {
	r := newRegistry()
	_, err := r.lookup(Driver("p4-missing"))
	if err == nil || err.Code.Code != 50034 || !errors.Is(err, ErrDriverNotLoaded) {
		t.Fatalf("missing driver: %v", err)
	}
	r.register(Driver("p4-nil"), func() Adapter { return nil })
	_, err = r.lookup(Driver("p4-nil"))
	if err == nil || err.Code.Code != 50035 || !errors.Is(err, ErrNilAdapter) {
		t.Fatalf("nil adapter: %v", err)
	}
}

func TestP4DeclarationPanicsStayStructured(t *testing.T) {
	cases := []struct {
		name string
		run  func()
		code int
	}{
		{"blank", func() { newRegistry().register(" ", func() Adapter { return nil }) }, 50033},
		{"none", func() { newRegistry().register("none", func() Adapter { return nil }) }, 50033},
		{"nil factory", func() { newRegistry().register("p4", nil) }, 50035},
		{"duplicate", func() {
			r := newRegistry()
			f := func() Adapter { return nil }
			r.register("p4", f)
			r.register("p4", f)
		}, 50033},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				v := recover()
				e, ok := v.(*apperr.Error)
				if !ok || e == nil || e.Code.Code != apperr.Code(tc.code) {
					t.Fatalf("unexpected panic: %#v", v)
				}
			}()
			tc.run()
		})
	}
}
