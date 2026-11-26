package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Darastas/Birdview/internal/model"
)

func TestEbirdClientAndLocale(t *testing.T) {
	var seenLocale string
	var seenToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/ref/taxonomy/ebird":
			seenLocale = r.URL.Query().Get("locale")
			w.Write([]byte(`[{"speciesCode":"comkin1","comName":"普通翠鸟","sciName":"Alcedo atthis","category":"species","order":"Coraciiformes","familyComName":"翠鸟科"}]`))
		case r.URL.Path == "/data/obs/CN-11/recent":
			seenToken = r.Header.Get("X-eBirdApiToken")
			seenLocale = r.URL.Query().Get("sppLocale")
			w.Write([]byte(`[{"speciesCode":"eurspa1","comName":"雀鹰","sciName":"Accipiter nisus","locId":"L1","locName":"奥森","obsDt":"2026-11-25 08:00","howMany":1,"subId":"S1","hasRichMedia":false}]`))
		case r.URL.Path == "/product/checklist/view/S1":
			w.Write([]byte(`{"subId":"S1","userDisplayName":"观鸟者A","numSpecies":3,"obs":[{"speciesCode":"eurspa1","howManyStr":"1","comments":"高空盘旋","mediaCounts":{"P":2}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient()
	c.EBirdBaseURL = srv.URL

	taxa, err := c.FetchTaxonomy(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if seenLocale != "zh_SIM" {
		t.Fatalf("expected default taxonomy locale zh_SIM, got %q", seenLocale)
	}
	if len(taxa) != 1 || taxa[0].ComName != "普通翠鸟" {
		t.Fatalf("unexpected taxa: %+v", taxa)
	}

	obs, err := c.FetchRecentObservations(context.Background(), "demo-token", "CN-11", 14, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if seenToken != "demo-token" || seenLocale != "zh_SIM" || len(obs) != 1 {
		t.Fatalf("unexpected recent obs: token=%q locale=%q obs=%+v", seenToken, seenLocale, obs)
	}

	cl, err := c.FetchChecklistDetail(context.Background(), "demo-token", "S1")
	if err != nil {
		t.Fatal(err)
	}
	if cl.UserDisplayName != "观鸟者A" || len(cl.Obs) != 1 || cl.Obs[0].MediaCounts["P"] != 2 {
		t.Fatalf("unexpected checklist detail: %+v", cl)
	}
}

func TestGuideMediaVsRegionalFieldPhotosSeparation(t *testing.T) {
	var wikiLang string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/taxa"):
			w.Write([]byte(`{"results":[
				{"id":1,"name":"Alcedo atthis taprobana","observations_count":10,"default_photo":{"medium_url":"https://example.com/subspecies_medium.jpg","attribution":"Subspecies"}},
				{"id":2599,"name":"Alcedo atthis","observations_count":500,"default_photo":{"medium_url":"https://example.com/guide_medium.jpg","attribution":"GuidePhoto"}}
			]}`))
		case strings.HasPrefix(r.URL.Path, "/observations") && r.URL.Query().Get("lat") != "":
			w.Write([]byte(`{"results":[{"id":999,"observed_on":"2026-11-24","place_guess":"北京沙河水库","quality_grade":"research","taxon":{"name":"Alcedo atthis","preferred_common_name":"普通翠鸟"},"user":{"name":"现场拍摄者"},"photos":[{"url":"https://example.com/field_square.jpg","attribution":"FieldObs"}]}]}`))
		case strings.HasPrefix(r.URL.Path, "/observations"):
			w.Write([]byte(`{"results":[{"quality_grade":"research","place_guess":"测试地","user":{"login":"u1"},"sounds":[{"file_url":"https://example.com/call.mp3","file_content_type":"audio/mpeg"}]}]}`))
		case strings.HasPrefix(r.URL.Path, "/page/summary/"):
			wikiLang = r.Header.Get("Accept-Language")
			w.Write([]byte(`{"title":"普通翠鸟","extract":"一种小型翠鸟。","content_urls":{"desktop":{"page":"https://zh.wikipedia.org/wiki/%E6%99%AE%E9%80%9A%E7%BF%A0%E9%B8%9F"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient()
	c.INatBaseURL = srv.URL
	c.WikiBaseURL = srv.URL

	guide := c.FetchSpeciesGuideMedia(context.Background(), "Alcedo atthis", "普通翠鸟", "")
	if len(guide.Photos) == 0 || !strings.Contains(guide.Photos[0].URL, "guide_large.jpg") {
		t.Fatalf("expected exact species match guide photo, got %+v", guide.Photos)
	}
	if len(guide.Sounds) == 0 || guide.Wiki == nil {
		t.Fatalf("expected guide sound and wiki, got sounds=%v wiki=%v", guide.Sounds, guide.Wiki)
	}
	if wikiLang != "zh-cn" || !strings.Contains(guide.Wiki.PageURL, "zh.wikipedia.org/zh-cn/") {
		t.Fatalf("expected zh-cn wiki request and desktop link, lang=%q url=%q", wikiLang, guide.Wiki.PageURL)
	}

	fieldObs, err := c.FetchRegionalPhotoObservations(context.Background(), 39.9, 116.4, 25, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(fieldObs) != 1 || !strings.Contains(fieldObs[0].Photos[0].URL, "field_large.jpg") {
		t.Fatalf("expected field photo from observation itself, got %+v", fieldObs)
	}
}

func TestFetchSpeciesThumbExactMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"results":[
				{"id":1,"name":"Alcedo atthis taprobana","default_photo":{"medium_url":"https://example.com/subspecies.jpg"}},
				{"id":2,"name":"Alcedo atthis","default_photo":{"medium_url":"https://example.com/exact_medium.jpg"}}
			]
		}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.INatBaseURL = srv.URL
	got, err := c.FetchSpeciesThumb(context.Background(), "Alcedo atthis")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/exact_medium.jpg" {
		t.Fatalf("expected exact species match thumb, got %q", got)
	}
}

func TestCleanEBirdTextAndChecklistUnescape(t *testing.T) {
	raw := "黑身&#xff0c;头灰白&#xff0c;腹部腋下白&#xff0c;长嘴长尾&#xff0c;有照片"
	want := "黑身，头灰白，腹部腋下白，长嘴长尾，有照片"
	if got := CleanEBirdText(raw); got != want {
		t.Fatalf("CleanEBirdText mismatch: got %q want %q", got, want)
	}
	if got := CleanEBirdText("A&amp;#xff0c;B&lt;br/&gt;C"); got != "A，B\nC" {
		t.Fatalf("double-escaped br decode mismatch: got %q", got)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/product/checklist/view/S2":
			w.Write([]byte(`{"subId":"S2","userDisplayName":"观鸟人&amp;队友","numSpecies":1,"obs":[{"speciesCode":"bludro1","comName":"黑卷尾&#xff08;普通&#xff09;","comments":"黑身&#xff0c;头灰白&#xff0c;腹部腋下白&#xff0c;长嘴长尾&#xff0c;有照片"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient()
	c.EBirdBaseURL = srv.URL
	detail, err := c.FetchChecklistDetail(context.Background(), "test-key", "S2")
	if err != nil {
		t.Fatal(err)
	}
	if detail.UserDisplayName != "观鸟人&队友" || len(detail.Obs) != 1 || detail.Obs[0].Comments != want {
		t.Fatalf("unexpected unescaped checklist detail: %+v", detail)
	}
}

func TestFetchSpeciesGuideMediaProgressiveDecouplesSlowWiki(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/taxa":
			w.Write([]byte(`{"results":[{"id":21,"name":"Alcedo atthis","observations_count":100,"default_photo":{"medium_url":"https://static.inaturalist.org/photos/123/medium.jpg","attribution":"A","license_code":"cc-by"},"taxon_photos":[{"photo":{"medium_url":"https://static.inaturalist.org/photos/456/medium.jpg","attribution":"B"}}]}]}`))
		case r.URL.Path == "/observations":
			w.Write([]byte(`{"results":[{"user":{"name":"Rec1"},"place_guess":"Beijing","sounds":[{"file_url":"https://example.com/call.mp3","file_content_type":"audio/mpeg","license_code":"cc0"}]}]}`))
		case strings.HasPrefix(r.URL.Path, "/page/summary/"):
			select {
			case <-time.After(5 * time.Second):
				w.Write([]byte(`{"title":"普通翠鸟","extract":"late wiki"}`))
			case <-r.Context().Done():
				return
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient()
	c.INatBaseURL = srv.URL
	c.WikiBaseURL = srv.URL

	// 1. 列表缩略图应自动将 /medium. 转为轻量 /small.
	thumb, err := c.FetchSpeciesThumb(context.Background(), "Alcedo atthis")
	if err != nil {
		t.Fatal(err)
	}
	if thumb != "https://static.inaturalist.org/photos/123/small.jpg" {
		t.Fatalf("expected /small.jpg thumb conversion, got %q", thumb)
	}

	// 2. 渐进拉取应在百科超时前（<500ms）先吐出照片与鸣声，并在约 3.5s 独立放弃卡顿百科而非等足 5s
	start := time.Now()
	var mu sync.Mutex
	var partialPhotosAndSoundsAt time.Duration
	var partialPhotos int
	var partialSounds int

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	finalMedia := c.FetchSpeciesGuideMediaProgressive(ctx, "Alcedo atthis", "普通翠鸟", "", func(partial *model.SpeciesMedia) {
		mu.Lock()
		defer mu.Unlock()
		if len(partial.Photos) > partialPhotos {
			partialPhotos = len(partial.Photos)
		}
		if len(partial.Sounds) > partialSounds {
			partialSounds = len(partial.Sounds)
		}
		if len(partial.Photos) >= 2 && len(partial.Sounds) >= 1 && partialPhotosAndSoundsAt == 0 {
			partialPhotosAndSoundsAt = time.Since(start)
		}
	})
	totalElapsed := time.Since(start)

	if partialPhotosAndSoundsAt == 0 || partialPhotosAndSoundsAt > 800*time.Millisecond {
		t.Fatalf("expected fast partial photos+sounds in <800ms despite 5s slow wiki, got %v (photos=%d sounds=%d)", partialPhotosAndSoundsAt, partialPhotos, partialSounds)
	}
	if totalElapsed >= 4700*time.Millisecond {
		t.Fatalf("expected slow wiki canceled by 3.5s budget instead of blocking 5s, elapsed=%v", totalElapsed)
	}
	if finalMedia == nil || len(finalMedia.Photos) < 2 || len(finalMedia.Sounds) < 1 {
		t.Fatalf("expected iNaturalist photos & sounds preserved, got %+v", finalMedia)
	}
}

func TestChinaRegionsCleanNamesReverseGeocodeAndSeasonalData(t *testing.T) {
	// 1. 观鸟地区名称不再展示 (CN-XX) / (HK) 等代号后缀
	if len(ChinaRegions) < 30 {
		t.Fatalf("expected at least 30 China regions, got %d", len(ChinaRegions))
	}
	for _, r := range ChinaRegions {
		if strings.Contains(r.Name, "(") || strings.Contains(r.Name, "CN-") {
			t.Fatalf("region display name should not contain code suffix: %+v", r)
		}
		if r.Code == "" {
			t.Fatalf("region code should remain populated for API calls: %+v", r)
		}
	}

	// 2. 反向地理编码：在线 BigDataCloud 响应与离线中国城市坐标回退
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"principalSubdivision":"北京市","city":"北京市","locality":"朝阳区"}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.GeocodeBaseURL = srv.URL
	if got := c.ReverseGeocodeCity(context.Background(), 39.99, 116.40); got != "北京市 朝阳区" {
		t.Fatalf("expected online geocode 北京市 朝阳区, got %q", got)
	}

	// 离线兜底（指向无效端点）仍能准确返回最近的中国城市名称，不再只显示经纬度
	cOffline := NewClient()
	cOffline.GeocodeBaseURL = "http://127.0.0.1:1"
	if got := cOffline.ReverseGeocodeCity(context.Background(), 30.658, 104.065); got != "成都市" {
		t.Fatalf("expected offline nearest city 成都市, got %q", got)
	}
	if got := cOffline.ReverseGeocodeCity(context.Background(), 22.54, 114.05); got != "深圳市" {
		t.Fatalf("expected offline nearest city 深圳市, got %q", got)
	}

	// 3. 观鸟指南数据库完整性与多区多月分类查询
	if len(SeasonalBirdGuideDatabase) < 25 {
		t.Fatalf("expected rich seasonal guide database (>=25 species), got %d", len(SeasonalBirdGuideDatabase))
	}
	seen := make(map[string]bool)
	for _, b := range SeasonalBirdGuideDatabase {
		if b.SpeciesCode == "" || b.ComName == "" || b.SciName == "" || b.NationalSpots == "" || b.PlumageChanges == "" {
			t.Fatalf("incomplete seasonal bird record: %+v", b)
		}
		if seen[b.SpeciesCode] {
			t.Fatalf("duplicate seasonal species code: %s", b.SpeciesCode)
		}
		seen[b.SpeciesCode] = true
	}
	bjNov := SeasonalBirdsByRegionAndMonth("CN-11", 11)
	if len(bjNov[model.PhaseInSeason]) == 0 || len(bjNov[model.PhaseLeaving]) == 0 {
		t.Fatalf("expected Beijing November guide to have in-season and leaving birds, got %+v", bjNov)
	}
	if _, ok := FindSeasonalBirdByCode("blfspo1"); !ok {
		t.Fatal("expected blfspo1 in seasonal guide database")
	}
	blncra, ok := FindSeasonalBirdByCode("blncra1")
	if !ok {
		t.Fatal("expected blncra1 in seasonal guide database")
	}
	if blncra.PhaseFor(model.ZoneNorth, 11) != model.PhaseAbsent || blncra.PhaseFor(model.ZoneSouthwest, 11) != model.PhaseInSeason || blncra.PhaseFor(model.ZoneNational, 11) != model.PhaseInSeason {
		t.Fatalf("expected strict zone filtering for 黑颈鹤: north=%v sw=%v national=%v",
			blncra.PhaseFor(model.ZoneNorth, 11), blncra.PhaseFor(model.ZoneSouthwest, 11), blncra.PhaseFor(model.ZoneNational, 11))
	}
}
