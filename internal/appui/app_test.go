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
