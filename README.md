# Birdview

基于 **Go + MyGo Native UI** 开发的桌面端观鸟知识图鉴与个人观鸟日志软件。所有鸟类分类名录、物种图鉴、鸟鸣录音与区域实时观鸟动态均通过在线 API 按需加载，避免在本地打包海量媒体资源。

## 当前特性

- **鸟类图鉴 (`internal/appui/view_guide.go`)**：
  - 实时加载 eBird API v2 简体中文名录 (`zh_SIM`)，支持按范围（中国鸟类 / 当前省份 / 全球名录 / 已解锁）、目/科与中英文/拉丁学名/代码检索。
  - 按需从云端加载物种标准图鉴照、鸟鸣音频流（支持内置流式播放器、进度拖拽与音量调节）及中文维基百科摘要。
- **音频流式播放引擎 (`internal/audioctl`)**：基于 `mygo/audio` 支持 MP3 / WAV / OGG / FLAC / M4A 流式解码播放，自动命中 64 MB LRU 磁盘缓存。
- **设置中心 (`internal/appui/view_settings.go`)**：支持配置与在线验证 eBird API Key（优先存入系统凭据管理器）及可选 Xeno-canto v3 Key、默认地区、浅色/深色外观切换与缓存清理。

## 环境要求与验证

- Go 1.27+（纯 Go，无需 CGO：`CGO_ENABLED=0`）
- MyGo v0.9.2

```bash
go test ./...
CGO_ENABLED=0 go build ./...
go tool mygo vet .
```
