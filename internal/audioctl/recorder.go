package audioctl

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"time"
)

// RecorderStatus 描述当前麦克风现场录音的状态、时长与实时电平。
type RecorderStatus struct {
	Recording bool
	Elapsed   time.Duration
	Level     float64 // 归一化 RMS 音量电平 [0, 1]
	ErrMsg    string
}

// Recorder 定义现场录音控制器接口（Windows 使用 winmm.dll 原生实现，测试可注入模拟实现）。
type Recorder interface {
	Start() error
	Stop(destWavPath string) error
	Cancel()
	Status() RecorderStatus
}

// WriteWAV 将 PCM 样本封装为标准 44 字节 RIFF/WAVE 文件并写入磁盘。
func WriteWAV(destPath string, pcm []byte, sampleRate, channels, bitsPerSample int) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	blockAlign := uint16(channels * bitsPerSample / 8)
	dataSize := uint32(len(pcm))
	riffSize := 36 + dataSize

	var hdr [44]byte
	copy(hdr[0:4], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:8], riffSize)
	copy(hdr[8:12], "WAVE")
	copy(hdr[12:16], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:20], 16) // PCM fmt chunk size
	binary.LittleEndian.PutUint16(hdr[20:22], 1)  // PCM format = 1
	binary.LittleEndian.PutUint16(hdr[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(hdr[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(hdr[28:32], byteRate)
	binary.LittleEndian.PutUint16(hdr[32:34], blockAlign)
	binary.LittleEndian.PutUint16(hdr[34:36], uint16(bitsPerSample))
	copy(hdr[36:40], "data")
	binary.LittleEndian.PutUint32(hdr[40:44], dataSize)

	if _, err := f.Write(hdr[:]); err != nil {
		return err
	}
	if len(pcm) > 0 {
		if _, err := f.Write(pcm); err != nil {
			return err
		}
	}
	return nil
}
