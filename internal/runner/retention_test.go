package runner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockRetentionDeleter struct {
	mu    sync.Mutex
	calls []time.Time
	err   error
	count int64
}

func (m *mockRetentionDeleter) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, cutoff)
	return m.count, m.err
}

func (m *mockRetentionDeleter) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func TestStartRetentionRunner_DeletesAndShutdown(t *testing.T) {
	deleter := &mockRetentionDeleter{count: 5}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		StartRetentionRunner(ctx, deleter, 24*time.Hour, 10*time.Millisecond)
		close(done)
	}()

	assert.Eventually(t, func() bool {
		return deleter.callCount() >= 1
	}, 2*time.Second, 5*time.Millisecond)

	cancel()
	<-done
}

func TestStartRetentionRunner_ErrorContinues(t *testing.T) {
	deleter := &mockRetentionDeleter{err: errors.New("db down"), count: 0}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		StartRetentionRunner(ctx, deleter, 24*time.Hour, 10*time.Millisecond)
		close(done)
	}()

	assert.Eventually(t, func() bool {
		return deleter.callCount() >= 2
	}, 2*time.Second, 5*time.Millisecond)

	cancel()
	<-done
}

func TestStartRetentionRunner_DefaultsOnZero(t *testing.T) {
	deleter := &mockRetentionDeleter{count: 0}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	StartRetentionRunner(ctx, deleter, 0, 0)
}

func TestStartRetentionRunner_ZeroCount(t *testing.T) {
	deleter := &mockRetentionDeleter{count: 0}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		StartRetentionRunner(ctx, deleter, 24*time.Hour, 10*time.Millisecond)
		close(done)
	}()

	assert.Eventually(t, func() bool {
		return deleter.callCount() >= 1
	}, 2*time.Second, 5*time.Millisecond)

	cancel()
	<-done
}
