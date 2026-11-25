# Birdview

基于 **Go + MyGo Native UI** 开发的桌面端观鸟知识图鉴与个人观鸟日志软件。所有鸟类分类名录、物种图鉴、鸟鸣录音与区域实时观鸟动态均通过在线 API 按需加载，避免在本地打包海量媒体资源。

## 当前特性

- **鸟类图鉴 (`internal/appui/view_guide.go`)**：
  - 实时加载 eBird API v2 简体中文名录 (`zh_SIM`)，支持按范围（中国鸟类 / 当前省份 / 全球名录 / 已解锁）、目/科与中英文/拉丁学名/代码检索。
  - 按需加载物种标准图鉴照、鸟鸣音频流与中文维基百科摘要。
- **观鸟日志与现场录音 (`internal/appui/view_journal.go`, `internal/audioctl`)**：
  - 记录每日观鸟物种、时间、地点、数量与观察笔记，支持上传多张个人照片。
  - 内置无 CGO 原生 Windows 麦克风现场录音按键（基于 `winmm.dll`，实时显示音量电平与录音时长，生成 44.1kHz WAV），也支持导入本地 WAV / MP3 / OGG / FLAC / M4A 录音并直接回放。
  - 支持将个人日志导出为 CSV（UTF-8 BOM）与 JSON。
- **已解锁鸟种 (`internal/appui/view_journal.go`)**：
  - 记录新鸟种后自动解锁并弹出提示，统计已解锁进度、观测次数及科属分布。
- **设置中心 (`internal/appui/view_settings.go`)**：
  - 配置与在线验证 eBird API Key（优先存入系统凭据管理器）及可选 Xeno-canto v3 Key、默认地区、浅色/深色外观切换与 64 MB LRU 缓存管理。

## 环境要求与验证

- Go 1.27+（纯 Go，无需 CGO：`CGO_ENABLED=0`）
- MyGo v0.9.2

```bash
go test -v ./...
CGO_ENABLED=0 go build ./...
go tool mygo vet .
```
