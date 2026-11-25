//go:build !windows

package audioctl

import "errors"

type stubRecorder struct{}

// NewRecorder 在非 Windows 平台返回不支持原生麦克风录音的占位实现（支持导入音频）。
func NewRecorder() Recorder {
	return &stubRecorder{}
}

func (*stubRecorder) Start() error                  { return errors.ErrUnsupported }
func (*stubRecorder) Stop(destWavPath string) error { return errors.ErrUnsupported }
func (*stubRecorder) Cancel()                       {}
func (*stubRecorder) Status() RecorderStatus        { return RecorderStatus{ErrMsg: "当前系统暂不支持直接麦克风录音，请使用导入音频"} }
