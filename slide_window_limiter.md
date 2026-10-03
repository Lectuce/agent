# 滑动窗口限流

设计一个接口限流器。可以先写单机，再说明怎么做成多机。

要求：
1. 按 userId 限流：任意连续 windowSeconds 秒内，每个用户最多 limit 次请求
2. 方法：bool allow(String userId)，返回是否放行
3. 必须是滑动窗口，不是固定窗口
4. 不活跃用户的数据要能清理，避免内存一直涨

需要回答：
1. 用什么数据结构，核心逻辑怎么写（伪代码或任意语言）
2. 时间、空间复杂度
3. 固定窗口、滑动窗口、令牌桶、漏桶分别适合什么场景
4. 如果用 Redis 做集群限流，key、数据结构和原子操作怎么设计

 ```go 
package main

import (
	"fmt"
	"math/rand"
	"slices"
	"sync"
	"time"
)

type limiter struct {
	window time.Duration
	limit  int
	users  map[string][]time.Time
	mu     sync.Mutex
}

var window = time.Duration(10 * time.Second)
var slidingWindowLimiter = createLimiter(10, window)

func createLimiter(limit int, window time.Duration) *limiter {
	return &limiter{
		window: window,
		users:  make(map[string][]time.Time, 0),
		limit:  limit,
	}
}

func (l *limiter) allow(userID string) bool {

	l.mu.Lock()
	defer l.mu.Unlock()

	// 边界
	now := time.Now()
	leftBoundary := now.Add(-l.window)


	// 滑动
	times := l.users[userID]
	times = slices.DeleteFunc(times, func(t time.Time) bool {
		return !t.After(leftBoundary)
	})

	// 限制，更新
	if len(times) >= l.limit {
		l.users[userID] = times
		return false
	}

	// 放行，更新
	times = append(times, now)
	l.users[userID] = times
	return true
}

func (l *limiter) deleteInactive() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	leftBoundary := now.Add(-l.window)

	for userID, times := range l.users {
		times = slices.DeleteFunc(times, func(t time.Time) bool {
			return !t.After((leftBoundary))
		})

		if len(times) == 0 {
			delete(l.users, userID)
			continue
		}

		l.users[userID] = times
	}

}

func limit(userID string) bool {
	return slidingWindowLimiter.allow(userID)
}

func main() {

	var ticker = time.NewTicker(time.Duration(60 * time.Second))
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			slidingWindowLimiter.deleteInactive()
		}
	}()

	for request := 1; request <= 100; request++ {
		userID := fmt.Sprintf("%v", rand.Intn(5)+1)

		allowed := limit(userID)

		fmt.Printf("userID=%v, request=%v, allowed=%v\n", userID, request, allowed)
	}

}

```

1. 数据结构和核心逻辑

```go
type limiter struct {
	window time.Duration    // 窗口大小
	limit  int				// 最多请求
	users  map[string][]time.Time  // 每个用户窗口内的请求
	mu     sync.Mutex      // 锁
}

```

bool allow(String userId)

```go
func (l *limiter) allow(userID string) bool {

	l.mu.Lock()
	defer l.mu.Unlock()

	// 边界
	now := time.Now()
	leftBoundary := now.Add(-l.window)


	// 滑动
	times := l.users[userID]
	times = slices.DeleteFunc(times, func(t time.Time) bool {
		return !t.After(leftBoundary)
	})

	// 限制，更新
	if len(times) >= l.limit {
		l.users[userID] = times
		return false
	}

	// 放行，更新
	times = append(times, now)
	l.users[userID] = times
	return true
}

```

不活跃用户清理

```go
func (l *limiter) deleteInactive() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	leftBoundary := now.Add(-l.window)

	for userID, times := range l.users {
		times = slices.DeleteFunc(times, func(t time.Time) bool {
			return !t.After((leftBoundary))
		})

		if len(times) == 0 {
			delete(l.users, userID)
			continue
		}

		l.users[userID] = times
	}

}
```

2. 时间和空间复杂度

- 时间复杂度
- - U：活跃用户数量
- - L：单个用户窗口内的请求数量，最多为 limit。

`allow()`
`slices.DeleteFunc()`
需要扫描当前用户的时间戳：时间复杂度：`O(L)`。

Map 查询为 `O(1)`，append 平均为 `O(1)`。

总复杂度是： `O(L)`

`deleteInactive()`需要遍历所有用户及其时间戳, 时间复杂度为`O(U × L)`。


- 空间复杂度

每个用户最多保存 limit 个有效时间戳：

`O(U × L)`


3. 固定窗口、滑动窗口、令牌桶、漏桶分别适合什么场景


- 固定窗口：固定窗口在固定周期内计数，在窗口交界处可能出现双倍突发，适用于简单接口、统计、低成本限流。

- 滑动窗口：统计任意连续时间内的请求，需要记录时间戳，空间成本较高，适用于需要严格计数，不允许突发，高精度限流的场景，以及登录、支付等敏感接口。

- 令牌桶：固定速率生成令牌，请求消费令牌；桶内可积攒令牌，支持短时间突发流量，适用于可以允许短时间内突发的场景，如Web后端，API网关限流。

- 漏桶：请求进入桶内缓存，桶底以固定速率匀速流出处理；桶满直接丢弃请求，输出流量平滑，禁止突发。适用于要求均速的场景，如爬虫，调用第三方API。


4. 如果用 Redis 做集群限流，key、数据结构和原子操作怎么设计

- Redis集群限流设计：

- - map分片

- - 使用ZSet

`Key：limit:{userId}`

`Score`：请求时间戳

`Member`：唯一请求ID




- 通过 Lua 脚本在该 Key 所属的 Redis 主节点上执行原子操作

- - 删除窗口外的历史请求记录
- - 统计当前窗口内的请求数量
- - 如果数量 >= limit，拒绝请求
- - 如果数量 < limit，将本次请求加入 ZSET
- - 更新整个 ZSET Key 的 TTL
- - 返回放行