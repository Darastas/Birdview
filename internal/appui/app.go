package appui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/audioctl"
	"github.com/Darastas/Birdview/internal/model"
	"github.com/Darastas/Birdview/internal/store"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const maxMemImages = 48

// AppState 管理整个应用的状态、后台异步数据加载与原生 UI 视图构建。
type AppState struct {
	win    *mygo.Window
	store    *store.Store
	cache    *store.Cache
	client   *api.Client
	player   *audioctl.PlayerController
	recorder audioctl.Recorder

	// 导航与全局布局状态
	page         string
	guideSplit   float32
	journalSplit float32
	regionSplit  float32
	hotspotSplit float32

	// 一次性 Toast 队列（在下一次视图构建时通过 c.Toast 发出）
	pendingToast string

	// 图片内存解码缓存（仅存已解码 ImageSource，不存 ui.Element）
	imgCache   map[string]ui.ImageSource
	imgOrder   []string
	imgLoading map[string]bool
	imgErr     map[string]string

	// 全局名录与物种索引
	allTaxa        []model.Taxon
	taxonByCode    map[string]model.Taxon
	chinaCodes     map[string]bool
	regionCodes    map[string]bool
	regionCodesFor string
	taxaLoading    bool
	taxaErr        string

	// 鸟类图鉴状态
	guideScope      int // 0=中国鸟类, 1=当前省份, 2=全球名录, 3=已解锁
	guideQuery      string
	guideOrder      string
	guideFamily     string
	guideOrders     []string
	guideFamilies   []string
	filteredTaxa    []model.Taxon
	guideList       ui.ListState
	guideSelected   int
	guideActiveCode string
	guideMediaMap   map[string]*model.SpeciesMedia
	guideMediaBusy  map[string]bool
	guidePhotoIndex int

	// 观鸟日志与已解锁鸟种状态
	journalSpeciesOpts []string
	journalQuery       string
	filteredEntries    []model.JournalEntry
	journalList        ui.ListState
	journalSelected    int
	journalActiveID    string
	journalPhotoIndex  int
	journalPrefillCode string
	journalModalOpen   bool
	jEditID            string
	jDraftSpecies      string
	jDraftObservedAt   string
	jDraftLocation     string
	jDraftCount        string
	jDraftNotes        string
	jDraftPhotos       []string
	jDraftAudio        string
	jDraftError        string
	jDeleteConfirm     bool
	jDeleteTargetID    string
	lifelistFilter     string
	lifelistSelected   string

	// 区域观鸟动态状态
	regionalRegion     string
	regSourceTab       int // 0=eBird近期观测, 1=eBird珍稀记录, 2=区域现场实拍
	regRegionLabel     string
	regBackLabel       string
	regCustomCode      string
	regObsBusy         bool
	regObsErr          string
	regObsList         []model.Observation
	regList            ui.ListState
	regSelected        int
	regActiveSubID     string
	regActiveObsID     string
	regActiveSpCode    string
	regStat            *model.RegionStat
	regChecklistMap    map[string]*model.ChecklistDetail
	regChecklistBusy   map[string]bool
	regPhotoBusy       bool
	regPhotoErr        string
	regFieldPhotos     []model.FieldPhotoObs
	regPhotoList       ui.ListState
	regPhotoSelected   int
	regPhotoGalleryIdx int

	// 观鸟热点探索状态
	hsRegionLabel string
	hsQuery       string
	hsSortMode    int // 0=累计鸟种数, 1=最近活跃日期
	hsBusy        bool
	hsErr         string
	hsAll         []model.Hotspot
	hsFiltered    []model.Hotspot
	hsList        ui.ListState
	hsSelected    int
	hsActiveLocID string
	hsObsMap      map[string][]model.Observation
	hsObsBusy     map[string]bool

	// 设置页面状态
	setEbirdKey     string
	setXenoKey      string
	setShowKeys     bool
	setRegionLabel  string
	setThemeMode    string
	setVerifyBusy   bool
	setVerifyMsg    string
	setVerifyOK     bool
	cacheStatsBytes int64
	cacheStatsFiles int
	localDataBytes  int64
}

// NewAppState 初始化应用主状态对象。
func NewAppState(st *store.Store, cache *store.Cache, client *api.Client) *AppState {
	cfg := st.Settings()
	a := &AppState{
		store:          st,
		cache:          cache,
		client:         client,
		player:         audioctl.NewPlayerController(),
		recorder:       audioctl.NewRecorder(),
		page:           "guide",
		guideSplit:     380,
		journalSplit:   380,
		regionSplit:    400,
		hotspotSplit:   380,
		imgCache:       make(map[string]ui.ImageSource),
		imgLoading:     make(map[string]bool),
		imgErr:         make(map[string]string),
		taxonByCode:    make(map[string]model.Taxon),
		chinaCodes:     make(map[string]bool),
		regionCodes:    make(map[string]bool),
		guideOrder:     "全部目",
		guideFamily:    "全部科",
		guideSelected:  -1,
		journalSelected: -1,
		guideMediaMap:  make(map[string]*model.SpeciesMedia),
		guideMediaBusy: make(map[string]bool),
		regionalRegion: cfg.DefaultRegion,
		regRegionLabel:   regionLabelByCode(cfg.DefaultRegion),
		regBackLabel:     "近 14 天",
		regSelected:      -1,
		regChecklistMap:  make(map[string]*model.ChecklistDetail),
		regChecklistBusy: make(map[string]bool),
		regPhotoSelected: -1,
		hsRegionLabel:    regionLabelByCode(cfg.DefaultRegion),
		hsSelected:       -1,
		hsObsMap:         make(map[string][]model.Observation),
		hsObsBusy:        make(map[string]bool),
	}
	a.syncSettingsDraft()
	a.refreshJournalList()
	a.refreshStorageStats()
	return a
}

// AttachWindow 绑定主窗口句柄，并触发启动后的首批异步数据拉取。
func (a *AppState) AttachWindow(win *mygo.Window) {
	a.win = win
	a.loadInitialData()
}

// SetPage 切换侧边栏主页面，供系统菜单和内部跳转按钮复用。
func (a *AppState) SetPage(page string) {
	a.page = page
	a.onPageActivated()
	if a.win != nil {
		a.win.Invalidate()
	}
}

func (a *AppState) onPageActivated() {
	switch a.page {
	case "settings":
		a.refreshStorageStats()
	case "regional":
		if a.win != nil && len(a.regObsList) == 0 && len(a.regFieldPhotos) == 0 && !a.regObsBusy && !a.regPhotoBusy && a.regObsErr == "" {
			a.LoadRegionalData()
		}
	case "hotspots":
		if a.win != nil && len(a.hsAll) == 0 && !a.hsBusy && a.hsErr == "" {
			a.LoadHotspots()
		}
	}
}

// RefreshCurrent 刷新当前页面的在线数据。
func (a *AppState) RefreshCurrent() {
	switch a.page {
	case "guide":
		a.loadTaxonomyAsync(true)
	case "regional":
		a.LoadRegionalData()
	case "hotspots":
		a.LoadHotspots()
	case "settings":
		a.refreshStorageStats()
		if a.win != nil {
			a.win.Invalidate()
		}
	}
}

// updateUI 在真实窗口运行时通过 Window.Update 切回主线程，无窗口测试时直接执行。
func (a *AppState) updateUI(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
		return
	}
	fn()
}

func (a *AppState) invalidateUI() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

func (a *AppState) syncSettingsDraft() {
	cfg := a.store.Settings()
	a.setEbirdKey = a.store.LoadEbirdKey()
	a.setXenoKey = a.store.LoadXenoKey()
	a.setRegionLabel = regionLabelByCode(cfg.DefaultRegion)
	a.setThemeMode = themeLabelFromMode(cfg.ThemeMode)
}

func (a *AppState) refreshStorageStats() {
	if a.cache != nil {
		a.cacheStatsBytes, a.cacheStatsFiles = a.cache.Stats()
	}
	if a.store != nil {
		a.localDataBytes = a.store.LocalStorageBytes()
	}
}

// SetRecorder 注入自定义录音器（主要用于单元测试模拟录音流程）。
func (a *AppState) SetRecorder(rec audioctl.Recorder) {
	a.recorder = rec
}

// View 是主窗口根视图函数（每帧在主线程执行，严禁做阻塞 I/O 或缓存 ui.Element）。
func (a *AppState) View(c *ui.Context) {
	t := applyAppTheme(c)

	if a.pendingToast != "" {
		c.Toast(a.pendingToast)
		a.pendingToast = ""
	}

	ui.Row(c).Fill().Children(func() {
		a.buildSidebar(c, t)

		ui.Column(c).Grow(1).FillHeight().Background(t.Background).Children(func() {
			ui.Box(c).Grow(1).FillWidth().Children(func() {
				switch a.page {
				case "guide":
					a.viewGuide(c, t)
				case "journal":
					a.viewJournal(c, t)
				case "lifelist":
					a.viewLifelist(c, t)
				case "regional":
					a.viewRegional(c, t)
				case "hotspots":
					a.viewHotspots(c, t)
				case "settings":
					a.viewSettings(c, t)
				default:
					a.viewGuide(c, t)
				}
			})
			a.buildAudioBar(c, t)
		})
	})

	a.viewJournalModal(c, t)
	a.viewJournalDeleteDialog(c)
}

func (a *AppState) buildSidebar(c *ui.Context, t *ui.Theme) {
	unlockedCount := len(a.store.UnlockedBirds())
	entriesCount := len(a.store.Entries())

	if ui.Sidebar(c, &a.page, func() {
		ui.SidebarSection(c, "资料与个人记录", nil, func() {
			ui.SidebarItem(c, "guide", iconBook, "鸟类图鉴").Children(func() {
				if len(a.allTaxa) > 0 {
					ui.Badge(c, fmt.Sprintf("%d", len(a.filteredTaxa)))
				}
			})
			ui.SidebarItem(c, "journal", iconJournal, "观鸟日志").Children(func() {
				if entriesCount > 0 {
					ui.Badge(c, fmt.Sprintf("%d", entriesCount))
				}
			})
			ui.SidebarItem(c, "lifelist", iconAward, "已解锁鸟种").Children(func() {
				if unlockedCount > 0 {
					ui.Badge(c, fmt.Sprintf("%d", unlockedCount))
				}
			})
		})

		ui.SidebarSection(c, "实时观测数据", nil, func() {
			ui.SidebarItem(c, "regional", iconGlobe, "区域观鸟动态")
			ui.SidebarItem(c, "hotspots", iconMapPin, "观鸟热点探索")
		})

		ui.SidebarSection(c, "系统", nil, func() {
			ui.SidebarItem(c, "settings", iconSettings, "设置")
		})
	}).Width(205).FillHeight().BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Changed() {
		a.onPageActivated()
	}
}

func (a *AppState) buildAudioBar(c *ui.Context, t *ui.Theme) {
	st := a.player.Status()
	if !st.Active {
		return
	}
	if st.Playing || st.Loading {
		c.After(120 * time.Millisecond)
	}

	ui.Row(c).FillWidth().Padding(10, 16).Gap(12).AlignItems(ui.Center).
		Background(t.Surface).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		if st.Loading {
			ui.Spinner(c).Label("加载音频中")
		} else {
			playIcon := iconPlay
			label := "继续播放"
			if st.Playing {
				playIcon = iconPause
				label = "暂停播放"
			}
			if ui.Button(c.Key("bar-toggle"), "").Label(label).Children(func() {
				ui.Icon(c, playIcon).FontSize(14)
			}).Clicked() {
				a.player.TogglePause()
			}
		}

		if ui.Button(c.Key("bar-stop"), "").Label("停止播放").Children(func() {
			ui.Icon(c, iconStop).FontSize(13)
		}).Clicked() {
			a.player.Stop()
		}

		ui.Column(c).Width(210).Gap(2).Children(func() {
			ui.Text(c, st.Title).Bold().SingleLine()
			sub := st.Subtitle
			if st.ErrMsg != "" {
				sub = st.ErrMsg
			}
			ui.Text(c, sub).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
		})

		// 进度滑块与时间码
		var frac float64
		if st.Duration > 0 {
			frac = float64(st.Position) / float64(st.Duration)
			if frac > 1 {
				frac = 1
			}
		}
		if st.Duration > 0 {
			ui.Text(c, formatDuration(st.Position)).FontSize(11.5).TextColor(t.TextMuted).FontFeatures("tnum")
			if ui.Slider(c.Key("bar-seek"), &frac, 0, 1).Label("播放进度").Grow(1).Changed() {
				a.player.SeekFraction(frac)
			}
			ui.Text(c, formatDuration(st.Duration)).FontSize(11.5).TextColor(t.TextMuted).FontFeatures("tnum")
		} else {
			ui.Spacer(c)
		}

		ui.Row(c).Gap(6).AlignItems(ui.Center).Width(130).Children(func() {
			ui.Icon(c, iconVolume).FontSize(15).TextColor(t.TextMuted)
			vol := st.Volume
			if ui.Slider(c.Key("bar-vol"), &vol, 0, 1).Label("音量").Grow(1).Changed() {
				a.player.SetVolume(vol)
			}
		})
	})
}

func (a *AppState) loadImageAsync(key, pathOrURL string, isLocalFile bool) {
	if key == "" || pathOrURL == "" {
		return
	}
	if _, ok := a.imgCache[key]; ok || a.imgLoading[key] {
		return
	}
	a.imgLoading[key] = true
	delete(a.imgErr, key)

	go func() {
		var data []byte
		var err error
		if isLocalFile {
			data, err = os.ReadFile(pathOrURL)
		} else {
			if a.cache != nil {
				if cached, ok := a.cache.GetBlob("img:" + pathOrURL); ok {
					data = cached
				}
			}
			if len(data) == 0 {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				data, err = a.client.FetchBytes(ctx, pathOrURL)
				cancel()
				if err == nil && a.cache != nil {
					_ = a.cache.PutBlob("img:"+pathOrURL, data)
				}
			}
		}

		var src ui.ImageSource
		if err == nil {
			src, err = ui.DecodeImage(data)
		}

		a.updateUI(func() {
			delete(a.imgLoading, key)
			if err != nil {
				a.imgErr[key] = err.Error()
				return
			}
			a.putMemImageLocked(key, src)
		})
	}()
}

func (a *AppState) putMemImageLocked(key string, src ui.ImageSource) {
	if _, exists := a.imgCache[key]; !exists {
		a.imgOrder = append(a.imgOrder, key)
	}
	a.imgCache[key] = src
	for len(a.imgOrder) > maxMemImages {
		oldest := a.imgOrder[0]
		a.imgOrder = a.imgOrder[1:]
		delete(a.imgCache, oldest)
	}
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

func regionLabelByCode(code string) string {
	for _, r := range api.ChinaRegions {
		if strings.EqualFold(r.Code, code) {
			return r.Name
		}
	}
	return api.ChinaRegions[0].Name
}

func regionCodeByLabel(label string) string {
	for _, r := range api.ChinaRegions {
		if r.Name == label {
			return r.Code
		}
	}
	return "CN-11"
}

func themeLabelFromMode(mode string) string {
	switch mode {
	case "light":
		return "浅色"
	case "dark":
		return "深色"
	default:
		return "跟随系统"
	}
}

func themeModeFromLabel(label string) string {
	switch label {
	case "浅色":
		return "light"
	case "深色":
		return "dark"
	default:
		return "system"
	}
}
