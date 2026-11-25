package appui

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/audioctl"
	"github.com/Darastas/Birdview/internal/model"
	"github.com/Darastas/Birdview/internal/store"
	"github.com/egoist/mygo/ui"
)

type fakeRecorder struct {
	recording bool
	startedAt time.Time
}

func (f *fakeRecorder) Start() error {
	f.recording = true
	f.startedAt = time.Now()
	return nil
}

func (f *fakeRecorder) Stop(destWavPath string) error {
	f.recording = false
	pcm := make([]byte, 4410*2)
	return audioctl.WriteWAV(destWavPath, pcm, 44100, 1, 16)
}

func (f *fakeRecorder) Cancel() {
	f.recording = false
}

func (f *fakeRecorder) Status() audioctl.RecorderStatus {
	return audioctl.RecorderStatus{
		Recording: f.recording,
		Elapsed:   2 * time.Second,
		Level:     0.45,
	}
}

func advanceUI(tt *ui.Tester, d time.Duration) {
	for range d / (16 * time.Millisecond) {
		tt.Advance(16 * time.Millisecond)
	}
}

func newTestAppState(t *testing.T) *AppState {
	t.Helper()
	st, err := store.New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := store.NewCache(t.TempDir(), 4*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	app := NewAppState(st, cache, api.NewClient(), &MobileSession{})
	app.SetRecorder(&fakeRecorder{})
	taxa := []model.Taxon{
		{SpeciesCode: "comkin1", ComName: "普通翠鸟", SciName: "Alcedo atthis", Order: "Coraciiformes", FamilySciName: "Alcedinidae", FamilyComName: "Kingfishers"},
		{SpeciesCode: "creibi1", ComName: "朱鹮", SciName: "Nipponia nippon", Order: "Pelecaniformes", FamilySciName: "Threskiornithidae", FamilyComName: "Ibises and Spoonbills"},
		{SpeciesCode: "hoopoe", ComName: "戴胜", SciName: "Upupa epops", Order: "Bucerotiformes", FamilySciName: "Upupidae", FamilyComName: "Hoopoes"},
	}
	app.guideMediaMap["comkin1"] = &model.SpeciesMedia{
		SciName: "Alcedo atthis",
		ComName: "普通翠鸟",
		Wiki:    &model.WikiSummary{Title: "普通翠鸟", Extract: "常见小型翠鸟。", Source: "中文维基百科"},
	}
	app.guideMediaMap["creibi1"] = &model.SpeciesMedia{
		SciName: "Nipponia nippon",
		ComName: "朱鹮",
		Wiki:    &model.WikiSummary{Title: "朱鹮", Extract: "珍稀涉禽。", Source: "中文维基百科"},
	}
	app.SetTaxonomy(taxa, map[string]bool{"comkin1": true, "creibi1": true, "hoopoe": true}, nil, "CN-11")
	return app
}

func TestMobileGuideSearchRouteAndSettings(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 402, 844)

	// 验证分类名已全部转为纯中文（绝不泄漏英文 Kingfishers）
	if !tt.HasText("图鉴") || !tt.HasText("普通翠鸟") || !tt.HasText("佛法僧目 · 翠鸟科") {
		t.Fatalf("expected localized mobile guide view, texts=%q", tt.Texts())
	}
	if tt.HasText("Kingfishers") {
		t.Fatalf("expected English family name localized to Chinese, texts=%q", tt.Texts())
	}

	if err := tt.Click("搜索鸟种"); err != nil {
		t.Fatal(err)
	}
	tt.Type("朱鹮")
	if len(app.filteredTaxa) != 1 || app.filteredTaxa[0].SpeciesCode != "creibi1" {
		t.Fatalf("expected filteredTaxa to contain only creibi1, got %+v", app.filteredTaxa)
	}

	// 点击鸟种卡片推入移动端详情路由，推进动画完成后点击返回
	if err := tt.Click("朱鹮"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 600*time.Millisecond)
	if app.router.Path() != "/guide/creibi1" {
		t.Fatalf("expected pushed guide detail route /guide/creibi1, got %q", app.router.Path())
	}
	if err := tt.Click("返回"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 600*time.Millisecond)
	if app.router.Path() != "/guide" {
		t.Fatalf("expected route back to /guide, got %q", app.router.Path())
	}

	// 切换到底部「设置」Tab -> 打开 API 密钥抽屉 -> 保存 Key
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 300*time.Millisecond)
	if app.session.Tab != 3 || app.router.Path() != "/settings" {
		t.Fatalf("expected settings tab/route, tab=%d path=%q", app.session.Tab, app.router.Path())
	}
	if err := tt.Click("eBird API Key"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 700*time.Millisecond)
	if !app.setKeySheetOpen {
		t.Fatal("expected API key sheet open")
	}
	if err := tt.Click("eBird API Key 输入框"); err != nil {
		t.Fatal(err)
	}
	tt.Type("unit-test-key")
	if err := tt.Click("保存设置"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 1500*time.Millisecond)
	if got := app.store.LoadEbirdKey(); got != "unit-test-key" {
		t.Fatalf("expected saved key unit-test-key, got %q", got)
	}
}

func TestMobileJournalFabRecordAndUnlock(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 402, 844)

	if err := tt.Click("日志"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 300*time.Millisecond)
	if app.router.Path() != "/journal" {
		t.Fatalf("expected /journal route, got %q", app.router.Path())
	}

	// 点击右下角 FAB 打开新建日志 Sheet
	if err := tt.Click("新建观鸟日志"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 700*time.Millisecond)
	if !app.journalSheetOpen {
		t.Fatal("expected journal composer sheet open")
	}

	if err := tt.Click("鸟种"); err != nil {
		t.Fatal(err)
	}
	tt.Type("普通翠鸟")

	if err := tt.Click("观测地点"); err != nil {
		t.Fatal(err)
	}
	tt.Type("奥林匹克森林公园")

	if err := tt.Click("文字记录"); err != nil {
		t.Fatal(err)
	}
	tt.Type("荷塘边停留")

	// 测试圆形麦克风按钮录音流程（开始 -> 停止生成 WAV）
	if err := tt.Click("开始录音"); err != nil {
		t.Fatal(err)
	}
	if !app.recorder.Status().Recording {
		t.Fatal("expected recorder active")
	}
	if err := tt.Click("停止录音"); err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(app.jDraftAudio) != ".wav" {
		t.Fatalf("expected wav audio path, got %q", app.jDraftAudio)
	}

	if err := tt.Click("保存记录"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 1500*time.Millisecond)
	if app.journalSheetOpen {
		t.Fatal("expected composer sheet closed after save")
	}
	if !app.store.IsUnlocked("comkin1") {
		t.Fatal("expected comkin1 unlocked after saving journal entry")
	}

	// 切换到「已解锁图谱」子标签核查展示
	if err := tt.Click("已解锁图谱"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("普通翠鸟") {
		t.Fatalf("expected unlocked bird in grid, texts=%q", tt.Texts())
	}
}

func TestMobileExploreChecklistIsolationAndHotspots(t *testing.T) {
	app := newTestAppState(t)

	app.regObsList = []model.Observation{
		{
			SpeciesCode:     "hoopoe",
			ComName:         "戴胜",
			SciName:         "Upupa epops",
			LocID:           "L12170807",
			LocName:         "沙河水库北岸",
			ObsDt:           "2026-11-25 08:30",
			HowMany:         2,
			Lat:             40.136,
			Lng:             116.313,
			ObsValid:        true,
			SubID:           "S999001",
			ObsID:           "OBS999001",
			UserDisplayName: "北京观鸟者",
			HasRichMedia:    true,
		},
	}
	app.regChecklistMap["S999001"] = &model.ChecklistDetail{
		SubID:           "S999001",
		UserDisplayName: "北京观鸟者",
		DurationHrs:     1.5,
		NumSpecies:      2,
		Obs: []model.ChecklistObs{
			{
				SpeciesCode: "hoopoe",
				ObsID:       "OBS999001",
				HowManyStr:  "2",
				Comments:    "草坪边缘觅食现场记录",
				MediaCounts: map[string]int{"P": 2},
			},
		},
	}
	app.hsAll = []model.Hotspot{
		{LocID: "L12170807", LocName: "沙河水库北岸", Subnational1Code: "CN-11", LatestObsDt: "2026-11-25", NumSpeciesAllTime: 268},
		{LocID: "L5817879", LocName: "奥林匹克森林公园", Subnational1Code: "CN-11", LatestObsDt: "2026-11-24", NumSpeciesAllTime: 310},
	}
	app.applyHotspotFilterAndSort()

	tt := ui.NewTester(app.View, 402, 844)

	// 1. 进入「探索」->「近期目击」-> 点击记录进入清单详情，校验不混用图鉴图
	if err := tt.Click("探索"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 300*time.Millisecond)
	if err := tt.Click("近期目击"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("戴胜"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 600*time.Millisecond)
	if app.router.Path() != "/explore/check/S999001" {
		t.Fatalf("expected /explore/check/S999001, got %q", app.router.Path())
	}
	if !tt.HasText("本次观测现场媒体（非图鉴标准图）") || !tt.HasText("草坪边缘觅食现场记录") {
		t.Fatalf("expected non-guide notice and checklist notes, texts=%q", tt.Texts())
	}
	if _, loadedGuideForHoopoe := app.guideMediaMap["hoopoe"]; loadedGuideForHoopoe {
		t.Fatal("explore checklist detail must NOT load guide species media")
	}

	// 2. 返回探索列表，切换到「观鸟热点」-> 打开「奥林匹克森林公园」并点「在此处写日志」
	if err := tt.Click("返回"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 600*time.Millisecond)
	if err := tt.Click("观鸟热点"); err != nil {
		t.Fatal(err)
	}
	if app.hsFiltered[0].LocName != "奥林匹克森林公园" {
		t.Fatalf("expected hotspot sorted by all-time species desc, got %+v", app.hsFiltered)
	}
	if err := tt.Click("奥林匹克森林公园"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 600*time.Millisecond)
	if app.router.Path() != "/explore/spot/L5817879" {
		t.Fatalf("expected /explore/spot/L5817879, got %q", app.router.Path())
	}
	if err := tt.Click("在此处写日志"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 700*time.Millisecond)
	if !app.journalSheetOpen || app.jDraftLocation != "奥林匹克森林公园" {
		t.Fatalf("expected journal sheet open with hotspot location, open=%v loc=%q", app.journalSheetOpen, app.jDraftLocation)
	}
}

func TestCrystalGlassAndSkeletonRender(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 402, 844)
	advanceUI(tt, 64*time.Millisecond)

	// 验证光学液态玻璃底栏、顶部渐变遮罩与网格渲染
	img := tt.Image()
	if img.Bounds().Dx() != 402 || img.Bounds().Dy() != 844 {
		t.Fatalf("unexpected guide render bounds: %v", img.Bounds())
	}

	// 验证探索加载骨架屏渲染
	if err := tt.Click("探索"); err != nil {
		t.Fatal(err)
	}
	app.regPhotoBusy = true
	advanceUI(tt, 64*time.Millisecond)
	if img := tt.Image(); img.Bounds().Dx() != 402 {
		t.Fatalf("unexpected explore skeleton render bounds: %v", img.Bounds())
	}

	// 验证设置页移除冗余小字分组标题后可正常渲染
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 64*time.Millisecond)
	for _, removedHeading := range []string{"接口与密钥", "偏好设置", "存储与备份"} {
		if tt.HasText(removedHeading) {
			t.Fatalf("expected removed settings section heading %q to be absent, texts=%q", removedHeading, tt.Texts())
		}
	}
	if !tt.HasText("eBird API Key") || !tt.HasText("外观主题") || !tt.HasText("关于 Birdview") {
		t.Fatalf("expected clean settings rows intact, texts=%q", tt.Texts())
	}
	if img := tt.Image(); img.Bounds().Dy() != 844 {
		t.Fatalf("unexpected settings render bounds: %v", img.Bounds())
	}
}

func TestGuideScopeAndFilterSearch(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 402, 844)

	// 1. 通过正交地点按钮切换图鉴范围至「全球」
	if err := tt.Click("切换图鉴地区范围"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 750*time.Millisecond)
	if !app.guideScopeSheet {
		t.Fatal("expected guideScopeSheet to open")
	}
	if err := tt.Click("全球"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 800*time.Millisecond)
	if app.session.GuideScope != 2 || app.guideScopeSheet {
		t.Fatalf("expected GuideScope=2 and sheet closed, got scope=%d open=%v", app.session.GuideScope, app.guideScopeSheet)
	}

	// 2. 紧凑型「全部 | 已解锁」开关切换
	if err := tt.Click("已解锁"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 100*time.Millisecond)
	if !app.session.GuideOnlyUnlocked || len(app.filteredTaxa) != 0 || !tt.HasText("暂无匹配的鸟种") {
		t.Fatalf("expected empty unlocked list, onlyUnlocked=%v count=%d", app.session.GuideOnlyUnlocked, len(app.filteredTaxa))
	}
	if err := tt.Click("全部"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 100*time.Millisecond)
	if app.session.GuideOnlyUnlocked || len(app.filteredTaxa) != 3 {
		t.Fatalf("expected all 3 taxa restored, onlyUnlocked=%v count=%d", app.session.GuideOnlyUnlocked, len(app.filteredTaxa))
	}

	// 3. 目科筛选抽屉内实时搜索
	if err := tt.Click("目科筛选"); err != nil {
		if err2 := tt.Click("目科"); err2 != nil {
			t.Fatalf("click filter button failed: %v / %v", err, err2)
		}
	}
	advanceUI(tt, 750*time.Millisecond)
	if !app.guideFilterSheet {
		t.Fatal("expected guideFilterSheet to open")
	}
	if err := tt.Click("搜索目科"); err != nil {
		t.Fatal(err)
	}
	tt.Type("翠鸟")
	advanceUI(tt, 100*time.Millisecond)
	if app.guideFilterQuery != "翠鸟" || !tt.HasText("按科筛选") || !tt.HasText("翠鸟科") || tt.HasText("按目筛选") {
		t.Fatalf("expected guide filter search narrowed to 按科筛选 -> 翠鸟科, query=%q texts=%q", app.guideFilterQuery, tt.Texts())
	}
	if err := tt.Click("翠鸟科"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("完成"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 800*time.Millisecond)
	if app.guideFilterSheet || len(app.filteredTaxa) != 1 || app.filteredTaxa[0].SpeciesCode != "comkin1" {
		t.Fatalf("expected filtered single kingfisher, open=%v taxa=%+v", app.guideFilterSheet, app.filteredTaxa)
	}
}

func TestPhotoLightboxContainAndReset(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 402, 844)

	photos := []model.MediaPhoto{
		{URL: "http://127.0.0.1:1/bird-1.jpg", ThumbURL: "http://127.0.0.1:1/bird-1-t.jpg", Source: "iNaturalist", Attribution: "Tester 1"},
		{URL: "http://127.0.0.1:1/bird-2.jpg", ThumbURL: "http://127.0.0.1:1/bird-2-t.jpg", Source: "iNaturalist", Attribution: "Tester 2"},
	}

	app.openNetworkPhotosViewer("普通翠鸟", photos, 1)
	tt.Frame()
	advanceUI(tt, 120*time.Millisecond)
	if !app.zoomOpen || app.zoomPager.Page != 1 || len(app.zoomPhotos) != 2 {
		t.Fatalf("expected lightbox open at page 1, open=%v page=%d len=%d", app.zoomOpen, app.zoomPager.Page, len(app.zoomPhotos))
	}
	if !tt.HasText("普通翠鸟") || !tt.HasText("第 2 / 2 张") {
		t.Fatalf("expected lightbox title & counter, texts=%q", tt.Texts())
	}
	if img := tt.Image(); img.Bounds().Dx() != 402 {
		t.Fatalf("unexpected lightbox skeleton render bounds: %v", img.Bounds())
	}

	if err := tt.Click("关闭大图查看"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 120*time.Millisecond)
	if app.zoomOpen {
		t.Fatal("expected lightbox closed")
	}

	// 重新打开不同图片时页码与缩放状态必须复位，不可残留上次索引 1
	app.openNetworkPhotosViewer("戴胜", photos[:1], 0)
	tt.Frame()
	advanceUI(tt, 120*time.Millisecond)
	if !app.zoomOpen || app.zoomPager.Page != 0 || !tt.HasText("第 1 / 1 张") {
		t.Fatalf("expected lightbox page reset to 0, open=%v page=%d texts=%q", app.zoomOpen, app.zoomPager.Page, tt.Texts())
	}
}

func TestFieldCommentUnescapeAndColorSwatch(t *testing.T) {
	// 1. HTML 实体先解码为中文逗号，绝不误判为色号
	plainSegs := parseCommentColorSegments(api.CleanEBirdText("黑身&#xff0c;头灰白"))
	if len(plainSegs) != 1 || plainSegs[0].HasColor || plainSegs[0].Text != "黑身，头灰白" {
		t.Fatalf("expected plain unescaped segment without color swatch, got %+v", plainSegs)
	}

	// 2. 真实十六进制色号提取与 RGB 转换
	colorSegs := parseCommentColorSegments("背羽 #2F4F4F，头顶 #FFF")
	var swatches []commentSegment
	for _, s := range colorSegs {
		if s.HasColor {
			swatches = append(swatches, s)
		}
	}
	if len(swatches) != 2 ||
		swatches[0].Text != "#2F4F4F" || swatches[0].Color != ui.RGB(0x2F, 0x4F, 0x4F) ||
		swatches[1].Text != "#FFF" || swatches[1].Color != ui.RGB(255, 255, 255) {
		t.Fatalf("unexpected parsed color swatches: %+v", swatches)
	}

	// 3. 清单详情页渲染含 HTML 实体 + 真实色号混合备注无异常，且文本解码显示
	app := newTestAppState(t)
	app.regObsList = []model.Observation{
		{
			SpeciesCode: "hoopoe",
			ComName:     "戴胜",
			SciName:     "Upupa epops",
			LocName:     "翠湖湿地",
			ObsDt:       "2026-11-25 10:15",
			HowMany:     1,
			SubID:       "S888001",
		},
	}
	app.regChecklistMap["S888001"] = &model.ChecklistDetail{
		SubID:           "S888001",
		UserDisplayName: "tester",
		NumSpecies:      1,
		Obs: []model.ChecklistObs{
			{
				SpeciesCode: "hoopoe",
				Comments:    "黑身&#xff0c;头灰白，冠羽 #D97706，腹羽 #FFFFFF",
			},
		},
	}
	tt := ui.NewTester(app.View, 402, 844)
	if err := tt.Click("探索"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("近期目击"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("戴胜"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 600*time.Millisecond)
	if !tt.HasText("黑身，头灰白") || !tt.HasText("#D97706") || !tt.HasText("#FFFFFF") {
		t.Fatalf("expected decoded comment and swatch labels visible, texts=%q", tt.Texts())
	}
	if tt.HasText("&#xff0c;") {
		t.Fatalf("expected raw html entity unescaped, texts=%q", tt.Texts())
	}
	if img := tt.Image(); img.Bounds().Dx() != 402 {
		t.Fatalf("unexpected checklist comment swatch render bounds: %v", img.Bounds())
	}
}

func TestExploreImmediateSkeletonAndConditionalScrim(t *testing.T) {
	blockCh := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-blockCh:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(blockCh)

	app := newTestAppState(t)
	app.client.INatBaseURL = srv.URL
	app.client.EBirdBaseURL = srv.URL
	_ = app.store.SaveEbirdKey("dummy-ebird-key")

	// 预填旧地区缓存（模拟北京旧数据）
	app.regFieldPhotos = []model.FieldPhotoObs{
		{
			ID:         1001,
			ComName:    "老北京现场鸟",
			SciName:    "Cyanopica cyanus",
			Observer:   "OldObs",
			PlaceGuess: "北京",
			Photos:     []model.MediaPhoto{{URL: srv.URL + "/p1.jpg", ThumbURL: srv.URL + "/t1.jpg"}},
		},
	}
	app.regObsList = []model.Observation{
		{SpeciesCode: "hoopoe", ComName: "旧地区戴胜", LocName: "奥森", ObsDt: "2026-11-25 08:00", HowMany: 1, SubID: "S_OLD"},
	}
	app.regStat = &model.RegionStat{NumChecklists: 10, NumSpecies: 20, NumContributors: 5}
	app.hsAll = []model.Hotspot{{LocID: "L_OLD", LocName: "旧地区热点", NumSpeciesAllTime: 99}}
	app.hsFiltered = append([]model.Hotspot(nil), app.hsAll...)

	tt := ui.NewTester(app.View, 402, 844)
	if err := tt.Click("探索"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 48*time.Millisecond)
	if !tt.HasText("老北京现场鸟") {
		t.Fatalf("expected old region photo visible before switch, texts=%q", tt.Texts())
	}

	// 1. 切换地区应立即清空旧数据并进入灰色骨架卡片加载，顶部不出现转圈动画
	app.regCustomCode = ""
	app.regRegionLabel = "四川 (CN-51)"
	app.resetExploreForRegionChange()
	advanceUI(tt, 32*time.Millisecond)

	if !app.regPhotoBusy || len(app.regFieldPhotos) != 0 || len(app.regObsList) != 0 || app.regStat != nil || len(app.hsFiltered) != 0 {
		t.Fatalf("expected immediate stale cache clear and busy=true, busy=%v photos=%d obs=%d hs=%d",
			app.regPhotoBusy, len(app.regFieldPhotos), len(app.regObsList), len(app.hsFiltered))
	}
	if tt.HasText("老北京现场鸟") || tt.HasText("加载中") {
		t.Fatalf("expected no stale card and no header spinner after region change, texts=%q", tt.Texts())
	}
	if img := tt.Image(); img.Bounds().Dx() != 402 {
		t.Fatalf("unexpected photo skeleton render bounds: %v", img.Bounds())
	}

	// 2. 切换上方 4 个选项卡时也立即进入骨架卡片，不显示转圈
	for _, tab := range []string{"近期目击", "珍稀记录", "观鸟热点", "现场实拍"} {
		if err := tt.Click(tab); err != nil {
			t.Fatal(err)
		}
		advanceUI(tt, 32*time.Millisecond)
		if tt.HasText("加载中") {
			t.Fatalf("expected no spinner after switching to %q, texts=%q", tab, tt.Texts())
		}
		if img := tt.Image(); img.Bounds().Dy() != 844 {
			t.Fatalf("unexpected skeleton render bounds on %q: %v", tab, img.Bounds())
		}
	}

	// 3. 验证图鉴/探索实拍/日志在首行(0)与滚动离开首行(40)条件遮罩均可正常渲染
	if err := tt.Click("图鉴"); err != nil {
		t.Fatal(err)
	}
	app.guideScroll.Y = 0
	advanceUI(tt, 32*time.Millisecond)
	_ = tt.Image()
	app.guideScroll.Y = 40
	advanceUI(tt, 32*time.Millisecond)
	_ = tt.Image()

	if err := tt.Click("探索"); err != nil {
		t.Fatal(err)
	}
	app.session.ExploreSubTab = 0
	app.regPhotoBusy = false
	app.regFieldPhotos = []model.FieldPhotoObs{{ID: 2, ComName: "测试鸟", SciName: "Test", Photos: []model.MediaPhoto{{URL: srv.URL + "/x.jpg"}}}}
	app.exploreScroll[0].Y = 0
	advanceUI(tt, 32*time.Millisecond)
	_ = tt.Image()
	app.exploreScroll[0].Y = 40
	advanceUI(tt, 32*time.Millisecond)
	_ = tt.Image()

	if err := tt.Click("日志"); err != nil {
		t.Fatal(err)
	}
	app.journalTimelineScroll.Y = 0
	advanceUI(tt, 32*time.Millisecond)
	_ = tt.Image()
	if err := tt.Click("已解锁图谱"); err != nil {
		t.Fatal(err)
	}
	app.journalLifelistScroll.Y = 0
	advanceUI(tt, 32*time.Millisecond)
	_ = tt.Image()
	app.journalLifelistScroll.Y = 40
	advanceUI(tt, 32*time.Millisecond)
	_ = tt.Image()
}

func touchSwipe(tt *ui.Tester, x0, y0, x1, y1 float32, steps int) {
	tt.FingerDown(x0, y0)
	for i := 1; i <= steps; i++ {
		tt.Advance(16 * time.Millisecond)
		f := float32(i) / float32(steps)
		tt.FingerMove(x0+(x1-x0)*f, y0+(y1-y0)*f)
	}
	tt.Advance(16 * time.Millisecond)
	tt.FingerUp(x1, y1)
}

func TestAndroidImmersiveSafeAreaAndNoBottomMask(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 402, 844)
	tt.SetInsets(44, 0, 24, 0)
	advanceUI(tt, 64*time.Millisecond)

	// 验证沉浸式状态栏顶部适配：头部筛选按钮应在 44dp 状态栏之下且不过度留白
	mr, ok := tt.Find("目科筛选")
	if !ok {
		mr, ok = tt.Find("目科")
	}
	if !ok || mr.Y < 44 || mr.Y > 92 {
		t.Fatalf("expected immersive header button below 44dp status bar without double gap, ok=%v rect=%+v", ok, mr)
	}

	// 验证悬浮底部导航底座贴合底部手势区上方（不再叠加安全区导致上浮 40dp+）
	sr, ok := tt.Find("设置")
	if !ok {
		t.Fatalf("expected bottom dock item 设置")
	}
	// dockBottom = 24+4 = 28, height = 62 -> dock y=[754, 816]
	if sr.Y+sr.H < 775 || sr.Y+sr.H > 824 {
		t.Fatalf("expected dock near gesture bar y=[754..816], got item rect=%+v", sr)
	}

	// 进入鸟种详情页验证 Hero 区全出血顶到状态栏上方
	if err := tt.Click("普通翠鸟"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 500*time.Millisecond)
	if app.router.Path() != "/guide/comkin1" {
		t.Fatalf("expected detail route, got %q", app.router.Path())
	}
	if img := tt.Image(); img.Bounds().Dx() != 402 || img.Bounds().Dy() != 844 {
		t.Fatalf("unexpected detail bounds: %v", img.Bounds())
	}
}

func TestTouchFlingCatchContinueAndBrakeNoClick(t *testing.T) {
	app := newTestAppState(t)
	taxa := make([]model.Taxon, 24)
	china := make(map[string]bool, 24)
	for i := range taxa {
		code := string([]byte{'s', 'p', byte('a' + i/26), byte('a' + i%26)})
		taxa[i] = model.Taxon{
			SpeciesCode:   code,
			ComName:       "测试鸟" + code,
			SciName:       "Avis " + code,
			Order:         "Passeriformes",
			FamilySciName: "Paridae",
		}
		china[code] = true
	}
	app.SetTaxonomy(taxa, china, nil, "CN-11")

	tt := ui.NewTester(app.View, 402, 844)
	tt.SetInsets(44, 0, 24, 0)
	advanceUI(tt, 64*time.Millisecond)

	// 1. 第一次向上滑动触发惯性滚动（fling）
	touchSwipe(tt, 200, 620, 200, 260, 8)
	tt.Advance(32 * time.Millisecond)
	yAfterSwipe1 := app.guideScroll.Y
	tt.Advance(48 * time.Millisecond)
	yMidFling := app.guideScroll.Y
	if yMidFling <= yAfterSwipe1 || yAfterSwipe1 < 80 {
		t.Fatalf("expected active fling scrolling downward, yAfterSwipe1=%.1f yMidFling=%.1f", yAfterSwipe1, yMidFling)
	}

	// 2. 惯性滚动途中手指再次按住并继续上滑，不能锁死或丢失拖拽
	touchSwipe(tt, 200, 560, 200, 320, 6)
	advanceUI(tt, 80*time.Millisecond)
	if app.guideScroll.Y <= yMidFling+30 {
		t.Fatalf("expected continued scroll after catching fling, before=%.1f after=%.1f", yMidFling, app.guideScroll.Y)
	}

	// 3. 再次触发 fling 后单指轻点急停，必须停下滚动且不能误点进入详情页
	touchSwipe(tt, 200, 620, 200, 280, 6)
	tt.Advance(32 * time.Millisecond)
	tt.FingerDown(110, 460)
	tt.Advance(20 * time.Millisecond)
	tt.FingerUp(110, 460)
	advanceUI(tt, 80*time.Millisecond)

	if app.router.Path() != "/guide" {
		t.Fatalf("expected tap-to-brake not to click card underneath, path=%q", app.router.Path())
	}
	yBraked := app.guideScroll.Y
	advanceUI(tt, 64*time.Millisecond)
	if app.guideScroll.Y != yBraked {
		t.Fatalf("expected fling stopped after brake tap, got %.1f -> %.1f", yBraked, app.guideScroll.Y)
	}
}

func TestTouchModalNoAutoFocusAndNoKeyboardOnClose(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 402, 844)
	tt.SetInsets(44, 0, 24, 0)
	advanceUI(tt, 64*time.Millisecond)

	before := tt.TextInputShowCount()

	// 1. 打开目科弹窗，搜索框绝不能自动聚焦或拉起输入法
	if err := tt.Click("目科"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 800*time.Millisecond)
	if !app.guideFilterSheet {
		t.Fatal("expected guideFilterSheet open")
	}
	if active, _ := tt.TextInputActive(); active || tt.TextInputShowCount() != before {
		t.Fatalf("expected no auto-focus/IME on sheet open, active=%v showCount=%d before=%d", active, tt.TextInputShowCount(), before)
	}

	// 2. 关闭目科弹窗，也绝不能拉起输入法
	if err := tt.Click("完成"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 800*time.Millisecond)
	if active, _ := tt.TextInputActive(); app.guideFilterSheet || active || tt.TextInputShowCount() != before {
		t.Fatalf("expected no IME popup after sheet close, open=%v active=%v showCount=%d before=%d", app.guideFilterSheet, active, tt.TextInputShowCount(), before)
	}

	// 3. 打开并关闭地区弹窗同样不能触发输入法
	if err := tt.Click("切换图鉴地区范围"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 800*time.Millisecond)
	if err := tt.Click("中国"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 800*time.Millisecond)
	if tt.TextInputShowCount() != before {
		t.Fatalf("expected scope sheet open/close not to show keyboard, got %d want %d", tt.TextInputShowCount(), before)
	}

	// 4. 明确用手指轻点搜索框才允许激活输入法；之后打开再关闭任意弹窗也不能把键盘重新弹起
	sr, ok := tt.Find("搜索鸟种")
	if !ok {
		t.Fatal("expected 搜索鸟种 search input")
	}
	cx, cy := sr.X+sr.W/2, sr.Y+sr.H/2
	tt.FingerDown(cx, cy)
	tt.Advance(20 * time.Millisecond)
	tt.FingerUp(cx, cy)
	advanceUI(tt, 64*time.Millisecond)
	afterTap := tt.TextInputShowCount()
	if active, _ := tt.TextInputActive(); !active || afterTap <= before {
		t.Fatalf("expected explicit touch tap to activate IME, active=%v afterTap=%d before=%d", active, afterTap, before)
	}
	if err := tt.Click("目科"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 800*time.Millisecond)
	if err := tt.Click("完成"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 800*time.Millisecond)
	if active, _ := tt.TextInputActive(); active || tt.TextInputShowCount() != 0 {
		t.Fatalf("expected closing modal after prior search never re-shows keyboard, active=%v show=%d", active, tt.TextInputShowCount())
	}
}

func TestUnloadedHeroAndLightboxPagerSingleStepSwipe(t *testing.T) {
	app := newTestAppState(t)
	app.guideMediaMap["comkin1"].Photos = []model.MediaPhoto{
		{URL: "http://127.0.0.1:1/u1.jpg", ThumbURL: "http://127.0.0.1:1/t1.jpg", Source: "iNaturalist", Attribution: "A1"},
		{URL: "http://127.0.0.1:1/u2.jpg", ThumbURL: "http://127.0.0.1:1/t2.jpg", Source: "iNaturalist", Attribution: "A2"},
		{URL: "http://127.0.0.1:1/u3.jpg", ThumbURL: "http://127.0.0.1:1/t3.jpg", Source: "iNaturalist", Attribution: "A3"},
		{URL: "http://127.0.0.1:1/u4.jpg", ThumbURL: "http://127.0.0.1:1/t4.jpg", Source: "iNaturalist", Attribution: "A4"},
	}

	tt := ui.NewTester(app.View, 402, 844)
	tt.SetInsets(44, 0, 24, 0)
	if err := tt.Click("普通翠鸟"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 600*time.Millisecond)

	if app.guideGallery.Page != 0 || !tt.HasText("1 / 4") {
		t.Fatalf("expected initial page 0 and 1 / 4, page=%d texts=%q", app.guideGallery.Page, tt.Texts())
	}

	// 图片均未加载出来时，每次横向滑动严格切换一页：0->1->2->3
	for want := 1; want <= 3; want++ {
		touchSwipe(tt, 350, 200, 50, 200, 8)
		advanceUI(tt, 420*time.Millisecond)
		wantLabel := string([]byte{byte('1' + want)}) + " / 4"
		if app.guideGallery.Page != want || !tt.HasText(wantLabel) {
			t.Fatalf("swipe %d: expected Page=%d and label %q, got Page=%d texts=%q", want, want, wantLabel, app.guideGallery.Page, tt.Texts())
		}
	}

	// 验证极快长距离滑动也不跨越多页（3->2->1）
	touchSwipe(tt, 20, 200, 385, 200, 3)
	advanceUI(tt, 420*time.Millisecond)
	if app.guideGallery.Page != 2 || !tt.HasText("3 / 4") {
		t.Fatalf("expected single-page fast flick back to 2 (3 / 4), got Page=%d", app.guideGallery.Page)
	}

	// 手势取消应干净吸附回整页而非悬在半页
	tt.FingerDown(280, 200)
	tt.Advance(16 * time.Millisecond)
	tt.FingerMove(180, 200)
	tt.Advance(16 * time.Millisecond)
	tt.FingerCancel()
	advanceUI(tt, 420*time.Millisecond)
	if app.guideGallery.Page != 2 || !tt.HasText("3 / 4") {
		t.Fatalf("expected canceled swipe snapped cleanly back to page 2, got Page=%d", app.guideGallery.Page)
	}

	// 点击 Hero 进入灯箱并滑向第 4 张，页码与摄影署名信息应无延迟立即同步
	if err := tt.Click("点击放大图片"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 120*time.Millisecond)
	if !app.zoomOpen || app.zoomPager.Page != 2 || !tt.HasText("第 3 / 4 张") {
		t.Fatalf("expected lightbox at page 2 (3/4), open=%v page=%d", app.zoomOpen, app.zoomPager.Page)
	}

	touchSwipe(tt, 350, 420, 50, 420, 7)
	tt.Advance(32 * time.Millisecond)
	if app.zoomPager.Page != 3 || !tt.HasText("第 4 / 4 张") || !tt.HasText("来源：iNaturalist · 署名：A4") {
		t.Fatalf("expected lightbox info updated immediately after swipe, page=%d texts=%q", app.zoomPager.Page, tt.Texts())
	}
}

func TestAndroidBackClosesLightboxOnlyNotDetailRoute(t *testing.T) {
	app := newTestAppState(t)
	app.guideMediaMap["comkin1"].Photos = []model.MediaPhoto{
		{URL: "http://127.0.0.1:1/b1.jpg", Source: "iNaturalist", Attribution: "U1"},
		{URL: "http://127.0.0.1:1/b2.jpg", Source: "iNaturalist", Attribution: "U2"},
	}

	tt := ui.NewTester(app.View, 402, 844)
	tt.SetInsets(44, 0, 24, 0)
	if err := tt.Click("普通翠鸟"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 600*time.Millisecond)
	if err := tt.Click("点击放大图片"); err != nil {
		t.Fatal(err)
	}
	advanceUI(tt, 120*time.Millisecond)
	if !app.zoomOpen || app.router.Path() != "/guide/comkin1" {
		t.Fatalf("expected lightbox open on detail page, zoom=%v path=%q", app.zoomOpen, app.router.Path())
	}

	// 第一次手机侧边返回：仅关闭灯箱，绝不能退回图鉴主页
	if handled := tt.MobileBack(); !handled {
		t.Fatal("expected MobileBack handled by lightbox")
	}
	advanceUI(tt, 150*time.Millisecond)
	if app.zoomOpen {
		t.Fatal("expected lightbox closed after first MobileBack")
	}
	if app.router.Path() != "/guide/comkin1" {
		t.Fatalf("expected to remain on /guide/comkin1 after closing lightbox, got %q", app.router.Path())
	}

	// 第二次返回才退回图鉴主页
	if handled := tt.MobileBack(); !handled {
		t.Fatal("expected MobileBack handled by router back")
	}
	advanceUI(tt, 600*time.Millisecond)
	if app.router.Path() != "/guide" {
		t.Fatalf("expected /guide after second MobileBack, got %q", app.router.Path())
	}
}
