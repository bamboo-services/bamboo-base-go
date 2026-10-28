package dialect

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

type dummyAdapter struct {
	name string
}

func (d *dummyAdapter) Name() string {
	return d.name
}

func (d *dummyAdapter) Open() (gorm.Dialector, error) {
	return nil, nil
}

func TestRegistry_NormalAndUnknown(t *testing.T) {
	reg := newRegistry()

	reg.register(DriverMySQL, func() Adapter {
		return &dummyAdapter{name: "mysql"}
	})

	adapter, err := reg.lookup(DriverMySQL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter == nil || adapter.Name() != "mysql" {
		t.Fatalf("expected adapter named mysql, got %v", adapter)
	}

	_, err = reg.lookup(DriverPostgres)
	if err == nil || !errors.Is(err, ErrDriverNotLoaded) {
		t.Fatalf("expected ErrDriverNotLoaded, got %v", err)
	}
}

func TestRegistry_TypedNilAndNil(t *testing.T) {
	reg := newRegistry()

	reg.register(DriverSQLite, func() Adapter {
		return nil
	})

	_, err := reg.lookup(DriverSQLite)
	if err == nil || !errors.Is(err, ErrNilAdapter) {
		t.Fatalf("expected ErrNilAdapter for nil return, got %v", err)
	}

	reg.register(DriverOracle, func() Adapter {
		var d *dummyAdapter
		return d // typed nil
	})

	_, err = reg.lookup(DriverOracle)
	if err == nil || !errors.Is(err, ErrNilAdapter) {
		t.Fatalf("expected ErrNilAdapter for typed nil return, got %v", err)
	}
}

func TestRegistry_Panics(t *testing.T) {
	cases := []struct {
		name    string
		driver  Driver
		factory Factory
	}{
		{
			name:    "empty driver",
			driver:  "",
			factory: func() Adapter { return &dummyAdapter{name: "x"} },
		},
		{
			name:    "whitespace driver",
			driver:  "   ",
			factory: func() Adapter { return &dummyAdapter{name: "x"} },
		},
		{
			name:    "none driver lowercase",
			driver:  "none",
			factory: func() Adapter { return &dummyAdapter{name: "x"} },
		},
		{
			name:    "none driver uppercase",
			driver:  "NONE",
			factory: func() Adapter { return &dummyAdapter{name: "x"} },
		},
		{
			name:    "nil factory",
			driver:  DriverMySQL,
			factory: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := newRegistry()
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic for case %s, but did not panic", tc.name)
				}
			}()
			reg.register(tc.driver, tc.factory)
		})
	}

	t.Run("duplicate driver registration", func(t *testing.T) {
		reg := newRegistry()
		reg.register(DriverMySQL, func() Adapter { return &dummyAdapter{name: "mysql"} })
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic on duplicate registration")
			}
		}()
		reg.register(DriverMySQL, func() Adapter { return &dummyAdapter{name: "mysql2"} })
	})
}

func TestRegistry_Concurrency(t *testing.T) {
	reg := newRegistry()
	var wg sync.WaitGroup
	workers := 16
	iterations := 100

	for i := 0; i < workers; i++ {
		driver := Driver(fmt.Sprintf("driver_%d", i))
		reg.register(driver, func() Adapter {
			return &dummyAdapter{name: string(driver)}
		})
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				driver := Driver(fmt.Sprintf("driver_%d", j%workers))
				adapter, err := reg.lookup(driver)
				if err != nil {
					t.Errorf("lookup failed: %v", err)
					return
				}
				if adapter.Name() != string(driver) {
					t.Errorf("expected %s, got %s", driver, adapter.Name())
					return
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestRegistry_DoesNotHoldLockDuringFactoryCall(t *testing.T) {
	reg := newRegistry()

	nestedDriver := Driver("nested_driver_during_factory")
	done := make(chan struct{})

	reg.register(DriverSQLServer, func() Adapter {
		// 向同一 registry 注册另一个唯一 driver（写锁操作）。
		// 若 factory 调用时持有锁（无论是写锁还是读锁），此处 register 都会因无法获取写锁而死锁。
		reg.register(nestedDriver, func() Adapter {
			return &dummyAdapter{name: string(nestedDriver)}
		})
		return &dummyAdapter{name: "sqlserver"}
	})

	go func() {
		defer close(done)
		adapter, err := reg.lookup(DriverSQLServer)
		if err != nil {
			t.Errorf("unexpected error on primary lookup: %v", err)
			return
		}
		if adapter.Name() != "sqlserver" {
			t.Errorf("expected sqlserver, got %s", adapter.Name())
			return
		}

		nestedAdapter, err := reg.lookup(nestedDriver)
		if err != nil {
			t.Errorf("unexpected error on nested lookup: %v", err)
			return
		}
		if nestedAdapter.Name() != string(nestedDriver) {
			t.Errorf("expected %s, got %s", nestedDriver, nestedAdapter.Name())
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected: factory call holds registry lock preventing nested registration")
	}
}

var customPkgTestSeq uint64

func TestPackageLevelRegisterAndLookup(t *testing.T) {
	id := atomic.AddUint64(&customPkgTestSeq, 1)
	customDriver := Driver(fmt.Sprintf("custom_pkg_test_%d", id))
	Register(customDriver, func() Adapter {
		return &dummyAdapter{name: string(customDriver)}
	})

	adapter, err := Lookup(customDriver)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if adapter.Name() != string(customDriver) {
		t.Fatalf("expected %s, got %s", customDriver, adapter.Name())
	}
}
