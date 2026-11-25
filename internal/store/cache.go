package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
)

const DefaultMaxCacheBytes int64 = 64 * 1024 * 1024 // 64 MB 媒体缓存上限

// Cache 管理在线图鉴图片、鸟鸣音频与分类名录的有上限磁盘缓存。
type Cache struct {
	mu       sync.Mutex
	dir      string
	maxBytes int64
}

// NewCache 在指定目录创建有界缓存。
func NewCache(cacheDir string, maxBytes int64) (*Cache, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxCacheBytes
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "blobs"), 0o755); err != nil {
		return nil, err
	}
	return &Cache{
		dir:      cacheDir,
		maxBytes: maxBytes,
	}, nil
}

// NewDefaultCache 使用 MyGo 标准 PathCache 目录创建缓存。
func NewDefaultCache() (*Cache, error) {
	dir, err := mygo.App.Path(mygo.PathCache)
	if err != nil {
		return nil, err
	}
	return NewCache(dir, DefaultMaxCacheBytes)
}

func (c *Cache) blobPath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, "blobs", hex.EncodeToString(sum[:]))
}

// GetBlob 读取缓存的二进制媒体（并刷新访问时间用于 LRU）。
func (c *Cache) GetBlob(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.blobPath(key)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	now := time.Now()
	_ = os.Chtimes(p, now, now)
	return data, true
}

// PutBlob 写入二进制缓存并在超限时按最久未用顺序自动淘汰。
func (c *Cache) PutBlob(key string, data []byte) error {
	if len(data) == 0 || int64(len(data)) > c.maxBytes {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.blobPath(key)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return err
	}
	c.evictLocked()
	return nil
}

// LoadTaxonomy 读取本地缓存的 eBird 分类名录（24 小时内视为新鲜）。
func (c *Cache) LoadTaxonomy(locale string, maxAge time.Duration) ([]model.Taxon, bool) {
	p := filepath.Join(c.dir, "taxonomy_"+locale+".json")
	info, err := os.Stat(p)
	if err != nil || (maxAge > 0 && time.Since(info.ModTime()) > maxAge) {
		return nil, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	var taxa []model.Taxon
	if err := json.Unmarshal(data, &taxa); err != nil || len(taxa) == 0 {
		return nil, false
	}
	return taxa, true
}

// SaveTaxonomy 将 eBird 分类名录写入缓存目录。
func (c *Cache) SaveTaxonomy(locale string, taxa []model.Taxon) error {
	p := filepath.Join(c.dir, "taxonomy_"+locale+".json")
	return writeJSON(p, taxa)
}

// LoadThumbIndex 读取本地缓存的鸟种缩略图 URL 索引（按 SpeciesCode 索引）。
func (c *Cache) LoadThumbIndex() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := filepath.Join(c.dir, "species_thumbs.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return make(map[string]string)
	}
	var out map[string]string
	if err := json.Unmarshal(data, &out); err != nil || out == nil {
		return make(map[string]string)
	}
	return out
}

// SaveThumbIndex 持久化鸟种缩略图 URL 索引以减少图鉴列表重复请求。
func (c *Cache) SaveThumbIndex(idx map[string]string) error {
	if idx == nil {
		idx = map[string]string{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	p := filepath.Join(c.dir, "species_thumbs.json")
	return writeJSON(p, idx)
}

// Stats 返回当前媒体缓存（含名录缓存）总字节数与文件数。
func (c *Cache) Stats() (bytes int64, files int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = filepath.Walk(c.dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			bytes += info.Size()
			files++
		}
		return nil
	})
	return bytes, files
}

// Clear 清空全部媒体与名录缓存。
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := os.RemoveAll(c.dir); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(c.dir, "blobs"), 0o755)
}

type blobEntry struct {
	path    string
	size    int64
	modTime time.Time
}

func (c *Cache) evictLocked() {
	blobsDir := filepath.Join(c.dir, "blobs")
	entries, err := os.ReadDir(blobsDir)
	if err != nil {
		return
	}
	var items []blobEntry
	var total int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		total += info.Size()
		items = append(items, blobEntry{
			path:    filepath.Join(blobsDir, entry.Name()),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
	}
	if total <= c.maxBytes {
		return
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime.Before(items[j].modTime)
	})
	for _, it := range items {
		if total <= c.maxBytes {
			break
		}
		if os.Remove(it.path) == nil {
			total -= it.size
		}
	}
}
