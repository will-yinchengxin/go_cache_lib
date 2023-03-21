package redis_lock

import (
	"context"
	_ "embed"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	//go:embed lua/unlock.lua
	luaUnlock string

	//go:embed lua/lock.lua
	luaLock string

	//go:embed lua/refresh.lua
	luaRefresh string

	ErrLockNotHold = errors.New("Do Not Hold The Lock !")

	DelSuccess, NotExistKey int64 = 1, 1
)

type Lock struct {
	client  redis.Cmdable
	key     string
	val     any
	expired time.Duration
	unlock  chan struct{}
}

func newLock(c redis.Cmdable, k string, v any, d time.Duration) *Lock {
	return &Lock{
		client:  c,
		key:     k,
		val:     v,
		expired: d,
	}
}

func (c *Lock) UnLock(ctx context.Context) error {
	res, err := c.client.Eval(ctx, luaUnlock, []string{c.key}, c.val).Int64()
	if err == redis.Nil || res != DelSuccess {
		return ErrLockNotHold
	}
	if err != nil {
		return err
	}

	return nil
}

func (c *Lock) Refresh(ctx context.Context) error {
	res, err := c.client.Eval(ctx, luaRefresh, []string{c.key}, c.val, c.expired).Int64()
	if err != nil {
		return err
	}
	if res != NotExistKey {
		return ErrLockNotHold
	}
	return nil
}

func (c *Lock) AutoRefresh(interval, timeout time.Duration) error {
	// 自动加锁到什么时候结束：1）手动 unlock  2) 续约规定的最大时长
	// 续时是否一直执行
	// 续约中途报错，应该怎么继续处理
	ticker := time.NewTicker(interval)
	defer func() {
		ticker.Stop()
	}()
	ch := make(chan struct{}, 1)
	for {
		select {
		case <-ticker.C:
			ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
			err := c.Refresh(ctx)
			cancelFunc()

			// 续约锁超过了最大限制时长
			if err == context.DeadlineExceeded {
				select {
				case ch <- struct{}{}:
				default:
				}
				continue
			}
			// TODO 针对不同错误类型应该怎么处理 锁 (调用方解决)
			if err != nil {
				return err
			}
		case <-ch:
			ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
			err := c.Refresh(ctx)
			cancelFunc()

			// 续约锁超过了最大限制时长
			if err == context.DeadlineExceeded {
				select {
				case ch <- struct{}{}:
				default:
				}
				continue
			}
			// TODO 针对不同错误类型应该怎么处理 锁 (调用方解决)
			if err != nil {
				return err
			}
		// 锁已经成功释放
		case <-c.unlock:
			return nil
		}
	}
}

// TODO
func (c *Lock) GiveLockToOther() {

}
