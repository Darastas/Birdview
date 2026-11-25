package store

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
)

const (
	ebirdSecretKey = "ebird-api-key"
	xenoSecretKey  = "xenocanto-api-key"
)

// SecretStore 抽象系统凭据读写，便于生产环境接入 mygo.Secrets，测试环境无窗口运行。
type SecretStore interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
}

type mygoSecretStore struct{}

func (mygoSecretStore) Get(key string) ([]byte, error)        { return mygo.Secrets.Get(key) }
func (mygoSecretStore) Set(key string, value []byte) error  { return mygo.Secrets.Set(key, value) }
func (mygoSecretStore) Delete(key string) error             { return mygo.Secrets.Delete(key) }

// Store 管理本地设置、个人观鸟日志、媒体附件与已解锁鸟种。
type Store struct {
	mu       sync.RWMutex
	dir      string
	secrets  SecretStore
	settings model.Settings
	entries  []model.JournalEntry
	unlocked map[string]model.UnlockedBird
}

// New 创建并加载指定数据目录下的持久化 Store。
// secrets 为 nil 时自动禁用系统凭据访问（回退至本地配置或环境变量）。
func New(dataDir string, secrets SecretStore) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "journal", "media"), 0o755); err != nil {
		return nil, err
	}
	s := &Store{
		dir:      dataDir,
		secrets:  secrets,
		unlocked: make(map[string]model.UnlockedBird),
		settings: model.Settings{
			DefaultRegion: "CN-11",
			Locale:        "zh_SIM",
			ThemeMode:     "system",
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// NewDefault 使用 MyGo 的标准用户数据目录创建 Store。
func NewDefault() (*Store, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return nil, err
	}
	return New(dir, mygoSecretStore{})
}

func (s *Store) DataDir() string {
	return s.dir
}

func (s *Store) settingsPath() string { return filepath.Join(s.dir, "settings.json") }
func (s *Store) entriesPath() string  { return filepath.Join(s.dir, "journal", "entries.json") }
func (s *Store) unlockedPath() string { return filepath.Join(s.dir, "journal", "unlocked.json") }
func (s *Store) mediaDir() string     { return filepath.Join(s.dir, "journal", "media") }

func (s *Store) load() error {
	if data, err := os.ReadFile(s.settingsPath()); err == nil {
		var cfg model.Settings
		if err := json.Unmarshal(data, &cfg); err == nil {
			if cfg.DefaultRegion == "" {
				cfg.DefaultRegion = "CN-11"
			}
			if cfg.Locale == "" {
				cfg.Locale = "zh_SIM"
			}
			if cfg.ThemeMode == "" {
				cfg.ThemeMode = "system"
			}
			s.settings = cfg
		}
	}

	if data, err := os.ReadFile(s.entriesPath()); err == nil {
		var list []model.JournalEntry
		if err := json.Unmarshal(data, &list); err == nil {
			s.entries = list
		}
	}

	if data, err := os.ReadFile(s.unlockedPath()); err == nil {
		var list []model.UnlockedBird
		if err := json.Unmarshal(data, &list); err == nil {
			for _, u := range list {
				if u.SpeciesCode != "" {
					s.unlocked[u.SpeciesCode] = u
				}
			}
		}
	} else if len(s.entries) > 0 {
		s.recomputeUnlockedLocked()
	}

	s.sortEntriesLocked()
	return nil
}

func (s *Store) Settings() model.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Store) SaveSettings(cfg model.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.DefaultRegion == "" {
		cfg.DefaultRegion = "CN-11"
	}
	if cfg.Locale == "" {
		cfg.Locale = "zh_SIM"
	}
	if cfg.ThemeMode == "" {
		cfg.ThemeMode = "system"
	}
	// 保留已有回退 Key，除非显式修改
	if cfg.EbirdKeyBackup == "" && s.settings.EbirdKeyBackup != "" {
		cfg.EbirdKeyBackup = s.settings.EbirdKeyBackup
	}
	if cfg.XenoKeyBackup == "" && s.settings.XenoKeyBackup != "" {
		cfg.XenoKeyBackup = s.settings.XenoKeyBackup
	}
	s.settings = cfg
	return writeJSON(s.settingsPath(), cfg)
}

// LoadEbirdKey 按优先级读取 eBird API Key：系统凭据管理器 -> 本地回退配置 -> 环境变量 EBIRD_API_KEY。
func (s *Store) LoadEbirdKey() string {
	return s.loadSecret(ebirdSecretKey, func(cfg model.Settings) string {
		return cfg.EbirdKeyBackup
	}, "EBIRD_API_KEY")
}

// SaveEbirdKey 保存 eBird API Key，优先写入系统凭据管理器，不支持时回退至本地配置。
func (s *Store) SaveEbirdKey(key string) error {
	return s.saveSecret(ebirdSecretKey, key, func(cfg *model.Settings, val string) {
		cfg.EbirdKeyBackup = val
	})
}

// LoadXenoKey 读取可选的 Xeno-canto API v3 Key。
func (s *Store) LoadXenoKey() string {
	return s.loadSecret(xenoSecretKey, func(cfg model.Settings) string {
		return cfg.XenoKeyBackup
	}, "XENO_CANTO_API_KEY")
}

// SaveXenoKey 保存可选的 Xeno-canto API v3 Key。
func (s *Store) SaveXenoKey(key string) error {
	return s.saveSecret(xenoSecretKey, key, func(cfg *model.Settings, val string) {
		cfg.XenoKeyBackup = val
	})
}

func (s *Store) loadSecret(secretName string, backupGetter func(model.Settings) string, envName string) string {
	if s.secrets != nil {
		if b, err := s.secrets.Get(secretName); err == nil {
			if v := strings.TrimSpace(string(b)); v != "" {
				return v
			}
		}
	}
	s.mu.RLock()
	backup := strings.TrimSpace(backupGetter(s.settings))
	s.mu.RUnlock()
	if backup != "" {
		return backup
	}
	return strings.TrimSpace(os.Getenv(envName))
}

func (s *Store) saveSecret(secretName, rawVal string, backupSetter func(*model.Settings, string)) error {
	val := strings.TrimSpace(rawVal)
	s.mu.Lock()
	defer s.mu.Unlock()

	if val == "" {
		if s.secrets != nil {
			_ = s.secrets.Delete(secretName)
		}
		backupSetter(&s.settings, "")
		return writeJSON(s.settingsPath(), s.settings)
	}

	if s.secrets != nil {
		err := s.secrets.Set(secretName, []byte(val))
		if err == nil {
			// 成功写入系统凭据后清理本地明文回退字段
			backupSetter(&s.settings, "")
			return writeJSON(s.settingsPath(), s.settings)
		}
		if !errors.Is(err, errors.ErrUnsupported) {
			return err
		}
	}

	backupSetter(&s.settings, val)
	return writeJSON(s.settingsPath(), s.settings)
}

// Entries 返回按观测时间/创建时间倒序排列的全部观鸟日志。
func (s *Store) Entries() []model.JournalEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.JournalEntry, len(s.entries))
	copy(out, s.entries)
	return out
}

// UnlockedBirds 返回所有已解锁鸟种列表（按首次观测日期倒序）。
func (s *Store) UnlockedBirds() []model.UnlockedBird {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.UnlockedBird, 0, len(s.unlocked))
	for _, u := range s.unlocked {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FirstSeenAt != out[j].FirstSeenAt {
			return out[i].FirstSeenAt > out[j].FirstSeenAt
		}
		return out[i].ComName < out[j].ComName
	})
	return out
}

// IsUnlocked 判断物种是否已在个人日志中解锁。
func (s *Store) IsUnlocked(speciesCode string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.unlocked[speciesCode]
	return ok
}

// AddJournalEntry 添加一条观鸟日志（自动复制本地图片/音频到应用媒体目录）并检查是否解锁新鸟种。
func (s *Store) AddJournalEntry(entry model.JournalEntry) (model.JournalEntry, bool, error) {
	if strings.TrimSpace(entry.SpeciesCode) == "" || strings.TrimSpace(entry.ComName) == "" {
		return entry, false, errors.New("请选择要记录的鸟种")
	}

	now := time.Now()
	if entry.ID == "" {
		entry.ID = newID("j")
	}
	if entry.ObservedAt == "" {
		entry.ObservedAt = now.Format("2006-01-02 15:04")
	}
	if entry.Count <= 0 {
		entry.Count = 1
	}
	entry.CreatedAt = now
	entry.UpdatedAt = now

	copiedPhotos := make([]string, 0, len(entry.PhotoPaths))
	for _, p := range entry.PhotoPaths {
		dst, err := s.importMediaFile(p, entry.ID)
		if err != nil {
			return entry, false, fmt.Errorf("复制图片失败: %w", err)
		}
		copiedPhotos = append(copiedPhotos, dst)
	}
	entry.PhotoPaths = copiedPhotos

	if entry.AudioPath != "" {
		dst, err := s.importMediaFile(entry.AudioPath, entry.ID)
		if err != nil {
			return entry, false, fmt.Errorf("复制录音失败: %w", err)
		}
		entry.AudioPath = dst
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, alreadyUnlocked := s.unlocked[entry.SpeciesCode]
	s.entries = append([]model.JournalEntry{entry}, s.entries...)
	s.sortEntriesLocked()
	s.recomputeUnlockedLocked()

	if err := writeJSON(s.entriesPath(), s.entries); err != nil {
		return entry, false, err
	}
	if err := s.saveUnlockedLocked(); err != nil {
		return entry, false, err
	}
	return entry, !alreadyUnlocked, nil
}

// UpdateJournalEntry 更新既有日志的文字/时间/地点/数量与媒体附件，并重算解锁状态。
func (s *Store) UpdateJournalEntry(entry model.JournalEntry) (model.JournalEntry, error) {
	if entry.ID == "" {
		return entry, errors.New("日志 ID 不能为空")
	}
	copiedPhotos := make([]string, 0, len(entry.PhotoPaths))
	for _, p := range entry.PhotoPaths {
		dst, err := s.importMediaFile(p, entry.ID)
		if err != nil {
			return entry, fmt.Errorf("复制图片失败: %w", err)
		}
		copiedPhotos = append(copiedPhotos, dst)
	}
	entry.PhotoPaths = copiedPhotos

	if entry.AudioPath != "" {
		dst, err := s.importMediaFile(entry.AudioPath, entry.ID)
		if err != nil {
			return entry, fmt.Errorf("复制录音失败: %w", err)
		}
		entry.AudioPath = dst
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.entries {
		if s.entries[i].ID == entry.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return entry, errors.New("未找到要更新的观鸟日志")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.entries[idx].CreatedAt
	}
	entry.UpdatedAt = time.Now()
	if entry.Count <= 0 {
		entry.Count = 1
	}
	s.entries[idx] = entry
	s.sortEntriesLocked()
	s.recomputeUnlockedLocked()

	if err := writeJSON(s.entriesPath(), s.entries); err != nil {
		return entry, err
	}
	return entry, s.saveUnlockedLocked()
}

// DeleteJournalEntry 删除指定观鸟日志并同步重算已解锁鸟种表。
func (s *Store) DeleteJournalEntry(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := s.entries[:0]
	found := false
	for _, e := range s.entries {
		if e.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return nil
	}
	s.entries = filtered
	s.recomputeUnlockedLocked()

	if err := writeJSON(s.entriesPath(), s.entries); err != nil {
		return err
	}
	return s.saveUnlockedLocked()
}

// ExportJournalCSV 将个人观鸟日志导出为 UTF-8（含 BOM 便于 Excel 直接打开）的 CSV 文件。
func (s *Store) ExportJournalCSV(destPath string) error {
	entries := s.Entries()
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 写入 UTF-8 BOM 防止 Windows Excel 打开中文乱码
	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if err := w.Write([]string{"日志ID", "鸟种中文名", "拉丁学名", "物种代码", "目", "科", "观测时间", "观测地点", "数量", "文字记录", "图片数量", "是否有录音"}); err != nil {
		return err
	}
	for _, e := range entries {
		hasAudio := "否"
		if e.AudioPath != "" {
			hasAudio = "是"
		}
		row := []string{
			e.ID,
			e.ComName,
			e.SciName,
			e.SpeciesCode,
			e.Order,
			e.FamilyName,
			e.ObservedAt,
			e.Location,
			strconv.Itoa(e.Count),
			e.Notes,
			strconv.Itoa(len(e.PhotoPaths)),
			hasAudio,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// ExportJournalJSON 将全部日志与解锁名录导出为 JSON 文件。
func (s *Store) ExportJournalJSON(destPath string) error {
	payload := struct {
		ExportedAt time.Time            `json:"exportedAt"`
		Unlocked   []model.UnlockedBird `json:"unlocked"`
		Entries    []model.JournalEntry `json:"entries"`
	}{
		ExportedAt: time.Now(),
		Unlocked:   s.UnlockedBirds(),
		Entries:    s.Entries(),
	}
	return writeJSON(destPath, payload)
}

// LocalStorageBytes 统计个人日志及附件占用的本地磁盘大小。
func (s *Store) LocalStorageBytes() int64 {
	var total int64
	_ = filepath.Walk(s.dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

func (s *Store) importMediaFile(srcPath, entryID string) (string, error) {
	if srcPath == "" {
		return "", nil
	}
	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		return "", err
	}
	absMedia, err := filepath.Abs(s.mediaDir())
	if err != nil {
		return "", err
	}
	// 如果文件已经在应用媒体目录下，直接保留路径不重复复制
	if strings.HasPrefix(strings.ToLower(absSrc), strings.ToLower(absMedia+string(filepath.Separator))) {
		return absSrc, nil
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	if ext == "" {
		ext = ".dat"
	}
	dstName := fmt.Sprintf("%s_%s%s", entryID, newID("m"), ext)
	dstPath := filepath.Join(s.mediaDir(), dstName)

	in, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return dstPath, nil
}

func (s *Store) sortEntriesLocked() {
	sort.Slice(s.entries, func(i, j int) bool {
		if s.entries[i].ObservedAt != s.entries[j].ObservedAt {
			return s.entries[i].ObservedAt > s.entries[j].ObservedAt
		}
		return s.entries[i].CreatedAt.After(s.entries[j].CreatedAt)
	})
}

func (s *Store) recomputeUnlockedLocked() {
	next := make(map[string]model.UnlockedBird)
	// 按正序扫描以准确找出首次解锁记录
	ordered := make([]model.JournalEntry, len(s.entries))
	copy(ordered, s.entries)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ObservedAt != ordered[j].ObservedAt {
			return ordered[i].ObservedAt < ordered[j].ObservedAt
		}
		return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
	})

	for _, e := range ordered {
		if e.SpeciesCode == "" {
			continue
		}
		u, exists := next[e.SpeciesCode]
		if !exists {
			u = model.UnlockedBird{
				SpeciesCode:   e.SpeciesCode,
				ComName:       e.ComName,
				SciName:       e.SciName,
				Order:         e.Order,
				FamilyName:    e.FamilyName,
				FirstSeenAt:   e.ObservedAt,
				FirstLocation: e.Location,
				FirstEntryID:  e.ID,
			}
		}
		if u.CoverPhotoPath == "" && len(e.PhotoPaths) > 0 {
			u.CoverPhotoPath = e.PhotoPaths[0]
		}
		u.TotalSightings++
		next[e.SpeciesCode] = u
	}
	s.unlocked = next
}

func (s *Store) saveUnlockedLocked() error {
	list := make([]model.UnlockedBird, 0, len(s.unlocked))
	for _, u := range s.unlocked {
		list = append(list, u)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].FirstSeenAt > list[j].FirstSeenAt
	})
	return writeJSON(s.unlockedPath(), list)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func newID(prefix string) string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixMilli(), hex.EncodeToString(b[:]))
}
