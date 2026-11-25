# Birdview

基于 **Go + MyGo v0.10.0 Native UI** 开发的跨平台移动端观鸟图鉴与个人观鸟日志应用（支持 **Android APK / AAB**、iOS 原生工程及桌面端手机视口预览）。所有物种名录、图鉴照片、鸟鸣参考录音、中文百科及区域实时观鸟数据均通过在线 API 按需流式加载，保持极轻量安装包体积。

## 核心功能

### 1. 鸟类图鉴
- 接入 **eBird API v2** 全球物种级分类名录（大陆简体中文 `zh_SIM`，支持全国、当前省份及已解锁范围切换）。
- 内置完整鸟纲目、科简体中文化索引，消除杂乱的英文分类混排。
- 图鉴列表支持按需解析并缓存物种代表缩略图；点击进入详情可浏览高清图集（支持左右滑动轮播与全屏双指缩放）、流式播放 **iNaturalist Sounds** / **Xeno-canto v3** 鸟鸣参考录音，并阅读 **中文维基百科** 物种介绍。

### 2. 区域探索与观鸟热点
- 集成手机 GPS 一键定位与全国 34 个省级行政区选择。
- **严格区分「现场实拍」与「图鉴资料」**：
  - **现场实拍**：拉取所在区域真实目击者拍摄并上传的现场摄影照片与现场录音，每张照片均严格对应当次观测记录。
  - **近期目击 / 珍稀记录 / 观鸟热点**：基于 **eBird API v2** 展示昨日统计、区域目击动态、完整清单上下文及近 30 天高鸟种数观鸟热点；eBird 记录详情页绝不混排图鉴通用图。

### 3. 观鸟日志与已解锁鸟种
- 支持拍摄现场照片或从系统相册选取图片（移动端原生调用 `plugins/imagepicker`，桌面端自动回退系统文件对话框）。
- **原生麦克风现场录音**：通过自主封装的 `internal/micplugin`（Android `AudioRecord` / iOS `AVFoundation` / Windows `winmm.dll`）实时录制 44.1 kHz 单声道 WAV 鸟鸣并附带声波电平反馈，同时支持导入外部音频文件。
- 首次记录鸟种自动解锁并累计观鸟成就进度与分科统计；支持左滑快速删除、CSV（UTF-8 BOM）/ JSON 导出与系统分享。

### 4. 设置与在线缓存
- 在「设置」页面配置并在线校验个人 **eBird API Key**（优先保存至系统凭据库 `mygo.Secrets` / Android Keystore / iOS Keychain）及可选 **Xeno-canto API v3 Key**。
- 支持自动 / 浅色 / 深色外观实时切换，以及 64 MB 媒体缓存与网络图片缓存的一键统计清理。

---

## 开发、测试与构建

### 环境要求
- Go `1.27+`
- Android 构建需配置 Android SDK、NDK 与 JDK 17（可通过 `go tool mygo doctor` 自检）

### 运行测试与检查

```bash
go test -v ./...
go vet ./...
go tool mygo vet .
```

### Android 打包构建

```bash
# 构建单架构快速验证（arm64）
go tool mygo build -platform android/arm64

# 正式构建全架构安装包（输出 build/android/birdview-2.0.0.apk 与 .aab）
go tool mygo build -platform android
```

### 桌面预览构建

```bash
go run .
go tool mygo build -skip-installer
```

---

## 目录结构

```text
Birdview/
├── main.go                 # 程序入口、窗口配置与生命周期
├── mygo.json               # MyGo v0.10.0 应用及 Android/iOS 原生打包配置
├── resources/              # 1024×1024 应用图标与 Android 自适应图标前景层
└── internal/
    ├── model/              # 领域数据模型与鸟纲目/科中文化词典
    ├── store/              # 个人观鸟日志、鸟种解锁、缩略图索引与 64MB LRU 缓存
    ├── api/                # eBird v2、iNaturalist、Xeno-canto v3 与中文维基客户端
    ├── audioctl/           # 跨平台流式播放器与麦克风录音控制
    ├── micplugin/          # Android (Kotlin) / iOS (Swift) 原生麦克风录音插件
    └── appui/              # 基于 MyGo v0.10.0 ui/mobile 的移动原生界面与交互测试
```

## 数据安全与隐私

- 网络请求统一经由 `mygo/netproxy` 遵循系统代理。
- 个人日志与录制/导入的照片和音频仅保存在本机应用数据目录，不上传至任何第三方服务器。
- API Key 通过系统安全凭据库加密存放，严禁将任何真实 API Key 写入或提交至代码仓库。
