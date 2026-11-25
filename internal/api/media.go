package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/Darastas/Birdview/internal/model"
)

// FetchSpeciesGuideMedia 为「鸟类图鉴」按需聚合指定鸟种的标准资料：
// 1. iNaturalist Taxa（标准生态照与物种信息）
// 2. iNaturalist Sounds（免 Key 野外实录鸟鸣 MP3）
// 3. Xeno-canto v3（可选：若用户配置了 Xeno-canto Key，附加专业鸟鸣录音）
// 4. 中文维基百科 REST Summary（中文百科简介）
// 注意：仅用于图鉴视图，绝不用于区域近期观测记录视图。
func (c *Client) FetchSpeciesGuideMedia(ctx context.Context, sciName, comName, xcKey string) *model.SpeciesMedia {
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

	// 1. iNaturalist 物种信息与标准配图
	wg.Add(1)
	go func() {
		defer wg.Done()
		u := fmt.Sprintf("%s/taxa?q=%s&rank=species&per_page=1&locale=zh-CN",
			c.INatBaseURL, url.QueryEscape(sciName))
		var resp inatTaxaResp
		if err := c.getJSON(ctx, u, nil, &resp); err != nil || len(resp.Results) == 0 {
			return
		}
		taxon := resp.Results[0]
		mu.Lock()
		defer mu.Unlock()
		res.INatTaxonID = taxon.ID
		res.INatURL = fmt.Sprintf("https://www.inaturalist.org/taxa/%d", taxon.ID)
		res.INatObservationsCount = taxon.ObservationsCount
		if taxon.DefaultPhoto.MediumURL != "" {
			res.Photos = append(res.Photos, model.MediaPhoto{
				URL:         strings.Replace(taxon.DefaultPhoto.MediumURL, "medium.", "large.", 1),
				ThumbURL:    taxon.DefaultPhoto.MediumURL,
				Attribution: fallbackStr(taxon.DefaultPhoto.Attribution, "iNaturalist"),
				License:     taxon.DefaultPhoto.LicenseCode,
				Source:      "iNaturalist",
			})
		}
		for _, tp := range taxon.TaxonPhotos {
			if tp.Photo.MediumURL == "" || len(res.Photos) >= 5 {
				continue
			}
			large := strings.Replace(tp.Photo.MediumURL, "medium.", "large.", 1)
			if !hasPhotoURL(res.Photos, large) {
				res.Photos = append(res.Photos, model.MediaPhoto{
					URL:         large,
					ThumbURL:    tp.Photo.MediumURL,
					Attribution: fallbackStr(tp.Photo.Attribution, "iNaturalist"),
					License:     tp.Photo.LicenseCode,
					Source:      "iNaturalist",
				})
			}
		}
	}()

	// 2. iNaturalist 鸟鸣实录（免 Key）+ 补充图集
	wg.Add(1)
	go func() {
		defer wg.Done()
		u := fmt.Sprintf("%s/observations?taxon_name=%s&sounds=true&quality_grade=research&per_page=6&order_by=votes",
			c.INatBaseURL, url.QueryEscape(sciName))
		var resp inatObsResp
		if err := c.getJSON(ctx, u, nil, &resp); err != nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for _, obs := range resp.Results {
			if len(res.Photos) < 5 {
				for _, p := range obs.Photos {
					if p.URL == "" {
						continue
					}
					large := strings.Replace(p.URL, "square.", "large.", 1)
					thumb := strings.Replace(p.URL, "square.", "medium.", 1)
					if !hasPhotoURL(res.Photos, large) {
						res.Photos = append(res.Photos, model.MediaPhoto{
							URL:         large,
							ThumbURL:    thumb,
							Attribution: fallbackStr(p.Attribution, "iNaturalist"),
							License:     p.LicenseCode,
							Source:      "iNaturalist",
						})
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
	}()

	// 3. 可选：Xeno-canto API v3 专业鸟鸣库
	if strings.TrimSpace(xcKey) != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			query := fmt.Sprintf(`sp:"%s"`, sciName)
			u := fmt.Sprintf("%s/recordings?query=%s&key=%s",
				c.XenoBaseURL, url.QueryEscape(query), url.QueryEscape(strings.TrimSpace(xcKey)))
			var resp xenoResp
			if err := c.getJSON(ctx, u, nil, &resp); err != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
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
		}()
	}

	// 4. 中文维基百科摘要
	wg.Add(1)
	go func() {
		defer wg.Done()
		titles := []string{}
		if comName != "" {
			titles = append(titles, comName)
		}
		if sciName != "" && sciName != comName {
			titles = append(titles, sciName)
		}
		for _, title := range titles {
			u := fmt.Sprintf("%s/page/summary/%s", c.WikiBaseURL, url.PathEscape(title))
			var w wikiResp
			if err := c.getJSON(ctx, u, nil, &w); err == nil && w.Extract != "" {
				wikiSummary = &model.WikiSummary{
					Title:       w.Title,
					Description: w.Description,
					Extract:     w.Extract,
					PageURL:     w.ContentURLs.Desktop.Page,
					Thumbnail:   w.Thumbnail.Source,
					Source:      "中文维基百科",
				}
				return
			}
		}
	}()

	wg.Wait()
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

type inatTaxaResp struct {
	Results []struct {
		ID                int `json:"id"`
		ObservationsCount int `json:"observations_count"`
		DefaultPhoto      struct {
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
	} `json:"results"`
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
