package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Darastas/Birdview/internal/model"
)

type memorySecretStore struct {
	data        map[string][]byte
	unsupported bool
}

func (m *memorySecretStore) Get(key string) ([]byte, error) {
	if m.unsupported {
		return nil, errors.ErrUnsupported
	}
	v, ok := m.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}

func (m *memorySecretStore) Set(key string, value []byte) error {
	if m.unsupported {
		return errors.ErrUnsupported
	}
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = append([]byte(nil), value...)
	return nil
}

func (m *memorySecretStore) Delete(key string) error {
	if m.unsupported {
		return errors.ErrUnsupported
	}
	delete(m.data, key)
	return nil
}

func TestStoreSecretsAndFallback(t *testing.T) {
	dir := t.TempDir()
	sec := &memorySecretStore{}
	st, err := New(dir, sec)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveEbirdKey("test-ebird-key"); err != nil {
		t.Fatal(err)
	}
	if got := st.LoadEbirdKey(); got != "test-ebird-key" {
		t.Fatalf("got %q, want test-ebird-key", got)
	}
	if st.Settings().EbirdKeyBackup != "" {
		t.Fatalf("expected backup cleared when secret store succeeds")
	}

	// 验证不支持凭据库时的本地回退存储
	dir2 := t.TempDir()
	st2, err := New(dir2, &memorySecretStore{unsupported: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := st2.SaveEbirdKey("fallback-key"); err != nil {
		t.Fatal(err)
	}
	if got := st2.LoadEbirdKey(); got != "fallback-key" {
		t.Fatalf("got %q, want fallback-key", got)
	}
}

func TestJournalEntryUnlockAndExport(t *testing.T) {
	dir := t.TempDir()
	st, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	photoSrc := filepath.Join(t.TempDir(), "bird.jpg")
	if err := os.WriteFile(photoSrc, []byte("fake-jpg"), 0o644); err != nil {
		t.Fatal(err)
	}

	e1, unlocked1, err := st.AddJournalEntry(model.JournalEntry{
		SpeciesCode: "comkin1",
		ComName:     "普通翠鸟",
		SciName:     "Alcedo atthis",
		Order:       "Coraciiformes",
		FamilyName:  "翠鸟科",
		ObservedAt:  "2026-11-24 08:30",
		Location:    "北京颐和园",
		Count:       2,
		Notes:       "停在荷塘枯枝上捕鱼",
		PhotoPaths:  []string{photoSrc},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !unlocked1 {
		t.Fatal("first sighting of comkin1 should unlock species")
	}
	if !st.IsUnlocked("comkin1") {
		t.Fatal("comkin1 should be marked unlocked")
	}

	// 第二次记录同一鸟种不应再次触发“首次解锁”
	e2, unlocked2, err := st.AddJournalEntry(model.JournalEntry{
		SpeciesCode: "comkin1",
		ComName:     "普通翠鸟",
		SciName:     "Alcedo atthis",
		ObservedAt:  "2026-11-25 09:00",
		Location:    "北京圆明园",
		Count:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if unlocked2 {
		t.Fatal("second sighting of comkin1 should not report newlyUnlocked")
	}

	birds := st.UnlockedBirds()
	if len(birds) != 1 || birds[0].TotalSightings != 2 || birds[0].FirstLocation != "北京颐和园" {
		t.Fatalf("unexpected unlocked list: %+v", birds)
	}

	// 删除两条日志后解锁状态应同步撤销
	if err := st.DeleteJournalEntry(e1.ID); err != nil {
		t.Fatal(err)
	}
	if !st.IsUnlocked("comkin1") {
		t.Fatal("comkin1 should stay unlocked while e2 exists")
	}

	csvPath := filepath.Join(t.TempDir(), "export.csv")
	if err := st.ExportJournalCSV(csvPath); err != nil {
		t.Fatal(err)
	}
	csvBytes, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(csvBytes, []byte{0xEF, 0xBB, 0xBF}) || !strings.Contains(string(csvBytes), "普通翠鸟") {
		t.Fatalf("unexpected csv content: %q", string(csvBytes))
	}

	if err := st.DeleteJournalEntry(e2.ID); err != nil {
		t.Fatal(err)
	}
	if st.IsUnlocked("comkin1") {
		t.Fatal("comkin1 should no longer be unlocked after deleting all entries")
	}
}

func TestCacheLRUEviction(t *testing.T) {
	c, err := NewCache(t.TempDir(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PutBlob("k1", []byte("1234567890")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(15 * time.Millisecond)
	if err := c.PutBlob("k2", []byte("abcdefghij")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(15 * time.Millisecond)
	if err := c.PutBlob("k3", []byte("ZZZZZZZZZZ")); err != nil {
		t.Fatal(err)
	}

	if _, ok := c.GetBlob("k1"); ok {
		t.Fatal("oldest entry k1 should have been evicted")
	}
	if _, ok := c.GetBlob("k3"); !ok {
		t.Fatal("newest entry k3 should remain in cache")
	}

	bytesCount, filesCount := c.Stats()
	if bytesCount > 20 || filesCount != 2 {
		t.Fatalf("unexpected cache stats: bytes=%d files=%d", bytesCount, filesCount)
	}
}

func TestCacheThumbIndex(t *testing.T) {
	c, err := NewCache(t.TempDir(), 4*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SaveThumbIndex(map[string]string{
		"comkin1": "https://example.com/comkin1.jpg",
		"creibi1": "https://example.com/creibi1.jpg",
	}); err != nil {
		t.Fatal(err)
	}
	got := c.LoadThumbIndex()
	if len(got) != 2 || got["comkin1"] != "https://example.com/comkin1.jpg" {
		t.Fatalf("unexpected thumb index: %+v", got)
	}
}
