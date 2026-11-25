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
	app := NewAppState(st, cache, api.NewClient())
	app.SetRecorder(&fakeRecorder{})
	taxa := []model.Taxon{
		{SpeciesCode: "comkin1", ComName: "普通翠鸟", SciName: "Alcedo atthis", Order: "Coraciiformes", FamilyComName: "翠鸟科"},
		{SpeciesCode: "creibi1", ComName: "朱鹮", SciName: "Nipponia nippon", Order: "Pelecaniformes", FamilyComName: "鹮科"},
		{SpeciesCode: "hoopoe", ComName: "戴胜", SciName: "Upupa epops", Order: "Bucerotiformes", FamilyComName: "戴胜科"},
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

func TestGuideSearchSelectAndSidebarNavigation(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 1180, 760)

	if !tt.HasText("鸟类图鉴") || !tt.HasText("普通翠鸟") {
		t.Fatalf("expected guide view with 普通翠鸟, texts=%q", tt.Texts())
	}

	if err := tt.Click("搜索鸟种"); err != nil {
		t.Fatal(err)
	}
	tt.Type("朱鹮")
	if len(app.filteredTaxa) != 1 || app.filteredTaxa[0].SpeciesCode != "creibi1" {
		t.Fatalf("expected filteredTaxa to contain only creibi1, got %+v", app.filteredTaxa)
	}
	if err := tt.Click("朱鹮"); err != nil {
		t.Fatal(err)
	}
	if app.guideActiveCode != "creibi1" {
		t.Fatalf("expected active species creibi1, got %q", app.guideActiveCode)
	}

	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if app.page != "settings" || !tt.HasText("数据源与 API 密钥") {
		t.Fatalf("expected settings page, page=%q", app.page)
	}
	if err := tt.Click("eBird API Key"); err != nil {
		t.Fatal(err)
	}
	tt.Type("unit-test-key")
	if err := tt.Click("保存设置"); err != nil {
		t.Fatal(err)
	}
	if got := app.store.LoadEbirdKey(); got != "unit-test-key" {
		t.Fatalf("expected saved key unit-test-key, got %q", got)
	}
}

func TestJournalCreateRecordAndUnlockSpecies(t *testing.T) {
	app := newTestAppState(t)
	tt := ui.NewTester(app.View, 1200, 780)

	// 从图鉴页直接点击“记入观鸟日志”
	if err := tt.Click("记入观鸟日志"); err != nil {
		t.Fatal(err)
	}
	if !app.journalModalOpen {
		t.Fatal("expected journal modal open")
	}

	if err := tt.Click("观测地点"); err != nil {
		t.Fatal(err)
	}
	tt.Type("奥林匹克森林公园")

	if err := tt.Click("文字记录"); err != nil {
		t.Fatal(err)
	}
	tt.Type("芦苇丛附近停栖")

	// 测试现场录音按键（开始 -> 停止生成 WAV）
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
		t.Fatalf("expected wav recording path, got %q", app.jDraftAudio)
	}

	if err := tt.Click("保存记录"); err != nil {
		t.Fatal(err)
	}
	if app.journalModalOpen {
		t.Fatal("expected modal closed after save")
	}
	if !app.store.IsUnlocked("comkin1") {
		t.Fatal("expected comkin1 unlocked after saving journal entry")
	}

	// 切换到“已解锁鸟种”页面检查是否已点亮
	if err := tt.Click("已解锁鸟种"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("普通翠鸟") || !tt.HasText("奥林匹克森林公园") {
		t.Fatalf("expected unlocked bird in lifelist, texts=%q", tt.Texts())
	}
}
