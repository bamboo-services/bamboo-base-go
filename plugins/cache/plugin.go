package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/memory"
	goredis "github.com/redis/go-redis/v9"
)

// PluginKey 声明缓存插件全局唯一标识。
const PluginKey ctxkey.Key = "plugin_cache"

// pluginImpl 实现 application.Plugin 接口。
type pluginImpl struct {
	cfg     Config
	mode    Type
	initErr error

	mu       sync.Mutex
	started  bool
	memStore *memory.Store
	rdb      *goredis.Client
	mgr      *Manager

	stopOnce sync.Once
	stopErr  error
}

// New 构造缓存插件元数据与工厂函数，作为 [application.ModeTool] 受管组件接入应用运行时。
//
// 选项应用与模式冻结在 New 执行期完成：
//   - Memory 模式声明 Provides [ctxkey.CacheManagerKey]；
//   - Redis 模式声明 Provides [ctxkey.CacheManagerKey] 与 [ctxkey.RedisClientKey]；
//   - None 或未知模式将配置错误记录于工厂实例，并在生命周期 Start 阶段显式返回错误，
//     绝不伪称向容器提供未就绪的依赖键。
func New(opts ...Option) (application.PluginInfo, func() application.Plugin) {
	var cfg Config
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	var initErr error
	if cfg.validationErr != nil {
		initErr = cfg.validationErr
	}

	mode := cfg.TypeVal
	var provides []ctxkey.Key

	switch mode {
	case CacheTypeMemory:
		provides = []ctxkey.Key{ctxkey.CacheManagerKey}
	case CacheTypeRedis:
		provides = []ctxkey.Key{ctxkey.CacheManagerKey, ctxkey.RedisClientKey}
	default:
		// None 或 未知模式：不伪称提供 Manager 或 Redis
		provides = nil
		if initErr == nil {
			if mode == CacheTypeNone || mode == "" {
				initErr = errors.New("cache plugin: no cache backend selected (CacheTypeNone)")
			} else {
				initErr = fmt.Errorf("cache plugin: unknown cache backend type %q", mode)
			}
		}
	}

	info := application.PluginInfo{
		Key:      PluginKey,
		Name:     "plugin_cache",
		Mode:     application.ModeTool,
		Provides: provides,
	}

	factory := func() application.Plugin {
		return &pluginImpl{
			cfg:     cfg,
			mode:    mode,
			initErr: initErr,
		}
	}

	return info, factory
}

// Start 启动插件生命周期，按需建立后端连接或内存存储。
func (p *pluginImpl) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initErr != nil {
		return p.initErr
	}

	var logNamed *logging.LogNamedLogger
	if p.cfg.Logger != nil {
		logNamed = p.cfg.Logger.WithName("CACH")
	}

	switch p.mode {
	case CacheTypeMemory:
		ttl := p.cfg.Memory.DefaultTTL
		store := memory.NewStore(p.cfg.Memory.ShardCount, p.cfg.Memory.MaxEntries, ttl)
		p.memStore = store

		var mgrOpts []ManagerOption
		mgrOpts = append(mgrOpts, WithMemoryStore(store), WithManagerTTL(ttl))
		if logNamed != nil {
			mgrOpts = append(mgrOpts, WithManagerLogger(logNamed))
		}
		p.mgr = NewManager(CacheTypeMemory, mgrOpts...)
		p.started = true
		return nil

	case CacheTypeRedis:
		rOpts := &goredis.Options{
			Addr:         p.cfg.Redis.Addr,
			Username:     p.cfg.Redis.Username,
			Password:     p.cfg.Redis.Password,
			DB:           p.cfg.Redis.DB,
			PoolSize:     p.cfg.Redis.PoolSize,
			MinIdleConns: p.cfg.Redis.MinIdleConns,
			DialTimeout:  p.cfg.Redis.DialTimeout,
			ReadTimeout:  p.cfg.Redis.ReadTimeout,
			WriteTimeout: p.cfg.Redis.WriteTimeout,
		}
		client := goredis.NewClient(rOpts)
		p.rdb = client

		pingCtx := ctx
		if pingCtx == nil {
			pingCtx = context.Background()
		}
		if err := client.Ping(pingCtx).Err(); err != nil {
			// 初始化部分失败也要能清理，并保存关闭可能产生的错误
			closeErr := client.Close()
			p.rdb = nil
			if closeErr != nil {
				return errors.Join(err, closeErr)
			}
			return err
		}

		var mgrOpts []ManagerOption
		mgrOpts = append(mgrOpts, WithRedisClient(client))
		if logNamed != nil {
			mgrOpts = append(mgrOpts, WithManagerLogger(logNamed))
		}
		p.mgr = NewManager(CacheTypeRedis, mgrOpts...)
		p.started = true
		return nil

	default:
		return fmt.Errorf("cache plugin: unsupported cache backend type %q", p.mode)
	}
}

// Stop 停止缓存插件，释放底层资源。重复调用共享结果与等待，不因 ctx.Done 提前返回假排空。
func (p *pluginImpl) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		store := p.memStore
		rdb := p.rdb
		mgr := p.mgr
		p.memStore = nil
		p.rdb = nil
		p.mgr = nil
		p.mu.Unlock()

		var errs []error

		if store != nil {
			store.Close()
		}
		if mgr != nil {
			mgr.Close()
		}
		if rdb != nil {
			if err := rdb.Close(); err != nil {
				errs = append(errs, err)
			}
		}

		p.stopErr = errors.Join(errs...)
	})

	return p.stopErr
}

// Expose 将真实实例暴露为只读 map。
func (p *pluginImpl) Expose() map[ctxkey.Key]any {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make(map[ctxkey.Key]any)
	if p.mgr != nil {
		out[ctxkey.CacheManagerKey] = p.mgr
	}
	if p.mode == CacheTypeRedis && p.rdb != nil {
		out[ctxkey.RedisClientKey] = p.rdb
	}
	return out
}
