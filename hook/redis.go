package xHook

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisHook struct {
	Prefix string
}

// DialHook 是一个 Redis DialHook 的转发钩子方法，无额外处理逻辑，仅直接调用下一钩子。
//
// 参数说明:
//   - next: 下一个 `redis.DialHook` 处理函数。
//
// 返回值:
//   - `redis.DialHook`: 直接返回传入的 `next` 钩子函数，无额外逻辑。
func (rh RedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook 为 Redis 命令添加前缀处理逻辑的钩子函数。
//
// 该方法拦截 Redis 请求，对指定命令的关键键值添加自定义前缀（`rh.prefix`），再调用下一个钩子函数。
//
// 参数说明:
//   - next: 下一个 `redis.ProcessHook` 处理函数。
//
// 返回值:
//   - `redis.ProcessHook`: 带有自定义前缀处理逻辑的钩子函数。
func (rh RedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		if len(args) < 2 {
			return next(ctx, cmd)
		}

		switch cmd.Name() {
		case "set", "get", "del", "expire", "ttl", "sadd", "hset", "hget":
			key := args[1]
			if keyStr, ok := key.(string); ok {
				args[1] = rh.Prefix + ":" + keyStr
			}
		case "mget", "mset":
			for i := 1; i < len(args); i++ {
				key := args[i]
				if keyStr, ok := key.(string); ok {
					args[i] = rh.Prefix + ":" + keyStr
				}
			}
		}

		return next(ctx, cmd)
	}
}

// ProcessPipelineHook 为 Redis 管道请求添加前缀处理逻辑。
//
// 该方法拦截并修改每个 Redis 命令的关键键值，为其添加自定义前缀（`rh.prefix`）。
// 修改后的命令再传递给下一个处理器。
//
// 参数说明:
//   - next: 下一个 `redis.ProcessPipelineHook` 处理函数。
//
// 返回值:
//   - `redis.ProcessPipelineHook`: 带有自定义前缀处理逻辑的管道钩子函数。
func (rh RedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for i, cmd := range cmds {
			args := cmd.Args()
			if len(args) < 2 {
				cmds[i] = cmd
				continue
			}

			switch cmd.Name() {
			case "set", "get", "del", "expire", "ttl", "sadd", "hset", "hget":
				key := args[1]
				if keyStr, ok := key.(string); ok {
					args[1] = rh.Prefix + ":" + keyStr
				}
			case "mget", "mset":
				for i := 1; i < len(args); i++ {
					key := args[i]
					if keyStr, ok := key.(string); ok {
						args[i] = rh.Prefix + ":" + keyStr
					}
				}
			}
			cmds[i] = cmd
		}
		return next(ctx, cmds)
	}
}
