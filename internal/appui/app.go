package appui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/audioctl"
	"github.com/Darastas/Birdview/internal/model"
	"github.com/Darastas/Birdview/internal/store"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/location"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"
	"github.com/egoist/mygo/ui/netimage"
)

const maxMemLocalImages = 24

var tabRoutes = []string{"/guide", "/explore", "/journal", "/settings"}

// MobileSession 保存轻量移动端页面切换状态，供 mygo.KeepState 热恢复。
type MobileSession struct {
	Tab               int
	GuideScope        int
	GuideOnlyUnlocked bool
	GuideQuery        string
	ExploreSubTab     int
	JournalSubTab     int
	JournalQuery      string
	LifelistFilter    string
}

// AppState 管理移动端路由、数据加载、音频控制与原生 UI 视图状态。
type AppState struct {
	win      *mygo.Window
	store    *store.Store
	cache    *store.Cache
	client   *api.Client
	player   *audioctl.PlayerController
	recorder audioctl.Recorder

	session *MobileSession
	router  *ui.Router

	pendingToast string

	// 本地用户照片内存缓存（网络图已交由 ui/netimage 统一缓存与按需下载）
	localImgCache   map[string]ui.ImageSource
	localImgOrder   []string
	localImgLoading map[string]bool
	localImgErr     map[string]string

	// 全局鸟类名录与缩略图缓存
	allTaxa        []model.Taxon
	taxonByCode    map[string]model.Taxon
	chinaCodes     map[string]bool
	regionCodes    map[string]bool
	regionCodesFor string
	taxaLoading    bool
	taxaErr        string
	thumbMap       map[string]string
	thumbTried     map[string]bool
	thumbSem       chan struct{}
	thumbPending   int

	// 鸟类图鉴状态
	guideOrder       string
	guideFamily      string
	guideOrders      []string
	guideFamilies    []string
	filteredTaxa     []model.Taxon
	guideList        ui.ListState
	guideScroll      ui.ScrollState
	guideRefresh     ui.RefreshState
	guideFilterSheet bool
	guideFilterQuery string
	guideScopeSheet  bool
	guideActiveCode  string
	guideMediaMap    map[string]*model.SpeciesMedia
	guideMediaBusy   map[string]bool
	guideGallery     mobile.PageState

	// 区域探索（现场实拍 + eBird 观测 + 热点）
	regionalRegion     string
	regRegionLabel     string
	regCustomCode      string
	regBackLabel       string
	regRegionSheetOpen bool
	regRegionQuery     string
	regRegionTarget    string // "explore" 或 "settings"
	regGPSBusy         bool
	regGPSNote         string

	regObsBusy       bool
	regObsErr        string
	regObsList       []model.Observation
	regObsSeq        uint64
	regActiveSubID   string
	regActiveObsID   string
	regSelectedObs   *model.Observation
	regStat          *model.RegionStat
	regChecklistMap  map[string]*model.ChecklistDetail
	regChecklistBusy map[string]bool

	regPhotoBusy   bool
	regPhotoErr    string
	regFieldPhotos []model.FieldPhotoObs
	regPhotoSeq    uint64
	regPhotoPager  mobile.PageState
	regActivePhoto *model.FieldPhotoObs
	exploreRefresh ui.RefreshState
	exploreScroll  [4]ui.ScrollState

	hsRegionLabel string
	hsQuery       string
	hsSortMode    int // 0=鸟种最多, 1=最近活跃
	hsBusy        bool
	hsErr         string
	hsAll         []model.Hotspot
	hsFiltered    []model.Hotspot
	hsSeq         uint64
	hsActiveSpot  *model.Hotspot
	hsObsMap      map[string][]model.Observation
	hsObsBusy     map[string]bool

	// 观鸟日志 & 已解锁图谱
	journalSpeciesOpts    []string
	filteredEntries       []model.JournalEntry
	journalTimelineScroll ui.ScrollState
	journalLifelistScroll ui.ScrollState
	journalRefresh        ui.RefreshState
	journalActiveEntry *model.JournalEntry
	journalDetailPager mobile.PageState
	journalSheetOpen   bool
	journalPhotoSheet  bool
	exportSheetOpen    bool
	jEditID            string
	jDraftSpecies      string
	jDraftObservedAt   string
	jDraftLocation     string
	jDraftLat          float64
	jDraftLng          float64
	jDraftCount        int
	jDraftNotes        string
	jDraftPhotos       []string
	jDraftAudio        string
	jDraftError        string
	jDraftGPSBusy      bool
	jRecBusy           bool
	jDeleteConfirm     bool
	jDeleteTargetID    string

	// 全屏照片缩放查看器
	zoomOpen       bool
	zoomTitle      string
	zoomPhotos     []model.MediaPhoto
	zoomLocalPaths []string
	zoomPager      mobile.PageState
	zoomLastPage   int
	zoomState      mobile.ZoomState

	// 设置状态
	setEbirdKey       string
	setXenoKey        string
	setShowKeys       bool
	setKeySheetOpen   bool
	setAboutSheetOpen bool
	setRegionLabel    string
	setThemeMode      string
	setVerifyBusy     bool
	setVerifyMsg      string
	setVerifyOK       bool
	cacheStatsBytes   int64
	cacheStatsFiles   int
	localDataBytes    int64
	lastBarStyle      mygo.BarStyle
}

func (a *AppState) hasModalOpen() bool {
	return a.guideFilterSheet || a.guideScopeSheet || a.regRegionSheetOpen ||
		a.journalSheetOpen || a.journalPhotoSheet || a.exportSheetOpen ||
		a.jDeleteConfirm || a.zoomOpen || a.setKeySheetOpen || a.setAboutSheetOpen
}

// NewAppState 初始化移动端应用状态。
func NewAppState(st *store.Store, cache *store.Cache, client *api.Client, sess *MobileSession) *AppState {
	if sess == nil {
		sess = &MobileSession{}
	}
	if sess.Tab < 0 || sess.Tab >= len(tabRoutes) {
		sess.Tab = 0
	}
	if sess.GuideScope == 3 {
		sess.GuideScope = 0
		sess.GuideOnlyUnlocked = true
	}
	cfg := st.Settings()
	var thumbs map[string]string
	if cache != nil {
		thumbs = cache.LoadThumbIndex()
	} else {
		thumbs = make(map[string]string)
	}

	a := &AppState{
		store:            st,
		cache:            cache,
		client:           client,
		player:           audioctl.NewPlayerController(),
		recorder:         audioctl.NewRecorder(),
		session:          sess,
		router:           ui.NewRouter(tabRoutes[sess.Tab]),
		localImgCache:    make(map[string]ui.ImageSource),
		localImgLoading:  make(map[string]bool),
		localImgErr:      make(map[string]string),
		taxonByCode:      make(map[string]model.Taxon),
		chinaCodes:       make(map[string]bool),
		regionCodes:      make(map[string]bool),
		thumbMap:         thumbs,
		thumbTried:       make(map[string]bool),
		thumbSem:         make(chan struct{}, 12),
		guideOrder:       "全部目",
		guideFamily:      "全部科",
		guideMediaMap:    make(map[string]*model.SpeciesMedia),
		guideMediaBusy:   make(map[string]bool),
		regionalRegion:   cfg.DefaultRegion,
		regRegionLabel:   regionLabelByCode(cfg.DefaultRegion),
		regBackLabel:     "14天",
		regRegionTarget:  "explore",
		regChecklistMap:  make(map[string]*model.ChecklistDetail),
		regChecklistBusy: make(map[string]bool),
		hsRegionLabel:    regionLabelByCode(cfg.DefaultRegion),
		hsObsMap:         make(map[string][]model.Observation),
		hsObsBusy:        make(map[string]bool),
		jDraftCount:      1,
	}
	a.router.Animation = &ui.Animation{Spring: ui.SpringOptions{
		Duration:  300 * time.Millisecond,
		Bounce:    0,
		Precision: 0.5,
	}}
	a.syncSettingsDraft()
	a.refreshJournalList()
	a.refreshStorageStats()
	return a
}

// AttachWindow 绑定主窗口句柄，并触发启动后的异步名录拉取。
func (a *AppState) AttachWindow(win *mygo.Window) {
	a.win = win
	a.loadTaxonomyAsync(false)
	a.onTabActivated(a.session.Tab)
}

// SetRecorder 注入自定义录音器（测试用途）。
func (a *AppState) SetRecorder(rec audioctl.Recorder) {
	a.recorder = rec
}

func (a *AppState) hapticSelection() {
	if a.win != nil {
		mygo.Haptics.Selection()
	}
}

func (a *AppState) hapticSuccess() {
	if a.win != nil {
		mygo.Haptics.Success()
	}
}

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

func (a *AppState) isRootRoute() bool {
	p := a.router.Path()
	for _, r := range tabRoutes {
		if p == r {
			return true
		}
	}
	return false
}

// SelectTab 切换底部主导航标签（供底栏及测试调用）。
func (a *AppState) SelectTab(idx int) {
	if idx < 0 || idx >= len(tabRoutes) {
		return
	}
	a.session.Tab = idx
	a.router.Replace(tabRoutes[idx])
	a.hapticSelection()
	a.onTabActivated(idx)
	a.invalidateUI()
}

func (a *AppState) onTabActivated(idx int) {
	switch idx {
	case 1: // 探索
		if a.win != nil && !a.regObsBusy && !a.regPhotoBusy && !a.hsBusy {
			if len(a.regFieldPhotos) == 0 && len(a.regObsList) == 0 && len(a.hsAll) == 0 {
				a.LoadExploreData()
			}
		}
	case 3: // 设置
		a.refreshStorageStats()
	}
}

func (a *AppState) goBack() {
	if a.router.CanGoBack() {
		a.router.Back()
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

// View 是移动端根视图函数（遵守每帧只声明构建的规则）。
func (a *AppState) View(c *ui.Context) {
	t := applyAppTheme(c)
	modalOpen := a.hasModalOpen()

	wantBarStyle := mygo.BarStyleDark
	curPath := a.router.Path()
	if a.zoomOpen || t.Dark || (!modalOpen && (strings.HasPrefix(curPath, "/guide/") || strings.HasPrefix(curPath, "/explore/photo/"))) {
		wantBarStyle = mygo.BarStyleLight
	}
	if wantBarStyle != a.lastBarStyle {
		a.lastBarStyle = wantBarStyle
		mygo.SystemBars.SetStyle(wantBarStyle)
	}

	if a.pendingToast != "" {
		c.Toast(a.pendingToast)
		a.pendingToast = ""
	}

	rootView := a.isRootRoute()

	ui.Box(c).Fill().Background(t.Background).Children(func() {
		applyCanvasBackground(ui.Box(c).Fill(), t).Children(func() {
			ui.Box(c).Fill().Children(func() {
				a.router.View(c, func(r *ui.Route) {
					switch {
					case r.Match("/guide"):
						r.Title("图鉴")
						a.viewGuideHome(c, t)
					case r.Match("/guide/{code}"):
						r.Title("鸟种详情")
						a.viewGuideDetailPage(c, t, r.Param("code"))
					case r.Match("/explore"):
						r.Title("探索")
						a.viewExploreHome(c, t)
					case r.Match("/explore/photo/{id}"):
						r.Title("现场实拍")
						a.viewExplorePhotoDetail(c, t, r.Param("id"))
					case r.Match("/explore/check/{id}"):
						r.Title("观鸟清单")
						a.viewExploreChecklistDetail(c, t, r.Param("id"))
					case r.Match("/explore/spot/{id}"):
						r.Title("观鸟热点")
						a.viewExploreHotspotDetail(c, t, r.Param("id"))
					case r.Match("/journal"):
						r.Title("日志")
						a.viewJournalHome(c, t)
					case r.Match("/journal/{id}"):
						r.Title("日志详情")
						a.viewJournalDetailPage(c, t, r.Param("id"))
					case r.Match("/settings"):
						r.Title("设置")
						a.viewSettingsHome(c, t)
					default:
						r.Title("图鉴")
						a.viewGuideHome(c, t)
					}
				})
			})

			a.buildMiniAudioBar(c, t, rootView, modalOpen)

			if rootView && floatingGlassDock(c, &a.session.Tab, t, modalOpen) {
				a.SelectTab(a.session.Tab)
			}
		})
	})

	a.viewRegionPickerSheet(c, t)
	a.viewGuideScopeSheet(c, t)
	a.viewGuideFilterSheet(c, t)
	a.viewJournalComposerSheet(c, t)
	a.viewJournalPhotoActionSheet(c)
	a.viewJournalExportActionSheet(c)
	a.viewJournalDeleteDialog(c)
	a.viewSettingsKeySheet(c, t)
	a.viewSettingsAboutSheet(c, t)
	a.viewPhotoZoomOverlay(c, t)
}

func (a *AppState) buildMiniAudioBar(c *ui.Context, t *ui.Theme, aboveDock, obscured bool) {
	st := a.player.Status()
	if !st.Active {
		return
	}
	if st.Playing || st.Loading {
		c.After(120 * time.Millisecond)
	}
	_, _, bottom, _ := c.SafeArea()
	barBottom := max(float32(10), bottom+4)
	if aboveDock && bottom <= keyboardInsetThreshold {
		barBottom = dockClearancePad(c, 6)
	}

	card := ui.Column(c.Key("mini-audio-bar")).Absolute().Left(16).Right(16).Bottom(barBottom).
		Radius(28).Clip()
	if obscured {
		if t.Dark {
			card.Background(ui.Hex("#161B26")).Border(1, ui.RGBA(255, 255, 255, 0.10))
		} else {
			card.Background(ui.Hex("#FFFFFF")).Border(1, ui.RGBA(15, 23, 42, 0.08))
		}
	} else {
		card.Material(floatingBarGlass(t.Dark))
	}
	if t.Dark {
		card.Shadow(0, 10, 26, -4, ui.RGBA(0, 0, 0, 0.5))
	} else {
		card.Shadow(0, 8, 22, -4, ui.RGBA(15, 23, 42, 0.14))
	}

	card.Children(func() {
		ui.Row(c).FillWidth().Padding(10, 12).Gap(10).AlignItems(ui.Center).Children(func() {
			if st.Loading {
				ui.Box(c).Size(36, 36).Center().Children(func() {
					ui.Spinner(c).Label("加载音频中")
				})
			} else {
				ic := iconPlay
				lbl := "播放"
				if st.Playing {
					ic = iconPause
					lbl = "暂停"
				}
				playBtn := applySculptedPrimaryStyle(
					ui.Box(c.Key("mini-toggle")).Size(36, 36).Radius(18).Center().Shrink(0).
						Role(ui.RoleButton).Label(lbl).Children(func() {
						ui.Icon(c, ic).FontSize(15).TextColor(t.AccentText)
					}),
					t,
				)
				if playBtn.Clicked() {
					a.player.TogglePause()
				}
			}

			ui.Column(c).Grow(1).Basis(0).Gap(1).Children(func() {
				ui.Text(c, st.Title).FontSize(14).FontWeight(500).SingleLine()
				sub := st.Subtitle
				if st.ErrMsg != "" {
					sub = st.ErrMsg
				} else if st.Duration > 0 {
					sub = fmt.Sprintf("%s / %s · %s", formatDuration(st.Position), formatDuration(st.Duration), st.Subtitle)
				}
				ui.Text(c, sub).FontSize(12).TextColor(secondaryTextColor(t)).SingleLine()
			})

			if circleIconBtn(c, "mini-stop", "关闭播放器", iconClose, softPillColor(t), t.TextMuted, 28).Clicked() {
				a.player.Stop()
			}
		})
	})
}

func (a *AppState) queueSpeciesThumb(tx model.Taxon) {
	if a.win == nil || tx.SpeciesCode == "" || tx.SciName == "" {
		return
	}
	if _, ok := a.thumbMap[tx.SpeciesCode]; ok || a.thumbTried[tx.SpeciesCode] {
		return
	}
	a.thumbTried[tx.SpeciesCode] = true

	select {
	case a.thumbSem <- struct{}{}:
		go func(code, sciName string) {
			defer func() { <-a.thumbSem }()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			url, err := a.client.FetchSpeciesThumb(ctx, sciName)
			cancel()

			a.updateUI(func() {
				if err == nil && url != "" {
					a.thumbMap[code] = url
					a.thumbPending++
					if a.cache != nil && a.thumbPending >= 6 {
						a.thumbPending = 0
						snapshot := copyStringMap(a.thumbMap)
						go func() { _ = a.cache.SaveThumbIndex(snapshot) }()
					}
				}
			})
		}(tx.SpeciesCode, tx.SciName)
	default:
		// 并发上限已满时允许之后再次进入视口重试
		delete(a.thumbTried, tx.SpeciesCode)
	}
}

func copyStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// loadLocalImageAsync 按移动端合理分辨率解码用户本地相册/相机照片。
func (a *AppState) loadLocalImageAsync(path string) {
	if path == "" {
		return
	}
	if _, ok := a.localImgCache[path]; ok || a.localImgLoading[path] {
		return
	}
	a.localImgLoading[path] = true
	delete(a.localImgErr, path)

	go func(p string) {
		data, err := os.ReadFile(p)
		var src ui.ImageSource
		if err == nil {
			if bmp, derr := ui.DecodeBitmapScaled(data, 1200); derr == nil {
				src = bmp
			} else {
				src, err = ui.DecodeImage(data)
			}
		}
		a.updateUI(func() {
			delete(a.localImgLoading, p)
			if err != nil {
				a.localImgErr[p] = err.Error()
				return
			}
			if _, exists := a.localImgCache[p]; !exists {
				a.localImgOrder = append(a.localImgOrder, p)
			}
			a.localImgCache[p] = src
			for len(a.localImgOrder) > maxMemLocalImages {
				oldest := a.localImgOrder[0]
				a.localImgOrder = a.localImgOrder[1:]
				delete(a.localImgCache, oldest)
			}
		})
	}(path)
}

func (a *AppState) openNetworkPhotosViewer(title string, photos []model.MediaPhoto, idx int) {
	if len(photos) == 0 {
		return
	}
	page := max(0, min(idx, len(photos)-1))
	a.zoomTitle = title
	a.zoomPhotos = photos
	a.zoomLocalPaths = nil
	a.zoomPager = mobile.PageState{Page: page}
	a.zoomLastPage = page
	a.zoomState.Reset()
	a.zoomOpen = true
}

func (a *AppState) openLocalPhotosViewer(title string, paths []string, idx int) {
	if len(paths) == 0 {
		return
	}
	page := max(0, min(idx, len(paths)-1))
	a.zoomTitle = title
	a.zoomPhotos = nil
	a.zoomLocalPaths = paths
	a.zoomPager = mobile.PageState{Page: page}
	a.zoomLastPage = page
	a.zoomState.Reset()
	a.zoomOpen = true
}

func (a *AppState) viewPhotoZoomOverlay(c *ui.Context, t *ui.Theme) {
	if !a.zoomOpen {
		return
	}
	count := max(len(a.zoomPhotos), len(a.zoomLocalPaths))
	if count == 0 {
		a.zoomOpen = false
		return
	}
	if a.zoomPager.Page != a.zoomLastPage {
		a.zoomLastPage = a.zoomPager.Page
		a.zoomState.Reset()
	}
	if len(a.zoomPhotos) > 1 {
		var urls []string
		for _, idx := range []int{a.zoomPager.Page, a.zoomPager.Page - 1, a.zoomPager.Page + 1} {
			if idx >= 0 && idx < len(a.zoomPhotos) {
				if u := a.zoomPhotos[idx].URL; u != "" {
					urls = append(urls, u)
				}
			}
		}
		if len(urls) > 0 {
			netimage.Preload(urls...)
		}
	}
	top, _, bottom, _ := c.SafeArea()

	ui.Overlay(c, func() {
		bg := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(ui.Hex("#000000")).Modal()
		if bg.OverlayShortcut(0, ui.KeyEscape) || bg.OverlayShortcut(0, ui.KeyBack) {
			a.zoomOpen = false
		}
		bg.Children(func() {
			ui.Column(c).Fill().Children(func() {
				// 顶部关闭栏
				ui.Row(c).FillWidth().Padding(top+10, 16, 10, 16).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Grow(1).Gap(2).Children(func() {
						ui.Text(c, a.zoomTitle).FontSize(16).FontWeight(600).TextColor(ui.RGB(255, 255, 255)).SingleLine()
						ui.Textf(c, "第 %d / %d 张", a.zoomPager.Page+1, count).FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.72))
					})
					if circleIconBtn(c, "zoom-close", "关闭大图查看", iconClose, ui.RGBA(255, 255, 255, 0.16), ui.RGB(255, 255, 255), 36).Clicked() {
						a.zoomOpen = false
					}
				})

				// 可滑动与缩放主体
				mobile.PageView(c, &a.zoomPager, count, func(i int) {
					if i == a.zoomPager.Page {
						mobile.Zoomable(c, &a.zoomState, mobile.ZoomOptions{Min: 1, Max: 4, DoubleTap: 2}, func() {
							a.renderZoomSlide(c, i)
						}).Fill()
					} else {
						a.renderZoomSlide(c, i)
					}
				}).Grow(1).Basis(0).FillWidth()

				// 底部摄影署名说明
				ui.Column(c).FillWidth().Padding(8, 16, bottom+14, 16).Gap(6).AlignItems(ui.Center).Children(func() {
					if count > 1 {
						mobile.PageDots(c, a.zoomPager.Page, count)
					}
					if len(a.zoomPhotos) > 0 && a.zoomPager.Page < len(a.zoomPhotos) {
						p := a.zoomPhotos[a.zoomPager.Page]
						ui.Textf(c, "来源：%s · 署名：%s", p.Source, p.Attribution).
							FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.75)).TextAlign(ui.Center)
					}
				})
			})
		})
	})
}

func (a *AppState) renderZoomSlide(c *ui.Context, i int) {
	if len(a.zoomPhotos) > 0 && i < len(a.zoomPhotos) {
		p := a.zoomPhotos[i]
		src, _ := netimage.Load(c, p.URL, netimage.Options{MaxSide: 1800})
		if src == nil && p.ThumbURL != "" {
			src, _ = netimage.Load(c, p.ThumbURL, netimage.Options{MaxSide: 500})
		}
		if src != nil {
			ui.Image(c, src).Fit(ui.Contain).Fill().Label(a.zoomTitle)
			return
		}
		darkTheme := ui.DarkTheme()
		skeletonBirdCard(c, 16, 42, darkTheme)
		return
	}
	if len(a.zoomLocalPaths) > 0 && i < len(a.zoomLocalPaths) {
		lp := a.zoomLocalPaths[i]
		a.loadLocalImageAsync(lp)
		if src, ok := a.localImgCache[lp]; ok {
			ui.Image(c, src).Fit(ui.Contain).Fill().Label("本地观鸟照片")
		} else {
			darkTheme := ui.DarkTheme()
			skeletonBirdCard(c, 16, 42, darkTheme)
		}
	}
}

func (a *AppState) fillGPSLocationAsync(target string) {
	if target == "explore" {
		if a.regGPSBusy {
			return
		}
		a.regGPSBusy = true
		a.regGPSNote = ""
	} else {
		if a.jDraftGPSBusy {
			return
		}
		a.jDraftGPSBusy = true
	}
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		fix, err := location.Current(ctx)
		cancel()

		a.updateUI(func() {
			if target == "explore" {
				a.regGPSBusy = false
				if err != nil {
					if errors.Is(err, errors.ErrUnsupported) {
						a.regGPSNote = "桌面端暂不支持手机 GPS，请在下方选择省份或直接输入“纬度,经度”"
					} else {
						a.regGPSNote = fmt.Sprintf("定位失败: %v", err)
					}
					return
				}
				a.regCustomCode = fmt.Sprintf("%.4f,%.4f", fix.Latitude, fix.Longitude)
				a.regRegionLabel = fmt.Sprintf("定位附近 (%.2f, %.2f)", fix.Latitude, fix.Longitude)
				a.regRegionSheetOpen = false
				a.resetExploreForRegionChange()
			} else {
				a.jDraftGPSBusy = false
				if err != nil {
					if errors.Is(err, errors.ErrUnsupported) {
						a.pendingToast = "桌面端暂不支持 GPS 定位，可手动填写观测地点"
					} else {
						a.pendingToast = fmt.Sprintf("定位失败: %v", err)
					}
					return
				}
				a.jDraftLat = fix.Latitude
				a.jDraftLng = fix.Longitude
				if strings.TrimSpace(a.jDraftLocation) == "" {
					a.jDraftLocation = fmt.Sprintf("%.4f, %.4f", fix.Latitude, fix.Longitude)
				}
				a.hapticSuccess()
				a.pendingToast = "已填充当前坐标"
			}
		})
	}()
}

// ShareOrExportJournal 优先支持移动端系统分享，导出 CSV/JSON 时兼容 Android 文档保存与桌面保存对话框。
func (a *AppState) ShareOrExportJournal(mode string) {
	if mode == "share" {
		entries := a.store.Entries()
		unlocked := a.store.UnlockedBirds()
		text := fmt.Sprintf("Birdview 观鸟记录：已解锁 %d 个鸟种，共记录 %d 条观测日志。", len(unlocked), len(entries))
		var files []string
		if len(entries) > 0 {
			latest := entries[0]
			text = fmt.Sprintf("我在 %s 观察到了 %s（%s）×%d，已累计解锁 %d 种鸟类。",
				fallbackText(latest.Location, "野外现场"), latest.ComName, latest.SciName, max(1, latest.Count), len(unlocked))
			if len(latest.PhotoPaths) > 0 {
				files = append(files, latest.PhotoPaths[0])
			}
		}
		go func() {
			err := mygo.Share(mygo.ShareOptions{Text: text, Files: files})
			if err != nil {
				if errors.Is(err, errors.ErrUnsupported) {
					a.ShareOrExportJournal("csv")
					return
				}
				a.updateUI(func() {
					a.pendingToast = fmt.Sprintf("分享失败: %v", err)
				})
			}
		}()
		return
	}

	defaultName := "birdview_journal." + mode
	go func() {
		dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      a.win,
			Title:       "导出观鸟日志",
			DefaultPath: defaultName,
		})
		if err != nil || dest == "" {
			return
		}
		if mode == "csv" {
			err = a.store.ExportJournalCSV(dest)
		} else {
			err = a.store.ExportJournalJSON(dest)
		}
		a.updateUI(func() {
			if err != nil {
				a.pendingToast = fmt.Sprintf("导出失败: %v", err)
			} else {
				a.pendingToast = "已导出观鸟日志至 " + filepath.Base(dest)
			}
		})
	}()
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

func regionShortNameByCode(code string) string {
	lbl := regionLabelByCode(code)
	if idx := strings.Index(lbl, " ("); idx > 0 {
		return lbl[:idx]
	}
	return lbl
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
