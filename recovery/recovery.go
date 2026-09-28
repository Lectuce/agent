package recovery

import (
	"agent/config"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

type RecoveryState struct {
	HasEscalated                bool   // 是否已经把 max_tokens 提高过一次
	RecoveryCount               int    // 因为输出被截断而续写恢复了多少次
	Consecutive529              int    // 连续遇到多少次 HTTP 529 / 服务过载类错误
	HasAttemptedReactiveCompact bool   // 是否已经尝试过一次“上下文压缩”来恢复。
	CurrentModel                string // 当前正在使用哪个模型，初始是 PRIMARYMODEL
}

func InitRecoveryState() *RecoveryState {
	return &RecoveryState{
		HasEscalated:                false,
		RecoveryCount:               0,
		Consecutive529:              0,
		HasAttemptedReactiveCompact: false,
		CurrentModel:                config.PRIMARY_MODEL,
	}
}

func IsPromptTooLongError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *anthropic.Error
	if !errors.As(err, &apiErr) {
		return false
	}

	// prompt/context 太长
	if apiErr.StatusCode != 400 {
		return false
	}

	if apiErr.Type() != anthropic.ErrorTypeInvalidRequestError {
		return false
	}

	msg := strings.ToLower(apiErr.Error())

	return strings.Contains(msg, "prompt is too long") ||
		strings.Contains(msg, "prompt too long") ||
		strings.Contains(msg, "context window") ||
		strings.Contains(msg, "too many input tokens")
}

func WithRetry(fn func() (*anthropic.Message, error), state *RecoveryState, maxRetries int) (*anthropic.Message, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		message, err := fn()
		if err == nil {
			state.Consecutive529 = 0
			return message, nil
		}
		var apiErr *anthropic.Error
		if !errors.As(err, &apiErr) {
			// DNS、网络等非 Anthropic API 错误
			return nil, err
		}

		errType := apiErr.Type()

		isRateLimit := errType == anthropic.ErrorTypeRateLimitError
		isOverloaded := errType == anthropic.ErrorTypeOverloadedError

		if !isRateLimit && !isOverloaded {
			return nil, err
		}

		if isOverloaded {
			state.Consecutive529++
			if state.Consecutive529 >= config.MAX_CONSECUTIVE_529 && config.FALLBACK_MODEL != "" {
				state.CurrentModel = config.FALLBACK_MODEL
			}
		}

		delay := RetryDelay(attempt, 0)

		fmt.Printf(
			"[retry] attempt=%d delay=%v type=%s\n",
			attempt+1,
			delay,
			errType,
		)

		time.Sleep(delay)
	}

	return nil, fmt.Errorf("max retries exceeded")

}

func RetryDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}

	baseMs := math.Min(
		500*math.Pow(2, float64(attempt)),
		32000,
	)

	base := time.Duration(baseMs) * time.Millisecond

	jitter := time.Duration(
		rand.Float64() * float64(base) * 0.25,
	)

	return base + jitter
}
