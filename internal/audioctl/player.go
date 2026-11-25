package audioctl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/store"
	"github.com/egoist/mygo/audio"
	_ "github.com/egoist/mygo/audio/flac"
	_ "github.com/egoist/mygo/audio/mp3"
	_ "github.com/egoist/mygo/audio/mp4"
	_ "github.com/egoist/mygo/audio/vorbis"
)

// PlayerStatus 描述当前音频播放器的实时状态，供 UI 渲染进度条与控制按键。
type PlayerStatus struct {
	Active    bool
	Loading   bool
	Playing   bool
	SourceKey string
	Title     string
	Subtitle  string
	Position  time.Duration
	Duration  time.Duration
	Volume    float64
	ErrMsg    string
}

// PlayerController 封装 MyGo audio 流式解码与播放控制，支持在线音频按需拉取与本地音频文件播放。
type PlayerController struct {
	mu        sync.Mutex
	player    *audio.Player
	sourceKey string
	title     string
	subtitle  string
	loading   bool
	errMsg    string
	volume    float64
	seq       uint64
	cancel    context.CancelFunc
}

// NewPlayerController 创建默认音量为 0.9 的音频控制器。
func NewPlayerController() *PlayerController {
	return &PlayerController{
		volume: 0.9,
	}
}

// PlayURL 在后台按需下载（优先命中缓存）并流式播放远程鸟鸣音频。
func (pc *PlayerController) PlayURL(rawURL, title, subtitle string, cache *store.Cache, client *api.Client, onUpdate func()) {
	pc.mu.Lock()
	if pc.cancel != nil {
		pc.cancel()
	}
	pc.stopPlayerLocked()
	pc.seq++
	seq := pc.seq
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	pc.cancel = cancel
	pc.sourceKey = rawURL
	pc.title = title
	pc.subtitle = subtitle
	pc.loading = true
	pc.errMsg = ""
	vol := pc.volume
	pc.mu.Unlock()

	if onUpdate != nil {
		onUpdate()
	}

	go func() {
		defer cancel()
		var data []byte
		var ok bool
		if cache != nil {
			data, ok = cache.GetBlob("audio:" + rawURL)
		}
		if !ok {
			var err error
			data, err = client.FetchBytes(ctx, rawURL)
			if err != nil {
				pc.setError(seq, fmt.Sprintf("音频加载失败: %v", err), onUpdate)
				return
			}
			if cache != nil {
				_ = cache.PutBlob("audio:"+rawURL, data)
			}
		}

		p, err := audio.OpenStream(bytes.NewReader(data))
		if err != nil {
			pc.setError(seq, fmt.Sprintf("不支持的音频格式: %v", err), onUpdate)
			return
		}
		p.SetVolume(vol)
		p.OnEnd(func() {
			if onUpdate != nil {
				onUpdate()
			}
		})

		pc.mu.Lock()
		if pc.seq != seq {
			pc.mu.Unlock()
			p.Close()
			return
		}
		pc.player = p
		pc.loading = false
		p.Play()
		pc.mu.Unlock()

		if onUpdate != nil {
			onUpdate()
		}
	}()
}

// PlayFile 流式播放本地音频文件（支持 WAV / MP3 / OGG / FLAC / M4A）。
func (pc *PlayerController) PlayFile(filePath, title, subtitle string, onUpdate func()) {
	pc.mu.Lock()
	if pc.cancel != nil {
		pc.cancel()
		pc.cancel = nil
	}
	pc.stopPlayerLocked()
	pc.seq++
	seq := pc.seq
	pc.sourceKey = filePath
	pc.title = title
	pc.subtitle = subtitle
	pc.loading = false
	pc.errMsg = ""
	vol := pc.volume
	pc.mu.Unlock()

	f, err := os.Open(filePath)
	if err != nil {
		pc.setError(seq, fmt.Sprintf("无法打开音频文件: %v", err), onUpdate)
		return
	}
	p, err := audio.OpenStream(f)
	if err != nil {
		_ = f.Close()
		pc.setError(seq, fmt.Sprintf("解码音频失败: %v", err), onUpdate)
		return
	}
	p.SetVolume(vol)
	p.OnEnd(func() {
		if onUpdate != nil {
			onUpdate()
		}
	})

	pc.mu.Lock()
	if pc.seq != seq {
		pc.mu.Unlock()
		p.Close()
		return
	}
	pc.player = p
	p.Play()
	pc.mu.Unlock()

	if onUpdate != nil {
		onUpdate()
	}
}

// TogglePause 切换当前音轨的播放与暂停。
func (pc *PlayerController) TogglePause() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if pc.player == nil {
		return
	}
	if pc.player.Playing() {
		pc.player.Pause()
	} else {
		pc.player.Play()
	}
}

// Stop 停止播放并关闭当前音轨。
func (pc *PlayerController) Stop() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if pc.cancel != nil {
		pc.cancel()
		pc.cancel = nil
	}
	pc.seq++
	pc.stopPlayerLocked()
	pc.sourceKey = ""
	pc.title = ""
	pc.subtitle = ""
	pc.loading = false
	pc.errMsg = ""
}

// SeekFraction 按 [0, 1] 比例跳转播放进度。
func (pc *PlayerController) SeekFraction(frac float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if pc.player == nil {
		return
	}
	dur := pc.player.Duration()
	if dur <= 0 {
		return
	}
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	pc.player.Seek(time.Duration(float64(dur) * frac))
}

// SetVolume 调整播放音量（0.0 - 1.0）。
func (pc *PlayerController) SetVolume(v float64) {
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.volume = v
	if pc.player != nil {
		pc.player.SetVolume(v)
	}
}

// Status 返回当前播放器快照。
func (pc *PlayerController) Status() PlayerStatus {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	st := PlayerStatus{
		Active:    pc.player != nil || pc.loading || pc.errMsg != "",
		Loading:   pc.loading,
		SourceKey: pc.sourceKey,
		Title:     pc.title,
		Subtitle:  pc.subtitle,
		Volume:    pc.volume,
		ErrMsg:    pc.errMsg,
	}
	if pc.player != nil {
		st.Playing = pc.player.Playing()
		st.Position = pc.player.Position()
		st.Duration = pc.player.Duration()
	}
	return st
}

func (pc *PlayerController) setError(seq uint64, msg string, onUpdate func()) {
	pc.mu.Lock()
	if pc.seq != seq {
		pc.mu.Unlock()
		return
	}
	pc.loading = false
	pc.errMsg = msg
	pc.mu.Unlock()
	if onUpdate != nil {
		onUpdate()
	}
}

func (pc *PlayerController) stopPlayerLocked() {
	if pc.player != nil {
		pc.player.Stop()
		pc.player.Close()
		pc.player = nil
	}
}
