package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
