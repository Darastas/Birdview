//go:build windows

package audioctl

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	waveFormatPCM   = 1
	waveMapper      = ^uintptr(0) // 0xFFFFFFFF / -1
	whdrDone        = 0x00000001
	recSampleRate   = 44100
	recChannels     = 1
	recBits         = 16
	recBufferCount  = 4
	recBufferMillis = 100
	recBufferBytes  = (recSampleRate * recChannels * (recBits / 8) * recBufferMillis) / 1000
)

var (
	winmm                   = syscall.NewLazyDLL("winmm.dll")
	procWaveInGetNumDevs    = winmm.NewProc("waveInGetNumDevs")
	procWaveInOpen          = winmm.NewProc("waveInOpen")
	procWaveInPrepareHeader = winmm.NewProc("waveInPrepareHeader")
	procWaveInUnprepareHdr  = winmm.NewProc("waveInUnprepareHeader")
	procWaveInAddBuffer     = winmm.NewProc("waveInAddBuffer")
	procWaveInStart         = winmm.NewProc("waveInStart")
	procWaveInStop          = winmm.NewProc("waveInStop")
	procWaveInReset         = winmm.NewProc("waveInReset")
	procWaveInClose         = winmm.NewProc("waveInClose")
)

type waveFormatEx struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	Size           uint16
}

type waveHdr struct {
	Data          uintptr
	BufferLength  uint32
	BytesRecorded uint32
	User          uintptr
	Flags         uint32
	Loops         uint32
	Next          uintptr
	Reserved      uintptr
}

type windowsRecorder struct {
	mu        sync.Mutex
	recording bool
	startedAt time.Time
	level     float64
	errMsg    string
	pcm       []byte

	hWaveIn uintptr
	hdrs    []waveHdr
	bufs    [][]byte
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewRecorder 创建当前平台的原生无 CGO 麦克风录音器。
func NewRecorder() Recorder {
	return &windowsRecorder{}
}

func (r *windowsRecorder) Start() error {
	r.mu.Lock()
	if r.recording {
		r.mu.Unlock()
		return errors.New("当前正在录音中")
	}

	numDevs, _, _ := procWaveInGetNumDevs.Call()
	if numDevs == 0 {
		r.errMsg = "未检测到可用麦克风设备，可使用旁边的“导入音频”功能"
		r.mu.Unlock()
		return errors.New(r.errMsg)
	}

	wfx := waveFormatEx{
		FormatTag:      waveFormatPCM,
		Channels:       recChannels,
		SamplesPerSec:  recSampleRate,
		AvgBytesPerSec: recSampleRate * recChannels * (recBits / 8),
		BlockAlign:     recChannels * (recBits / 8),
		BitsPerSample:  recBits,
		Size:           0,
	}

	var hWaveIn uintptr
	ret, _, _ := procWaveInOpen.Call(
		uintptr(unsafe.Pointer(&hWaveIn)),
		waveMapper,
		uintptr(unsafe.Pointer(&wfx)),
		0, 0, 0, // CALLBACK_NULL
	)
	if ret != 0 {
		r.errMsg = fmt.Sprintf("打开麦克风失败 (错误码 %d)，可使用旁边的“导入音频”功能", ret)
		r.mu.Unlock()
		return errors.New(r.errMsg)
	}

	hdrs := make([]waveHdr, recBufferCount)
	bufs := make([][]byte, recBufferCount)
	for i := 0; i < recBufferCount; i++ {
		buf := make([]byte, recBufferBytes)
		bufs[i] = buf
		hdrs[i] = waveHdr{
			Data:         uintptr(unsafe.Pointer(&buf[0])),
			BufferLength: uint32(len(buf)),
		}
		if res, _, _ := procWaveInPrepareHeader.Call(hWaveIn, uintptr(unsafe.Pointer(&hdrs[i])), unsafe.Sizeof(hdrs[i])); res != 0 {
			_, _, _ = procWaveInClose.Call(hWaveIn)
			r.errMsg = fmt.Sprintf("初始化录音缓冲区失败 (%d)", res)
			r.mu.Unlock()
			return errors.New(r.errMsg)
		}
		if res, _, _ := procWaveInAddBuffer.Call(hWaveIn, uintptr(unsafe.Pointer(&hdrs[i])), unsafe.Sizeof(hdrs[i])); res != 0 {
			_, _, _ = procWaveInReset.Call(hWaveIn)
			_, _, _ = procWaveInClose.Call(hWaveIn)
			r.errMsg = fmt.Sprintf("提交录音缓冲区失败 (%d)", res)
			r.mu.Unlock()
			return errors.New(r.errMsg)
		}
	}

	if res, _, _ := procWaveInStart.Call(hWaveIn); res != 0 {
		_, _, _ = procWaveInReset.Call(hWaveIn)
		_, _, _ = procWaveInClose.Call(hWaveIn)
		r.errMsg = fmt.Sprintf("启动麦克风采集失败 (%d)", res)
		r.mu.Unlock()
		return errors.New(r.errMsg)
	}

	r.hWaveIn = hWaveIn
	r.hdrs = hdrs
	r.bufs = bufs
	r.pcm = make([]byte, 0, recSampleRate*2*10)
	r.recording = true
	r.startedAt = time.Now()
	r.level = 0
	r.errMsg = ""
	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})
	r.mu.Unlock()

	go r.pollLoop()
	return nil
}

func (r *windowsRecorder) pollLoop() {
	defer close(r.doneCh)
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.drainDoneBuffers(true)
		}
	}
}

func (r *windowsRecorder) drainDoneBuffers(requeue bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.hWaveIn == 0 {
		return
	}

	for i := range r.hdrs {
		hdr := &r.hdrs[i]
		if hdr.Flags&whdrDone == 0 {
			continue
		}
		n := int(hdr.BytesRecorded)
		if n > 0 && n <= len(r.bufs[i]) {
			chunk := r.bufs[i][:n]
			r.pcm = append(r.pcm, chunk...)
			r.level = calcPCM16Level(chunk)
		}
		if requeue && r.recording {
			hdr.Flags &^= whdrDone
			hdr.BytesRecorded = 0
			_, _, _ = procWaveInAddBuffer.Call(r.hWaveIn, uintptr(unsafe.Pointer(hdr)), unsafe.Sizeof(*hdr))
		}
	}
}

func (r *windowsRecorder) Stop(destWavPath string) error {
	pcm, err := r.stopInternal()
	if err != nil {
		return err
	}
	if len(pcm) == 0 {
		return errors.New("未采集到有效的音频数据")
	}
	return WriteWAV(destWavPath, pcm, recSampleRate, recChannels, recBits)
}

func (r *windowsRecorder) Cancel() {
	_, _ = r.stopInternal()
}

func (r *windowsRecorder) stopInternal() ([]byte, error) {
	r.mu.Lock()
	if !r.recording {
		r.mu.Unlock()
		return nil, errors.New("当前未在录音")
	}
	r.recording = false
	stopCh := r.stopCh
	doneCh := r.doneCh
	hWaveIn := r.hWaveIn
	r.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	if doneCh != nil {
		<-doneCh
	}

	if hWaveIn != 0 {
		_, _, _ = procWaveInStop.Call(hWaveIn)
		_, _, _ = procWaveInReset.Call(hWaveIn)
	}
	r.drainDoneBuffers(false)

	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.hdrs {
		if r.hWaveIn != 0 {
			_, _, _ = procWaveInUnprepareHdr.Call(r.hWaveIn, uintptr(unsafe.Pointer(&r.hdrs[i])), unsafe.Sizeof(r.hdrs[i]))
		}
	}
	if r.hWaveIn != 0 {
		_, _, _ = procWaveInClose.Call(r.hWaveIn)
		r.hWaveIn = 0
	}
	runtime.KeepAlive(r.hdrs)
	runtime.KeepAlive(r.bufs)
	out := append([]byte(nil), r.pcm...)
	r.hdrs = nil
	r.bufs = nil
	r.pcm = nil
	r.level = 0
	return out, nil
}

func (r *windowsRecorder) Status() RecorderStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	var elapsed time.Duration
	if r.recording {
		elapsed = time.Since(r.startedAt)
	}
	return RecorderStatus{
		Recording: r.recording,
		Elapsed:   elapsed,
		Level:     r.level,
		ErrMsg:    r.errMsg,
	}
}

func calcPCM16Level(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	var sumSq float64
	samples := len(pcm) / 2
	for i := 0; i < samples; i++ {
		s := int16(uint16(pcm[i*2]) | uint16(pcm[i*2+1])<<8)
		f := float64(s) / 32768.0
		sumSq += f * f
	}
	rms := math.Sqrt(sumSq / float64(samples))
	level := rms * 3.5
	if level > 1 {
		level = 1
	}
	return level
}
