package redis

import (
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/control"
	"github.com/bamboo-services/bamboo-base-go/plugins/cache/v2/driver"
	goredis "github.com/redis/go-redis/v9"
)

// errNotInitialized 是缓存实例不可用时的稳定原因。
var errNotInitialized = errors.New("redis: cache instance is not initialized")

// notInitializedErr 返回缓存实例不可用时的统一错误。
//
// 用于 nil 接收器或未注入 Redis 客户端的场景：调用方在错误路径上得到
// [driver.OperationFailed]（50039），而不是触发 panic。
func notInitializedErr() *apperr.Error {
	return apperr.New(driver.OperationFailed, "缓存实例未初始化", errNotInitialized)
}

// opErr 把 Redis 原生错误归一为框架统一错误。
//
// 归一顺序：
//   - 真正的 nil 与 goredis.Nil（键或成员不存在）都不是失败，直接返回 nil；
//   - context 取消与超时经 [control.Classify] 归一为控制类错误码
//     （49900 / 50401），原生 context 哨兵继续作为根因保留，
//     可经 errors.Is 精确判定；
//   - 其余 Redis 故障归一为 [driver.OperationFailed]（50039），
//     Redis 原生错误作为根因保留。
//
// 安全描述只来自受控错误码文案，不复制 Redis 原生错误文本，
// 避免把连接串等敏感内容带入对外文案。
func opErr(err error) *apperr.Error {
	if apperr.IsNil(err) || errors.Is(err, goredis.Nil) {
		return nil
	}
	if classified := control.Classify(err, control.PhaseOperation); classified != nil {
		return apperr.New(&classified.Code, "", err)
	}
	return apperr.New(driver.OperationFailed, "", err)
}
