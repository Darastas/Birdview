package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Darastas/Birdview/internal/model"
)

const maxGuidePhotos = 12

func cloneSpeciesMedia(src *model.SpeciesMedia) *model.SpeciesMedia {
	if src == nil {
		return nil
	}
	cp := *src
	if len(src.Photos) > 0 {
		cp.Photos = append([]model.MediaPhoto(nil), src.Photos...)
	}
	if len(src.Sounds) > 0 {
		cp.Sounds = append([]model.MediaSound(nil), src.Sounds...)
	}
	if src.Wiki != nil {
		w := *src.Wiki
		cp.Wiki = &w
	}
	return &cp
}

// FetchSpeciesGuideMedia 为「鸟类图鉴」按需聚合指定鸟种的标准资料：
// 1. iNaturalist Taxa（标准生态照与物种信息）
// 2. iNaturalist Sounds（免 Key 野外实录鸟鸣 MP3）
// 3. Xeno-canto v3（可选：若用户配置了 Xeno-canto Key，附加专业鸟鸣录音）
// 4. 中文维基百科 REST Summary（中文百科简介）
// 注意：仅用于图鉴视图，绝不用于区域近期观测记录视图。
func (c *Client) FetchSpeciesGuideMedia(ctx context.Context, sciName, comName, xcKey string) *model.SpeciesMedia {
	return c.FetchSpeciesGuideMediaProgressive(ctx, sciName, comName, xcKey, nil)
}

// FetchSpeciesGuideMediaProgressive 并发拉取图鉴物种资料并在快源（iNaturalist 等）返回时增量回调，
// 避免中文维基百科慢响应或超时拖慢首图与鸣声展示。
func (c *Client) FetchSpeciesGuideMediaProgressive(ctx context.Context, sciName, comName, xcKey string, onPartial func(*model.SpeciesMedia)) *model.SpeciesMedia {
	res := &model.SpeciesMedia{
		SciName: sciName,
		ComName: comName,
	}
	if strings.TrimSpace(sciName) == "" {
		return res
	}

	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		wikiSummary *model.WikiSummary
	)
	emitPartial := func() {
		if onPartial == nil {
			return
		}
		mu.Lock()
		snap := cloneSpeciesMedia(res)
		if wikiSummary != nil {
			w := *wikiSummary
			snap.Wiki = &w
		}
		mu.Unlock()
		onPartial(snap)
	}

	// 1. iNaturalist 物种信息与标准配图（优先精确匹配拉丁学名，并深度拉取 /taxa/{id} 完整精选图集与研究级野外照）
	wg.Add(1)
	go func() {
		defer wg.Done()
		subCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
		trimmedSci := strings.TrimSpace(sciName)
		trimmedCom := strings.TrimSpace(comName)

		var resp inatTaxaResp
		searchURL := fmt.Sprintf("%s/taxa?q=%s&rank=species&per_page=5&locale=zh-CN",
			c.INatBaseURL, url.QueryEscape(trimmedSci))
		_ = c.getJSON(subCtx, searchURL, nil, &resp)
		if len(resp.Results) == 0 && trimmedCom != "" {
			fbURL := fmt.Sprintf("%s/taxa?q=%s&iconic_taxa=Aves&rank=species&per_page=5&locale=zh-CN",
				c.INatBaseURL, url.QueryEscape(trimmedCom))
			_ = c.getJSON(subCtx, fbURL, nil, &resp)
		}
		if len(resp.Results) == 0 {
			return
		}

		taxon := selectBestInatTaxon(resp.Results, trimmedSci, trimmedCom)
		mu.Lock()
		res.INatTaxonID = taxon.ID
		res.INatURL = fmt.Sprintf("https://www.inaturalist.org/taxa/%d", taxon.ID)
		res.INatObservationsCount = taxon.ObservationsCount
		appendTaxonMediaPhoto(&res.Photos, taxon.DefaultPhoto.MediumURL, taxon.DefaultPhoto.Attribution, taxon.DefaultPhoto.LicenseCode, maxGuidePhotos)
		for _, tp := range taxon.TaxonPhotos {
			appendTaxonMediaPhoto(&res.Photos, tp.Photo.MediumURL, tp.Photo.Attribution, tp.Photo.LicenseCode, maxGuidePhotos)
		}
		curPhotoCount := len(res.Photos)
		mu.Unlock()
		emitPartial()

		// 1b. iNaturalist /v1/taxa?q= 搜索接口不返回 taxon_photos，需请求 /v1/taxa/{id} 获取完整官方精选多图（通常 10-12 张）
		if taxon.ID > 0 && curPhotoCount < maxGuidePhotos {
			detailCtx, detailCancel := context.WithTimeout(subCtx, 4500*time.Millisecond)
			detailURL := fmt.Sprintf("%s/taxa/%d?locale=zh-CN", c.INatBaseURL, taxon.ID)
			var detailResp inatTaxaResp
			err := c.getJSON(detailCtx, detailURL, nil, &detailResp)
			detailCancel()
			if err == nil && len(detailResp.Results) > 0 {
				dt := detailResp.Results[0]
				mu.Lock()
				added := false
				if appendTaxonMediaPhoto(&res.Photos, dt.DefaultPhoto.MediumURL, dt.DefaultPhoto.Attribution, dt.DefaultPhoto.LicenseCode, maxGuidePhotos) {
					added = true
				}
				for _, tp := range dt.TaxonPhotos {
					if appendTaxonMediaPhoto(&res.Photos, tp.Photo.MediumURL, tp.Photo.Attribution, tp.Photo.LicenseCode, maxGuidePhotos) {
						added = true
					}
				}
				curPhotoCount = len(res.Photos)
				mu.Unlock()
				if added {
					emitPartial()
				}
			}
		}

		// 1c. 若精选图仍不足 10 张，补充该物种高赞研究级（research grade）野外生态照（不依赖 sounds=true）
		if taxon.ID > 0 && curPhotoCount < 10 {
			obsCtx, obsCancel := context.WithTimeout(subCtx, 4500*time.Millisecond)
			obsURL := fmt.Sprintf("%s/observations?taxon_id=%d&photos=true&quality_grade=research&per_page=10&order_by=votes",
				c.INatBaseURL, taxon.ID)
			var obsResp inatObsResp
			err := c.getJSON(obsCtx, obsURL, nil, &obsResp)
			obsCancel()
			if err == nil && len(obsResp.Results) > 0 {
				mu.Lock()
				added := false
				for _, obs := range obsResp.Results {
					if len(res.Photos) >= maxGuidePhotos {
						break
					}
					for _, p := range obs.Photos {
						if appendObsMediaPhoto(&res.Photos, p.URL, p.Attribution, p.LicenseCode, maxGuidePhotos) {
							added = true
							break
						}
					}
				}
				mu.Unlock()
				if added {
					emitPartial()
				}
			}
		}
	}()

	// 2. iNaturalist 鸟鸣实录（免 Key）+ 补充图集
	wg.Add(1)
	go func() {
		defer wg.Done()
		subCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		u := fmt.Sprintf("%s/observations?taxon_name=%s&sounds=true&quality_grade=research&per_page=6&order_by=votes",
			c.INatBaseURL, url.QueryEscape(sciName))
		var resp inatObsResp
		if err := c.getJSON(subCtx, u, nil, &resp); err != nil {
			return
		}
		mu.Lock()
		for _, obs := range resp.Results {
			if len(res.Photos) < maxGuidePhotos {
				for _, p := range obs.Photos {
					if appendObsMediaPhoto(&res.Photos, p.URL, p.Attribution, p.LicenseCode, maxGuidePhotos) {
						break
					}
				}
			}
			for _, s := range obs.Sounds {
				if s.Hidden || s.FileURL == "" || !isPlayableAudioURL(s.FileURL, s.FileContentType) {
					continue
				}
				if len(res.Sounds) >= 5 {
					break
				}
				rec := obs.User.Name
				if rec == "" {
					rec = obs.User.Login
				}
				res.Sounds = append(res.Sounds, model.MediaSound{
					URL:       s.FileURL,
					Type:      "野外实录鸟鸣",
					Quality:   "研究级鉴定",
					Recordist: fallbackStr(rec, "iNaturalist 观察者"),
					Place:     fallbackStr(obs.PlaceGuess, "野外记录"),
					License:   s.LicenseCode,
					Source:    "iNaturalist",
				})
			}
		}
		mu.Unlock()
		emitPartial()
	}()

	// 3. 可选：Xeno-canto API v3 专业鸟鸣库
	if strings.TrimSpace(xcKey) != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			subCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			defer cancel()
			query := fmt.Sprintf(`sp:"%s"`, sciName)
			u := fmt.Sprintf("%s/recordings?query=%s&key=%s",
				c.XenoBaseURL, url.QueryEscape(query), url.QueryEscape(strings.TrimSpace(xcKey)))
			var resp xenoResp
			if err := c.getJSON(subCtx, u, nil, &resp); err != nil {
				return
			}
			mu.Lock()
			var xcSounds []model.MediaSound
			for _, rec := range resp.Recordings {
				if rec.File == "" || len(xcSounds) >= 4 {
					continue
				}
				audioURL := rec.File
				if strings.HasPrefix(audioURL, "//") {
					audioURL = "https:" + audioURL
				}
				xcSounds = append(xcSounds, model.MediaSound{
					URL:       audioURL,
					Type:      fallbackStr(rec.Type, "鸟鸣"),
					Quality:   "评级 " + rec.Q,
					Recordist: fallbackStr(rec.Rec, "录音师"),
					Place:     strings.Trim(rec.Cnt+" · "+rec.Loc, " ·"),
					License:   rec.Lic,
					Source:    "Xeno-canto v3",
				})
			}
			res.Sounds = append(xcSounds, res.Sounds...)
			mu.Unlock()
			emitPartial()
		}()
	}

	// 4. 中文维基百科摘要（强制大陆简体中文，独立 3.5s 超时防卡死整体详情加载）
	wg.Add(1)
	go func() {
		defer wg.Done()
		subCtx, cancel := context.WithTimeout(ctx, 3500*time.Millisecond)
		defer cancel()
		titles := []string{}
		if comName != "" {
			titles = append(titles, comName)
		}
		if sciName != "" && sciName != comName {
			titles = append(titles, sciName)
		}
		wikiHeaders := map[string]string{"Accept-Language": "zh-cn"}
		for _, title := range titles {
			u := fmt.Sprintf("%s/page/summary/%s", c.WikiBaseURL, url.PathEscape(title))
			var w wikiResp
			if err := c.getJSON(subCtx, u, wikiHeaders, &w); err == nil && w.Extract != "" {
				pageURL := strings.Replace(w.ContentURLs.Desktop.Page, "zh.wikipedia.org/wiki/", "zh.wikipedia.org/zh-cn/", 1)
				mu.Lock()
				wikiSummary = &model.WikiSummary{
					Title:       w.Title,
					Description: w.Description,
					Extract:     w.Extract,
					PageURL:     pageURL,
					Thumbnail:   w.Thumbnail.Source,
					Source:      "中文维基百科",
				}
				mu.Unlock()
				return
			}
		}
	}()

	wg.Wait()
	mu.Lock()
	res.Wiki = wikiSummary
	if len(res.Photos) == 0 && res.Wiki != nil && res.Wiki.Thumbnail != "" {
		res.Photos = append(res.Photos, model.MediaPhoto{
			URL:         res.Wiki.Thumbnail,
			ThumbURL:    res.Wiki.Thumbnail,
			Attribution: "Wikimedia Commons",
			License:     "CC",
			Source:      "Wikipedia",
		})
	}
	mu.Unlock()
	return res
}

// FetchRegionalPhotoObservations 获取指定经纬度周边的真实带图现场观鸟记录（iNaturalist Geo Observations）。
// 这里的每一张照片均来自当次目击者上传的现场原图，严格不与图鉴通用图混用。
func (c *Client) FetchRegionalPhotoObservations(ctx context.Context, lat, lng float64, radiusKm, perPage int) ([]model.FieldPhotoObs, error) {
	if radiusKm <= 0 {
		radiusKm = 35
	}
	if perPage <= 0 {
		perPage = 30
	}
	u := fmt.Sprintf("%s/observations?iconic_taxa=Aves&lat=%s&lng=%s&radius=%d&photos=true&per_page=%d&order_by=observed_on&locale=zh-CN",
		c.INatBaseURL, formatFloat(lat), formatFloat(lng), radiusKm, perPage)
	var resp inatObsResp
	if err := c.getJSON(ctx, u, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]model.FieldPhotoObs, 0, len(resp.Results))
	for _, obs := range resp.Results {
		if len(obs.Photos) == 0 {
			continue
		}
		item := model.FieldPhotoObs{
			ID:           obs.ID,
			ComName:      fallbackStr(obs.Taxon.PreferredCommonName, obs.Taxon.Name),
			SciName:      obs.Taxon.Name,
			ObservedOn:   fallbackStr(obs.ObservedOnString, obs.ObservedOn),
			PlaceGuess:   fallbackStr(obs.PlaceGuess, "野外现场"),
			Observer:     fallbackStr(obs.User.Name, obs.User.Login),
			QualityGrade: obs.QualityGrade,
			ObsURL:       obs.URI,
			Lat:          obs.GeoJSON.CoordinatesLat(),
			Lng:          obs.GeoJSON.CoordinatesLng(),
		}
		for _, p := range obs.Photos {
			if p.URL == "" {
				continue
			}
			item.Photos = append(item.Photos, model.MediaPhoto{
				URL:         strings.Replace(p.URL, "square.", "large.", 1),
				ThumbURL:    strings.Replace(p.URL, "square.", "medium.", 1),
				Attribution: fallbackStr(p.Attribution, item.Observer),
				License:     p.LicenseCode,
				Source:      "观测者现场实拍 (iNaturalist)",
			})
		}
		for _, s := range obs.Sounds {
			if s.Hidden || s.FileURL == "" || !isPlayableAudioURL(s.FileURL, s.FileContentType) {
				continue
			}
			item.Sounds = append(item.Sounds, model.MediaSound{
				URL:       s.FileURL,
				Type:      "本次目击现场录音",
				Quality:   obs.QualityGrade,
				Recordist: item.Observer,
				Place:     item.PlaceGuess,
				License:   s.LicenseCode,
				Source:    "iNaturalist",
			})
		}
		out = append(out, item)
	}
	return out, nil
}

// FetchSpeciesThumb 为图鉴列表条目按需获取物种代表缩略图，严格核对拉丁学名避免张冠李戴。
func (c *Client) FetchSpeciesThumb(ctx context.Context, sciName string) (string, error) {
	return c.FetchSpeciesThumbWithFallback(ctx, sciName, "")
}

// FetchSpeciesThumbWithFallback 获取物种缩略图：优先精确匹配拉丁学名；若发生分类学异名拆并则按中文名/鸟纲候选兜底。
func (c *Client) FetchSpeciesThumbWithFallback(ctx context.Context, sciName, comName string) (string, error) {
	trimmedSci := strings.TrimSpace(sciName)
	trimmedCom := strings.TrimSpace(comName)
	if trimmedSci == "" && trimmedCom == "" {
		return "", nil
	}
	if trimmedSci != "" {
		u := fmt.Sprintf("%s/taxa?q=%s&rank=species&per_page=5&locale=zh-CN",
			c.INatBaseURL, url.QueryEscape(trimmedSci))
		var resp inatTaxaResp
		err := c.getJSON(ctx, u, nil, &resp)
		if err != nil {
			if trimmedCom == "" {
				return "", err
			}
		} else {
			for _, taxon := range resp.Results {
				if strings.EqualFold(strings.TrimSpace(taxon.Name), trimmedSci) && taxon.DefaultPhoto.MediumURL != "" {
					return strings.Replace(taxon.DefaultPhoto.MediumURL, "/medium.", "/small.", 1), nil
				}
			}
			// 异名或拼写差异兜底（同一中文名或唯一 Aves 候选）
			for _, taxon := range resp.Results {
				if taxon.DefaultPhoto.MediumURL == "" {
					continue
				}
				if trimmedCom != "" && strings.EqualFold(strings.TrimSpace(taxon.PreferredCommonName), trimmedCom) {
					return strings.Replace(taxon.DefaultPhoto.MediumURL, "/medium.", "/small.", 1), nil
				}
			}
			if trimmedCom != "" {
				for _, taxon := range resp.Results {
					if strings.EqualFold(taxon.IconicTaxonName, "Aves") && taxon.DefaultPhoto.MediumURL != "" {
						return strings.Replace(taxon.DefaultPhoto.MediumURL, "/medium.", "/small.", 1), nil
					}
				}
			}
		}
	}
	if trimmedCom != "" {
		u := fmt.Sprintf("%s/taxa?q=%s&iconic_taxa=Aves&rank=species&per_page=5&locale=zh-CN",
			c.INatBaseURL, url.QueryEscape(trimmedCom))
		var resp inatTaxaResp
		if err := c.getJSON(ctx, u, nil, &resp); err == nil {
			for _, taxon := range resp.Results {
				if taxon.DefaultPhoto.MediumURL == "" {
					continue
				}
				if strings.EqualFold(strings.TrimSpace(taxon.PreferredCommonName), trimmedCom) || strings.EqualFold(taxon.IconicTaxonName, "Aves") {
					return strings.Replace(taxon.DefaultPhoto.MediumURL, "/medium.", "/small.", 1), nil
				}
			}
		} else {
			return "", err
		}
	}
	return "", nil
}

func selectBestInatTaxon(results []inatTaxonItem, trimmedSci, trimmedCom string) inatTaxonItem {
	for _, candidate := range results {
		if trimmedSci != "" && strings.EqualFold(strings.TrimSpace(candidate.Name), trimmedSci) {
			return candidate
		}
	}
	for _, candidate := range results {
		if trimmedCom != "" && strings.EqualFold(strings.TrimSpace(candidate.PreferredCommonName), trimmedCom) {
			return candidate
		}
	}
	for _, candidate := range results {
		if strings.EqualFold(strings.TrimSpace(candidate.IconicTaxonName), "Aves") {
			return candidate
		}
	}
	return results[0]
}

func appendTaxonMediaPhoto(photos *[]model.MediaPhoto, mediumURL, attribution, license string, maxCount int) bool {
	if mediumURL == "" || len(*photos) >= maxCount {
		return false
	}
	large := strings.Replace(mediumURL, "medium.", "large.", 1)
	if hasPhotoURL(*photos, large) {
		return false
	}
	*photos = append(*photos, model.MediaPhoto{
		URL:         large,
		ThumbURL:    mediumURL,
		Attribution: fallbackStr(attribution, "iNaturalist"),
		License:     license,
		Source:      "iNaturalist",
	})
	return true
}

func appendObsMediaPhoto(photos *[]model.MediaPhoto, rawURL, attribution, license string, maxCount int) bool {
	if rawURL == "" || len(*photos) >= maxCount {
		return false
	}
	large := strings.Replace(rawURL, "square.", "large.", 1)
	thumb := strings.Replace(rawURL, "square.", "medium.", 1)
	if hasPhotoURL(*photos, large) {
		return false
	}
	*photos = append(*photos, model.MediaPhoto{
		URL:         large,
		ThumbURL:    thumb,
		Attribution: fallbackStr(attribution, "iNaturalist"),
		License:     license,
		Source:      "iNaturalist",
	})
	return true
}

func hasPhotoURL(list []model.MediaPhoto, u string) bool {
	for _, p := range list {
		if p.URL == u {
			return true
		}
	}
	return false
}

func isPlayableAudioURL(rawURL, contentType string) bool {
	lower := strings.ToLower(rawURL)
	if strings.Contains(lower, ".mp3") || strings.Contains(lower, ".wav") || strings.Contains(lower, ".ogg") || strings.Contains(lower, ".m4a") {
		return true
	}
	return strings.HasPrefix(strings.ToLower(contentType), "audio/")
}

func fallbackStr(v, fb string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fb
}

type inatTaxonItem struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	PreferredCommonName string `json:"preferred_common_name"`
	IconicTaxonName     string `json:"iconic_taxon_name"`
	ObservationsCount   int    `json:"observations_count"`
	DefaultPhoto        struct {
		MediumURL   string `json:"medium_url"`
		Attribution string `json:"attribution"`
		LicenseCode string `json:"license_code"`
	} `json:"default_photo"`
	TaxonPhotos []struct {
		Photo struct {
			MediumURL   string `json:"medium_url"`
			Attribution string `json:"attribution"`
			LicenseCode string `json:"license_code"`
		} `json:"photo"`
	} `json:"taxon_photos"`
}

type inatTaxaResp struct {
	Results []inatTaxonItem `json:"results"`
}

type inatGeoJSON struct {
	Coordinates []float64 `json:"coordinates"`
}

func (g inatGeoJSON) CoordinatesLng() float64 {
	if len(g.Coordinates) >= 1 {
		return g.Coordinates[0]
	}
	return 0
}

func (g inatGeoJSON) CoordinatesLat() float64 {
	if len(g.Coordinates) >= 2 {
		return g.Coordinates[1]
	}
	return 0
}

type inatObsResp struct {
	Results []struct {
		ID               int64       `json:"id"`
		URI              string      `json:"uri"`
		ObservedOn       string      `json:"observed_on"`
		ObservedOnString string      `json:"observed_on_string"`
		PlaceGuess       string      `json:"place_guess"`
		QualityGrade     string      `json:"quality_grade"`
		GeoJSON          inatGeoJSON `json:"geojson"`
		Taxon            struct {
			Name                string `json:"name"`
			PreferredCommonName string `json:"preferred_common_name"`
		} `json:"taxon"`
		User struct {
			Login string `json:"login"`
			Name  string `json:"name"`
		} `json:"user"`
		Photos []struct {
			URL         string `json:"url"`
			Attribution string `json:"attribution"`
			LicenseCode string `json:"license_code"`
		} `json:"photos"`
		Sounds []struct {
			FileURL         string `json:"file_url"`
			FileContentType string `json:"file_content_type"`
			LicenseCode     string `json:"license_code"`
			Hidden          bool   `json:"hidden"`
		} `json:"sounds"`
	} `json:"results"`
}

type xenoResp struct {
	Recordings []struct {
		File string `json:"file"`
		Type string `json:"type"`
		Q    string `json:"q"`
		Rec  string `json:"rec"`
		Cnt  string `json:"cnt"`
		Loc  string `json:"loc"`
		Lic  string `json:"lic"`
	} `json:"recordings"`
}

type wikiResp struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Extract     string `json:"extract"`
	Thumbnail   struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}
