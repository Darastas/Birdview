//go:build !windows && !android && !ios

package audioctl

import "errors"

type stubRecorder struct{}

// NewRecorder 在不支持原生录音的桌面平台返回占位实现（支持导入音频）。
func NewRecorder() Recorder {
	return &stubRecorder{}
}

func (*stubRecorder) Start() error                  { return errors.ErrUnsupported }
func (*stubRecorder) Stop(destWavPath string) error { return errors.ErrUnsupported }
func (*stubRecorder) Cancel()                       {}
func (*stubRecorder) Status() RecorderStatus        { return RecorderStatus{ErrMsg: "当前系统暂不支持直接麦克风录音，请使用导入音频"} }
