# Birdview

基于 **Go + MyGo Native UI** 开发的跨平台桌面观鸟图鉴与个人观鸟日志软件。所有物种分类名录、图鉴照片、鸟鸣参考录音、中文百科及区域实时观鸟数据均通过在线 API 按需加载，无需在安装包内打包海量媒体素材。

## 核心功能

### 1. 鸟类图鉴
- 接入 **eBird API v2** 全球物种级分类名录（支持大陆简体中文名录 `zh_SIM`、中国名录及当前省份筛选、已解锁筛选）。
- 支持按**目 / 科**分类过滤，以及中文名、拉丁学名、物种代码即时检索。
- 点击物种后按需在线加载：
  - **iNaturalist Taxa** 物种标准高清生态照片（带摄影师署名与许可信息）；
  - **iNaturalist Sounds** 免 Key 野外鸟鸣参考录音（支持在 **设置** 中配置 **Xeno-canto API v3 Key** 扩展专业曲库）；
  - **中文维基百科 REST Summary** 物种简介与外部百科链接。
- 内置流式音频播放器（基于 `mygo/audio`，支持 MP3 / WAV / OGG / FLAC / M4A），底部常驻播放进度与音量控制栏。

### 2. 观鸟日志与现场录音
- 记录观测鸟种、时间、地点、数量与详细观察笔记，支持一键从图鉴或热点带入物种与地点。
- 支持添加多张个人观测实拍照片。
- **内置原生无 CGO 麦克风现场录音**（Windows 基于 `winmm.dll`，录音期间实时显示音量电平与时长，自动生成 44.1 kHz 单声道 WAV），同时支持导入本地音频文件并直接试听/回放。
- 支持将个人日志导出为 **CSV**（UTF-8 BOM，兼容 Excel）或 **JSON**。

### 3. 已解锁鸟种（Life List）
- 首次在观鸟日志中记录某一鸟种时自动点亮解锁，并统计中国鸟类名录解锁进度、累计观测次数与科属分布；删除全部对应记录时同步撤销解锁状态。

### 4. 区域观鸟动态
- 切换全国 34 个省级行政区、任意 eBird 区域码/热点 ID 或经纬度坐标（`lat, lng`），查看昨日地区统计、**eBird 近期观测**与**eBird 珍稀记录**。
- 自动加载观测清单上下文（`Checklist` 观测人、观测时长、行进距离、现场备注与同清单伴生鸟种）。
- **严格区分「现场观测图」与「图鉴标准图」**：eBird 观测记录详情绝不混排图鉴通用配图，而是准确展示该次清单的媒体计数与原清单跳转入口；切换至 **区域现场实拍**（iNaturalist Geo Observations）可浏览区域内观测者当次目击拍摄上传的真实照片与录音。

### 5. 观鸟热点探索
- 拉取各省市近 30 天活跃观鸟热点，支持按历史累计鸟种数或最近活跃日期排序、按名称/ID 检索。
- 查看热点坐标、累计鸟种数、近期目击鸟种列表，并支持一键在指定热点新建观鸟日志。

### 6. 设置与按需缓存
- 在「设置」页面配置并在线验证个人 **eBird API Key**（优先保存至系统凭据管理器 `mygo.Secrets`，不支持时回退本地配置；亦可通过环境变量 `EBIRD_API_KEY` 注入）及可选 **Xeno-canto API v3 Key**。
- 支持默认省份设置、外观切换（跟随系统 / 浅色 / 深色）、64 MB 有上限 LRU 磁盘媒体缓存查看与一键清理。

---

## 快速开始

### 环境要求
- Go `1.27+`（纯 Go 构建，全程 `CGO_ENABLED=0`）
- Windows 10/11（推荐）或 Linux / macOS

### 本地运行与测试

```bash
# 运行全部单元测试与 UI 交互测试
CGO_ENABLED=0 go test -v ./...

# 运行 Go 与 MyGo 生命周期静态检查
go vet ./...
go tool mygo vet .

# 直接运行桌面应用
go run .
```

### 正式打包构建

```bash
# 构建当前平台正式发布可执行文件（产物输出到 build/ 目录）
go tool mygo build -skip-installer

# 交叉编译验证 Linux / Windows
GOOS=linux CGO_ENABLED=0 go build ./...
GOOS=windows CGO_ENABLED=0 go build ./...
```

---

## 目录结构

```text
Birdview/
├── main.go                 # 程序入口、单实例锁、原生菜单栏与主窗口生命周期
├── mygo.json               # MyGo 应用配置与打包信息
├── resources/icon.png      # 1024×1024 应用图标
└── internal/
    ├── model/              # 领域数据模型定义
    ├── store/              # 个人日志、鸟种解锁、设置与 64MB LRU 磁盘缓存
    ├── api/                # eBird API v2、iNaturalist、Xeno-canto v3 与中文维基客户端
    ├── audioctl/           # 流式音频播放器与 Windows 原生无 CGO 麦克风录音器
    └── appui/              # 基于 MyGo Native UI 的各功能视图与交互测试
```

## 数据存储与隐私说明

- 所有网络请求均通过 `mygo/netproxy` 自动遵循系统代理设置。
- 个人观鸟日志与导入/录制的图片、音频仅保存在本机用户数据目录（`PathUserData`），不上传至任何第三方服务器。
- API Key 优先由操作系统凭据管理器（Windows 凭据管理器 / macOS 钥匙串 / Linux Secret Service）加密保存，严禁内置或硬编码至仓库源码。
