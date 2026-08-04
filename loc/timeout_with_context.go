package main

import (
	"context"
	"fmt"
	"time"
)

func withTimeout(ctx context.Context) error {
	select {
	case <-time.After(1 * time.Second):
		return fmt.Errorf("операция превысила таймаут")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := withTimeout(ctx)
	if err != nil {
		fmt.Println("ошибка:", err)
	}
}
