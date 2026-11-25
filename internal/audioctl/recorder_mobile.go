//go:build android || ios

package audioctl

import (
	"context"
	"sync"
	"time"

	"github.com/Darastas/Birdview/internal/micplugin"
)

type mobileRecorder struct {
	mu        sync.Mutex
	recording bool
	startedAt time.Time
	level     float64
	elapsed   time.Duration
	errMsg    string
	offLevel  func()
}

// NewRecorder 在 Android 与 iOS 平台返回调用原生 BirdviewMicPlugin 的现场录音器。
func NewRecorder() Recorder {
	return &mobileRecorder{}
}

func (r *mobileRecorder) Start() error {
	r.mu.Lock()
	if r.offLevel == nil {
		r.offLevel = micplugin.OnLevel(func(level float64, elapsedMs int64) {
			r.mu.Lock()
			r.level = level
			r.elapsed = time.Duration(elapsedMs) * time.Millisecond
			r.mu.Unlock()
		})
	}
	r.errMsg = ""
	r.level = 0
	r.elapsed = 0
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := micplugin.Start(ctx); err != nil {
		r.mu.Lock()
		r.recording = false
		r.errMsg = err.Error()
		r.mu.Unlock()
		return err
	}

	r.mu.Lock()
	r.recording = true
	r.startedAt = time.Now()
	r.errMsg = ""
	r.mu.Unlock()
	return nil
}

func (r *mobileRecorder) Stop(destWavPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := micplugin.Stop(ctx, destWavPath)
	r.mu.Lock()
	r.recording = false
	r.level = 0
	if err != nil {
		r.errMsg = err.Error()
	} else {
		r.errMsg = ""
	}
	r.mu.Unlock()
	return err
}

func (r *mobileRecorder) Cancel() {
	r.mu.Lock()
	r.recording = false
	r.level = 0
	r.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = micplugin.Cancel(ctx)
	}()
}

func (r *mobileRecorder) Status() RecorderStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	elapsed := r.elapsed
	if r.recording && elapsed <= 0 {
		elapsed = time.Since(r.startedAt)
	}
	return RecorderStatus{
		Recording: r.recording,
		Elapsed:   elapsed,
		Level:     r.level,
		ErrMsg:    r.errMsg,
	}
}
