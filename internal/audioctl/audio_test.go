package audioctl

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/audio"
)

func TestWriteWAVAndAudioDecoder(t *testing.T) {
	// 生成 0.2 秒 440Hz 单声道 16-bit PCM 测试波形
	sampleRate := 44100
	samples := sampleRate / 5
	pcm := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		v := int16(12000 * math.Sin(2*math.Pi*440*float64(i)/float64(sampleRate)))
		binary.LittleEndian.PutUint16(pcm[i*2:i*2+2], uint16(v))
	}

	wavPath := filepath.Join(t.TempDir(), "bird_test.wav")
	if err := WriteWAV(wavPath, pcm, sampleRate, 1, 16); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	dec, formatName, err := audio.NewDecoder(f)
	if err != nil {
		t.Fatalf("failed to decode written WAV: %v", err)
	}
	if formatName == "" || dec.SampleRate() != sampleRate || dec.Channels() != 1 {
		t.Fatalf("unexpected wav decoder metadata: format=%q rate=%d ch=%d", formatName, dec.SampleRate(), dec.Channels())
	}
}
