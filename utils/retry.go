package utils

import (
	"context"
	"github.com/avast/retry-go"
)

func Retry(ctx context.Context, attempts uint, shouldRetry func(err error) bool, op func(ctx context.Context, isRetry bool) error) error {

	retryCount := 0
	if err := retry.Do(
		func() error {
			return op(ctx, retryCount > 0)
		},
		retry.Attempts(attempts),
		retry.Delay(0),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.RetryIf(shouldRetry),
		retry.OnRetry(func(n uint, err error) {
			retryCount++
		}),
	); err != nil {
		return err
	}

	return nil
}
