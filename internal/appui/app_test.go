package appui

import (
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
	if !tt.HasText("鸟类图鉴") || !tt.HasText("普通翠鸟") || !tt.HasText("佛法僧目 · 翠鸟科") {
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
