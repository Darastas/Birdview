// Package micplugin 封装 Birdview 在 Android 与 iOS 上的原生麦克风录音插件桥接。
package micplugin

import (
	"context"
	"encoding/json"

	"github.com/egoist/mygo"
)

const pluginID = "birdview_mic"

// StopResult 描述结束录音后生成的本地 WAV 文件路径与录音时长。
type StopResult struct {
	Path      string `json:"path"`
	ElapsedMs int64  `json:"elapsedMs"`
}

// Available 报告当前运行平台是否支持移动端原生录音插件（仅 Android 与 iOS 返回 true）。
func Available() bool {
	return mygo.Native(pluginID).Available()
}

// Start 请求麦克风权限并开始采集单声道 44.1kHz PCM 录音（需在非主线程调用）。
func Start(ctx context.Context) error {
	return mygo.Native(pluginID).Call(ctx, "start", nil, nil)
}

// Stop 停止录音并写入标准 WAV 文件到 destPath（需在非主线程调用）。
func Stop(ctx context.Context, destPath string) (StopResult, error) {
	var res StopResult
	args := map[string]any{"path": destPath}
	err := mygo.Native(pluginID).Call(ctx, "stop", args, &res)
	return res, err
}

// Cancel 中止当前录音并清理临时音频缓存。
func Cancel(ctx context.Context) error {
	return mygo.Native(pluginID).Call(ctx, "cancel", nil, nil)
}

// OnLevel 订阅录音期间原生层每 ~80ms 推送的归一化音量电平（[0, 1]）与已录时长（毫秒）。
func OnLevel(fn func(level float64, elapsedMs int64)) func() {
	return mygo.Native(pluginID).On("level", func(data []byte) {
		var payload struct {
			Level     float64 `json:"level"`
			ElapsedMs int64   `json:"elapsedMs"`
		}
		if json.Unmarshal(data, &payload) == nil && fn != nil {
			fn(payload.Level, payload.ElapsedMs)
		}
	})
}
