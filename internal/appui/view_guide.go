package appui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"
	"github.com/egoist/mygo/ui/netimage"
)

func (a *AppState) loadTaxonomyAsync(forceRefresh bool) {
	if a.taxaLoading {
		return
	}
	a.taxaLoading = true
	a.taxaErr = ""
	a.invalidateUI()

	go func() {
		const staleLimit = 30 * 24 * time.Hour
		const freshLimit = 24 * time.Hour

		var taxa []model.Taxon
		defaultRegion := a.store.Settings().DefaultRegion
		cachedChina := make(map[string]bool)
		cachedRegion := make(map[string]bool)
		needRevalidate := forceRefresh

		if !forceRefresh && a.cache != nil {
			if cached, ok := a.cache.LoadTaxonomy("zh_SIM", staleLimit); ok {
				taxa = cached
				if age, hasAge := a.cache.TaxonomyAge("zh_SIM"); !hasAge || age > freshLimit {
					needRevalidate = true
				}
			} else {
				needRevalidate = true
			}
			if codes, ok := a.cache.LoadSpeciesList("CN", staleLimit); ok {
				for _, code := range codes {
					cachedChina[code] = true
				}
				if age, hasAge := a.cache.SpeciesListAge("CN"); !hasAge || age > freshLimit {
					needRevalidate = true
				}
			} else {
				needRevalidate = true
			}
			if codes, ok := a.cache.LoadSpeciesList(defaultRegion, staleLimit); ok {
				for _, code := range codes {
					cachedRegion[code] = true
				}
				if age, hasAge := a.cache.SpeciesListAge(defaultRegion); !hasAge || age > freshLimit {
					needRevalidate = true
				}
			} else {
				needRevalidate = true
			}
		} else {
			needRevalidate = true
		}

		key := a.store.LoadEbirdKey()
		if len(taxa) > 0 {
			a.updateUI(func() {
				a.SetTaxonomy(taxa, cachedChina, cachedRegion, defaultRegion)
				a.taxaLoading = false
				a.guideRefresh.Refreshing = false
				a.refreshStorageStats()
			})
			if !needRevalidate {
				return
			}
		}

		var (
			wg        sync.WaitGroup
			mu        sync.Mutex
			netTaxa   []model.Taxon
			taxaErr   error
			chinaSet  = make(map[string]bool)
			regionSet = make(map[string]bool)
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			fetched, err := a.client.FetchTaxonomy(ctx, "zh_SIM")
			mu.Lock()
			netTaxa = fetched
			taxaErr = err
			mu.Unlock()
			if err == nil && a.cache != nil && len(fetched) > 0 {
				_ = a.cache.SaveTaxonomy("zh_SIM", fetched)
			}
		}()

		if key != "" {
			wg.Add(2)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
				defer cancel()
				if codes, e := a.client.FetchSpeciesList(ctx, key, "CN"); e == nil && len(codes) > 0 {
					if a.cache != nil {
						_ = a.cache.SaveSpeciesList("CN", codes)
					}
					mu.Lock()
					for _, code := range codes {
						chinaSet[code] = true
					}
					mu.Unlock()
				}
			}()
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
				defer cancel()
				if codes, e := a.client.FetchSpeciesList(ctx, key, defaultRegion); e == nil && len(codes) > 0 {
					if a.cache != nil {
						_ = a.cache.SaveSpeciesList(defaultRegion, codes)
					}
					mu.Lock()
					for _, code := range codes {
						regionSet[code] = true
					}
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		finalTaxa := taxa
		if len(netTaxa) > 0 {
			finalTaxa = netTaxa
		}
		if len(finalTaxa) == 0 && taxaErr != nil {
			a.updateUI(func() {
				a.taxaLoading = false
				a.guideRefresh.Refreshing = false
				a.taxaErr = fmt.Sprintf("加载鸟类名录失败: %v", taxaErr)
			})
			return
		}
		if len(chinaSet) == 0 {
			chinaSet = cachedChina
		}
		if len(regionSet) == 0 {
			regionSet = cachedRegion
		}

		a.updateUI(func() {
			if len(finalTaxa) > 0 {
				if len(chinaSet) > 0 && a.session.GuideScope == 2 && a.session.GuideQuery == "" {
					a.session.GuideScope = 0
				}
				a.SetTaxonomy(finalTaxa, chinaSet, regionSet, defaultRegion)
			}
			a.taxaLoading = false
			a.guideRefresh.Refreshing = false
			a.refreshStorageStats()
		})
	}()
}

// SetTaxonomy 写入鸟类名录并重建中文化目/科索引及筛选列表（同时供测试直接注入数据）。
func (a *AppState) SetTaxonomy(taxa []model.Taxon, chinaSet, regionSet map[string]bool, regionCode string) {
	a.allTaxa = taxa
	a.taxonByCode = make(map[string]model.Taxon, len(taxa))
	orderMap := make(map[string]bool)
	familyMap := make(map[string]bool)

	for _, tx := range taxa {
		a.taxonByCode[tx.SpeciesCode] = tx
		if ord := model.OrderZh(tx.Order); ord != "" {
			orderMap[ord] = true
		}
		if fam := model.FamilyZh(tx.FamilySciName, tx.FamilyComName); fam != "" {
			familyMap[fam] = true
		}
	}

	if len(chinaSet) > 0 {
		a.chinaCodes = chinaSet
	} else if len(a.chinaCodes) == 0 {
		if a.session.GuideScope == 0 || a.session.GuideScope == 1 {
			a.session.GuideScope = 2
		}
	}
	if len(regionSet) > 0 {
		a.regionCodes = regionSet
		a.regionCodesFor = regionCode
	}

	orders := []string{"全部目"}
	for k := range orderMap {
		orders = append(orders, k)
	}
	sort.Strings(orders[1:])
	a.guideOrders = orders

	families := []string{"全部科"}
	for k := range familyMap {
		families = append(families, k)
	}
	sort.Strings(families[1:])
	a.guideFamilies = families

	a.applyGuideFilter()
	a.rebuildJournalSpeciesOptions()
}

// OpenSpeciesInGuide 跳转至图鉴页并推入指定物种的详情屏（不破坏当前 Tab 历史栈）。
func (a *AppState) OpenSpeciesInGuide(speciesCode string) {
	tx, ok := a.taxonByCode[speciesCode]
	if !ok {
		a.session.Tab = 0
		a.router.Replace("/guide")
		a.invalidateUI()
		return
	}
	a.guideActiveCode = tx.SpeciesCode
	a.guideGallery = mobile.PageState{}
	a.ensureGuideMediaLoaded(tx)
	a.router.Push("/guide/" + tx.SpeciesCode)
	a.invalidateUI()
}

func (a *AppState) selectGuideProvinceScope(regionCode string) {
	a.session.GuideScope = 1
	if regionCode == "" {
		regionCode = fallbackText(a.regionCodesFor, a.store.Settings().DefaultRegion)
	}
	if regionCode == "" {
		regionCode = "CN-11"
	}
	if regionCode == a.regionCodesFor && len(a.regionCodes) > 0 {
		a.applyGuideFilter()
		return
	}
	if a.cache != nil {
		if codes, ok := a.cache.LoadSpeciesList(regionCode, 24*time.Hour); ok && len(codes) > 0 {
			set := make(map[string]bool, len(codes))
			for _, c := range codes {
				set[c] = true
			}
			a.regionCodes = set
			a.regionCodesFor = regionCode
			a.applyGuideFilter()
			return
		}
	}
	a.regionCodes = nil
	a.regionCodesFor = regionCode
	a.applyGuideFilter()

	key := a.store.LoadEbirdKey()
	if key == "" || a.win == nil {
		return
	}
	a.taxaLoading = true
	a.invalidateUI()
	go func(rc, k string) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		codes, err := a.client.FetchSpeciesList(ctx, k, rc)
		cancel()
		a.updateUI(func() {
			a.taxaLoading = false
			if err == nil && len(codes) > 0 {
				if a.cache != nil {
					_ = a.cache.SaveSpeciesList(rc, codes)
				}
				set := make(map[string]bool, len(codes))
				for _, code := range codes {
					set[code] = true
				}
				if a.regionCodesFor == rc {
					a.regionCodes = set
				}
			}
			a.applyGuideFilter()
		})
	}(regionCode, key)
}

func (a *AppState) applyGuideFilter() {
	q := strings.ToLower(strings.TrimSpace(a.session.GuideQuery))
	out := make([]model.Taxon, 0, 256)

	for _, tx := range a.allTaxa {
		switch a.session.GuideScope {
		case 0:
			if len(a.chinaCodes) > 0 && !a.chinaCodes[tx.SpeciesCode] {
				continue
			}
		case 1:
			if len(a.regionCodes) > 0 && !a.regionCodes[tx.SpeciesCode] {
				continue
			}
		}
		if a.session.GuideOnlyUnlocked && !a.store.IsUnlocked(tx.SpeciesCode) {
			continue
		}

		ordZh := model.OrderZh(tx.Order)
		famZh := model.FamilyZh(tx.FamilySciName, tx.FamilyComName)
		if a.guideOrder != "" && a.guideOrder != "全部目" && ordZh != a.guideOrder {
			continue
		}
		if a.guideFamily != "" && a.guideFamily != "全部科" && famZh != a.guideFamily {
			continue
		}

		if q != "" {
			match := strings.Contains(strings.ToLower(tx.ComName), q) ||
				strings.Contains(strings.ToLower(tx.SciName), q) ||
				strings.Contains(strings.ToLower(ordZh), q) ||
				strings.Contains(strings.ToLower(famZh), q) ||
				strings.Contains(strings.ToLower(tx.SpeciesCode), q)
			if !match {
				continue
			}
		}
		out = append(out, tx)
	}
	a.filteredTaxa = out
	if len(out) > 0 {
		a.guideList.ScrollIntoView(0)
	}
	a.guideScroll.Y = 0
}

func mergeMediaPreservingRicher(base, incoming *model.SpeciesMedia) *model.SpeciesMedia {
	if incoming == nil {
		return base
	}
	if base == nil {
		return incoming
	}
	out := *incoming
	if len(incoming.Photos) < len(base.Photos) {
		out.Photos = append([]model.MediaPhoto(nil), base.Photos...)
		for _, p := range incoming.Photos {
			dup := false
			for _, ex := range out.Photos {
				if ex.URL == p.URL {
					dup = true
					break
				}
			}
			if !dup {
				out.Photos = append(out.Photos, p)
			}
		}
	}
	if out.INatTaxonID == 0 {
		out.INatTaxonID = base.INatTaxonID
		out.INatURL = base.INatURL
		out.INatObservationsCount = base.INatObservationsCount
	}
	if len(out.Sounds) == 0 && len(base.Sounds) > 0 {
		out.Sounds = base.Sounds
	}
	if out.Wiki == nil && base.Wiki != nil {
		out.Wiki = base.Wiki
	}
	return &out
}

func (a *AppState) ensureGuideMediaLoaded(tx model.Taxon) {
	if a.guideMediaBusy[tx.SpeciesCode] {
		return
	}
	existing := a.guideMediaMap[tx.SpeciesCode]
	if existing != nil && len(existing.Photos) >= 4 {
		return
	}
	if a.guideMediaAttempt != nil {
		if lastTry, tried := a.guideMediaAttempt[tx.SpeciesCode]; tried {
			if existing != nil && len(existing.Photos) > 0 {
				return
			}
			if time.Since(lastTry) < 30*time.Second {
				return
			}
		}
	} else {
		a.guideMediaAttempt = make(map[string]time.Time)
	}
	a.guideMediaAttempt[tx.SpeciesCode] = time.Now()
	a.guideMediaBusy[tx.SpeciesCode] = true
	xcKey := a.store.LoadXenoKey()

	preloadMediaPhotos := func(m *model.SpeciesMedia) {
		if m == nil || len(m.Photos) == 0 {
			return
		}
		var urls []string
		for _, p := range m.Photos {
			if p.ThumbURL != "" {
				urls = append(urls, p.ThumbURL)
			}
			if p.URL != "" {
				urls = append(urls, p.URL)
			}
		}
		if len(urls) > 0 {
			netimage.Preload(urls...)
		}
	}

	go func() {
		var cachedBase *model.SpeciesMedia
		if a.cache != nil {
			if cached, ok := a.cache.LoadSpeciesMedia(tx.SpeciesCode, 7*24*time.Hour); ok {
				cachedBase = cached
				if len(cached.Photos) > 0 {
					preloadMediaPhotos(cached)
					a.updateUI(func() {
						a.guideMediaMap[tx.SpeciesCode] = cached
						a.thumbMap[tx.SpeciesCode] = fallbackText(cached.Photos[0].ThumbURL, cached.Photos[0].URL)
						a.refreshStorageStats()
					})
				}
				if len(cached.Photos) >= 4 {
					a.updateUI(func() {
						delete(a.guideMediaBusy, tx.SpeciesCode)
					})
					return
				}
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		media := a.client.FetchSpeciesGuideMediaProgressive(ctx, tx.SciName, tx.ComName, xcKey, func(partial *model.SpeciesMedia) {
			if partial == nil || (len(partial.Photos) == 0 && len(partial.Sounds) == 0 && partial.Wiki == nil) {
				return
			}
			merged := mergeMediaPreservingRicher(cachedBase, partial)
			preloadMediaPhotos(merged)
			a.updateUI(func() {
				a.guideMediaMap[tx.SpeciesCode] = merged
				if len(merged.Photos) > 0 && a.thumbMap[tx.SpeciesCode] == "" {
					a.thumbMap[tx.SpeciesCode] = fallbackText(merged.Photos[0].ThumbURL, merged.Photos[0].URL)
				}
			})
		})
		cancel()

		finalMedia := mergeMediaPreservingRicher(cachedBase, media)
		// 仅当包含至少 1 张照片时才写入 7 天磁盘缓存，避免无图瞬时失败污染后续打开
		if a.cache != nil && finalMedia != nil && len(finalMedia.Photos) > 0 {
			_ = a.cache.SaveSpeciesMedia(tx.SpeciesCode, finalMedia)
		}
		preloadMediaPhotos(finalMedia)

		a.updateUI(func() {
			delete(a.guideMediaBusy, tx.SpeciesCode)
			a.guideMediaMap[tx.SpeciesCode] = finalMedia
			if finalMedia != nil && len(finalMedia.Photos) > 0 {
				a.thumbMap[tx.SpeciesCode] = fallbackText(finalMedia.Photos[0].ThumbURL, finalMedia.Photos[0].URL)
				if a.cache != nil {
					snapshot := copyStringMap(a.thumbMap)
					go func() { _ = a.cache.SaveThumbIndex(snapshot) }()
				}
			}
			a.refreshStorageStats()
		})
	}()
}

func (a *AppState) speciesThumbURL(code string) string {
	if u := a.thumbMap[code]; u != "" {
		return strings.Replace(u, "/medium.", "/small.", 1)
	}
	if m, ok := a.guideMediaMap[code]; ok && len(m.Photos) > 0 {
		return strings.Replace(fallbackText(m.Photos[0].ThumbURL, m.Photos[0].URL), "/medium.", "/small.", 1)
	}
	return ""
}

func (a *AppState) viewGuideHome(c *ui.Context, t *ui.Theme) {
	hasFilter := (a.guideOrder != "" && a.guideOrder != "全部目") || (a.guideFamily != "" && a.guideFamily != "全部科")

	ui.Column(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(immersiveTopPad(c, 16), 18, 10, 18).Gap(12).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "图鉴").FontSize(26).FontWeight(600).LetterSpacing(-0.1).Role(ui.RoleHeading).Level(1)

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					lbl := "目科"
					fg := t.Text
					if hasFilter {
						fg = t.AccentText
						lbl = fallbackText(strings.TrimPrefix(a.guideFamily, "全部科"), a.guideOrder)
						if lbl == "" {
							lbl = a.guideOrder
						}
					}
					filterBtn := ui.Row(c.Key("guide-filter-open")).Height(34).PaddingX(14).Radius(17).
						Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("目科筛选")
					if hasFilter {
						applySculptedPrimaryStyle(filterBtn, t)
					} else {
						applySculptedSecondaryPillStyle(filterBtn, t)
					}
					filterBtn.Children(func() {
						ui.Icon(c, iconFilter).FontSize(13).TextColor(fg)
						ui.Text(c, lbl).FontSize(13).FontWeight(500).TextColor(fg).SingleLine()
					})
					if hapticClick(filterBtn) {
						a.guideFilterSheet = true
					}
				})
			})

			if minimalSearchBar(c, "guide-search", &a.session.GuideQuery, "搜索鸟种、学名或科名", "搜索鸟种", t) {
				a.applyGuideFilter()
			}

			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					scopeLabel := "中国"
					scopeIcon := iconMapPin
					switch a.session.GuideScope {
					case 1:
						scopeCode := fallbackText(a.regionCodesFor, a.store.Settings().DefaultRegion)
						scopeLabel = regionShortNameByCode(scopeCode)
					case 2:
						scopeLabel = "全球"
						scopeIcon = iconGlobe
					}
					scopeBtn := applySculptedSecondaryPillStyle(
						ui.Row(c.Key("guide-scope-btn")).Height(34).PaddingX(12).Radius(17).
							Gap(5).AlignItems(ui.Center).Shrink(0).
							Role(ui.RoleButton).Label("切换图鉴地区范围"),
						t,
					)
					scopeBtn.Children(func() {
						ui.Icon(c, scopeIcon).FontSize(13).TextColor(opticalAccentColor(t))
						ui.Text(c, scopeLabel).FontSize(13).FontWeight(500).TextColor(t.Text).SingleLine()
					})
					if hapticClick(scopeBtn) {
						a.regRegionQuery = ""
						a.guideScopeSheet = true
					}

					ulIdx := 0
					if a.session.GuideOnlyUnlocked {
						ulIdx = 1
					}
					if tactileCompactSegment(c, "guide-ul-seg", &ulIdx, t, "全部", "已解锁") {
						a.session.GuideOnlyUnlocked = ulIdx == 1
						a.applyGuideFilter()
					}
				})

				if a.taxaErr == "" {
					ui.Textf(c, "%d 种", len(a.filteredTaxa)).FontSize(12).FontWeight(500).TextColor(t.TextMuted).FontFeatures("tnum").Shrink(0)
				}
			})

			if a.taxaErr != "" {
				ui.Row(c).FillWidth().Padding(10, 14).Radius(14).Background(t.Danger.Alpha(0.1)).
					Border(1, t.Danger.Alpha(0.25)).
					Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(10).Children(func() {
					ui.Text(c, a.taxaErr).FontSize(12).TextColor(t.Danger).Grow(1)
					if hapticClick(filterPill(c, "taxa-retry", "重试", false, t)) {
						a.loadTaxonomyAsync(true)
					}
				})
			}
		})

		// 虚拟化全出血立体摄影卡牌双列网格（丝绒暗部渐变 + 顶部高光描边 + 双层纵深阴影）
		rowCount := (len(a.filteredTaxa) + 1) / 2
		a.guideList.Key = func(r int) any {
			idx := r * 2
			if idx < len(a.filteredTaxa) {
				return "gr-" + a.filteredTaxa[idx].SpeciesCode
			}
			return r
		}
		a.guideList.Label = nil

		ui.Box(c).Grow(1).Basis(0).FillWidth().Children(func() {
			list := ui.List(c.Key("guide-taxa-list"), &a.guideList, rowCount).Fill().Scrollbars(false).TrackScroll(&a.guideScroll)
			if list.PullToRefresh(&a.guideRefresh) {
				a.loadTaxonomyAsync(true)
			}
			list.Rows(func(row ui.ListRow) {
				rc := row.Context
				baseIdx := row.Index * 2
				padTop := float32(4)
				if row.Index == 0 {
					padTop = 10
				}
				padBottom := float32(12)
				if row.Index == rowCount-1 {
					padBottom = dockClearancePad(c, 22)
				}
				ui.Row(rc).FillWidth().Padding(padTop, 18, padBottom, 18).Gap(12).AlignItems(ui.Start).Children(func() {
					for col := 0; col < 2; col++ {
						idx := baseIdx + col
						if idx >= len(a.filteredTaxa) {
							ui.Box(rc).Grow(1).Basis(0)
							continue
						}
						tx := a.filteredTaxa[idx]
						unlocked := a.store.IsUnlocked(tx.SpeciesCode)
						thumbURL := a.speciesThumbURL(tx.SpeciesCode)
						var thumbSrc ui.ImageSource
						if thumbURL != "" {
							var thumbErr error
							thumbSrc, thumbErr = netimage.Load(rc, thumbURL, netimage.Options{MaxSide: speciesThumbMaxSide})
							if thumbSrc == nil && thumbErr != nil && strings.Contains(thumbURL, "/small.") {
								fbURL := strings.Replace(thumbURL, "/small.", "/medium.", 1)
								thumbSrc, _ = netimage.Load(rc, fbURL, netimage.Options{MaxSide: speciesThumbMaxSide})
							}
						}
						hasThumb := thumbSrc != nil

						tile := applyCardRimAndShadow(
							ui.Box(rc.Key("gtile-"+tx.SpeciesCode)).Grow(1).Basis(0).Height(206),
							18, t,
						).Clip().Role(ui.RoleButton).Label(tx.ComName)

						tile.Children(func() {
							if hasThumb {
								ui.Image(rc.Key("gt-"+tx.SpeciesCode), thumbSrc).Fill().Fit(ui.Cover).Role(ui.RoleImage)

								// 丝绒三段式底部渐隐遮罩：让白字铭牌浮现在清晰照片之上
								ui.Box(rc).Absolute().Left(0).Bottom(0).Right(0).Height(96).PassThrough().
									LinearGradient(ui.LinearGradient{
										Angle: 180,
										Stops: []ui.GradientStop{
											{Color: ui.Transparent, At: 0},
											{Color: ui.RGBA(8, 11, 17, 0.36), At: 0.45},
											{Color: ui.RGBA(8, 11, 17, 0.86), At: 1.0},
										},
									})
							} else {
								a.queueSpeciesThumb(tx)
								skeletonBirdCard(rc, 0, 34, t)
							}

							if unlocked {
								ui.Box(rc).Absolute().Top(10).Right(10).Size(22, 22).Radius(11).
									Background(t.Success).
									Border(1.5, ui.RGBA(255, 255, 255, 0.9)).
									InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.35)).
									Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.25)).
									Center().Children(func() {
									ui.Icon(rc, iconCheck).FontSize(11).TextColor(ui.RGB(255, 255, 255))
								})
							}

							titleCol := t.Text
							subCol := t.TextMuted
							if hasThumb {
								titleCol = ui.RGB(255, 255, 255)
								subCol = ui.RGBA(255, 255, 255, 0.80)
							}
							ui.Column(rc).Absolute().Left(0).Right(0).Bottom(0).Padding(0, 12, 12, 12).Gap(2).PassThrough().Children(func() {
								ui.Text(rc, tx.ComName).FontSize(15).FontWeight(600).TextColor(titleCol).SingleLine()
								ui.Text(rc, model.FormatTaxonLine(tx)).FontSize(12).FontWeight(400).TextColor(subCol).SingleLine()
							})
						})
						if hapticClick(tile) {
							a.OpenSpeciesInGuide(tx.SpeciesCode)
						}
					}
				})
			}).Children(func() {
				if len(a.filteredTaxa) == 0 && !a.taxaLoading {
					ui.Box(c).Fill().Padding(24, 20, 104, 20).Center().Children(func() {
						tactileCard(c, t).Padding(28).Center().Gap(10).Children(func() {
							tactileRecessedStyle(ui.Box(c).Size(52, 52), t, 26).Center().Children(func() {
								ui.Icon(c, iconBird).FontSize(24).TextColor(t.TextMuted)
							})
							ui.Text(c, "暂无匹配的鸟种").FontSize(16).FontWeight(600)
							ui.Text(c, "可切换浏览范围或清除筛选条件").FontSize(13).TextColor(t.TextMuted)
							if hasFilter || a.session.GuideQuery != "" || a.session.GuideOnlyUnlocked {
								if hapticClick(filterPill(c, "guide-reset-empty", "清除筛选条件", false, t)) {
									a.session.GuideQuery = ""
									a.session.GuideOnlyUnlocked = false
									a.guideOrder = "全部目"
									a.guideFamily = "全部科"
									a.applyGuideFilter()
								}
							}
						})
					})
				}
			})
			if len(a.filteredTaxa) > 0 {
				scrollTopFadeScrim(c, t, a.guideScroll.Y)
			}
		})
	})
}

func (a *AppState) viewGuideScopeSheet(c *ui.Context, t *ui.Theme) {
	sheetBg := t.Background
	mobile.Sheet(c, &a.guideScopeSheet, mobile.SheetOptions{
		Detents:    []float32{0.72, 0.92},
		Background: &sheetBg,
		Label:      "图鉴显示范围",
	}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 20, 14, 20).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "图鉴显示范围").FontSize(16).FontWeight(600)
				if hapticClick(filterPill(c, "gs-close", "关闭", false, t)) {
					a.guideScopeSheet = false
				}
			})
			minimalDivider(c, t)

			ui.Column(c).Padding(14, 18, 10, 18).Gap(10).Children(func() {
				minimalSearchBar(c, "gs-search", &a.regRegionQuery, "搜索省市名称或区号", "搜索地区", t)
			})

			ui.Scroll(c.Key("gs-list")).Grow(1).Basis(0).FillWidth().Scrollbars(false).Children(func() {
				ui.Column(c).FillWidth().Padding(4, 18, 36, 18).Gap(14).Children(func() {
					q := strings.ToLower(strings.TrimSpace(a.regRegionQuery))
					if q == "" {
						tactileCard(c, t).Clip().Dividers(1, subtleHairlineColor(t)).Children(func() {
							globalRow := ui.Row(c.Key("gs-global")).FillWidth().Padding(13, 16).
								Justify(ui.SpaceBetween).AlignItems(ui.Center).
								Role(ui.RoleButton).Label("全球")
							globalRow.Children(func() {
								w := 400
								if a.session.GuideScope == 2 {
									w = 500
								}
								ui.Text(c, "全球全部鸟种").FontSize(15).FontWeight(w)
								if a.session.GuideScope == 2 {
									ui.Icon(c, iconCheck).FontSize(16).TextColor(opticalAccentColor(t))
								}
							})
							if hapticClick(globalRow) {
								a.session.GuideScope = 2
								a.applyGuideFilter()
								a.guideScopeSheet = false
							}

							cnRow := ui.Row(c.Key("gs-cn")).FillWidth().Padding(13, 16).
								Justify(ui.SpaceBetween).AlignItems(ui.Center).
								Role(ui.RoleButton).Label("中国")
							cnRow.Children(func() {
								w := 400
								if a.session.GuideScope == 0 {
									w = 500
								}
								ui.Text(c, "中国全境鸟种").FontSize(15).FontWeight(w)
								if a.session.GuideScope == 0 {
									ui.Icon(c, iconCheck).FontSize(16).TextColor(opticalAccentColor(t))
								}
							})
							if hapticClick(cnRow) {
								a.session.GuideScope = 0
								a.applyGuideFilter()
								a.guideScopeSheet = false
							}
						})
					}

					tactileCard(c, t).Clip().Dividers(1, subtleHairlineColor(t)).Children(func() {
						activeRegion := fallbackText(a.regionCodesFor, a.store.Settings().DefaultRegion)
						for _, reg := range api.ChinaRegions {
							r := reg
							if q != "" && !strings.Contains(strings.ToLower(r.Name), q) && !strings.Contains(strings.ToLower(r.Code), q) {
								continue
							}
							active := a.session.GuideScope == 1 && strings.EqualFold(activeRegion, r.Code)
							row := ui.Row(c.Key("gs-reg-"+r.Code)).FillWidth().Padding(13, 16).
								Justify(ui.SpaceBetween).AlignItems(ui.Center).
								Role(ui.RoleButton).Label(r.Name)
							row.Children(func() {
								w := 400
								if active {
									w = 500
								}
								ui.Text(c, r.Name).FontSize(15).FontWeight(w)
								if active {
									ui.Icon(c, iconCheck).FontSize(16).TextColor(opticalAccentColor(t))
								}
							})
							if hapticClick(row) {
								a.selectGuideProvinceScope(r.Code)
								a.guideScopeSheet = false
							}
						}
					})
				})
			})
		})
	})
}

func (a *AppState) viewGuideFilterSheet(c *ui.Context, t *ui.Theme) {
	sheetBg := t.Background
	mobile.Sheet(c, &a.guideFilterSheet, mobile.SheetOptions{
		Detents:    []float32{0.68, 0.9},
		Background: &sheetBg,
		Label:      "目科筛选",
	}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 20, 14, 20).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				rst := ui.Box(c.Key("gf-reset")).Padding(6, 4).Role(ui.RoleButton).Label("重置").Children(func() {
					ui.Text(c, "重置").FontSize(14).TextColor(t.TextMuted)
				})
				if hapticClick(rst) {
					a.guideFilterQuery = ""
					a.guideOrder = "全部目"
					a.guideFamily = "全部科"
					a.applyGuideFilter()
				}
				ui.Text(c, "目科筛选").FontSize(16).FontWeight(600)
				if hapticClick(filterPill(c, "gf-done", "完成", true, t)) {
					a.guideFilterSheet = false
				}
			})
			minimalDivider(c, t)

			ui.Box(c).Padding(12, 18, 4, 18).Children(func() {
				minimalSearchBar(c, "gf-search", &a.guideFilterQuery, "搜索目名或科名（如：雀形目、鸭科）", "搜索目科", t)
			})

			fq := strings.ToLower(strings.TrimSpace(a.guideFilterQuery))
			var matchedOrders []string
			for _, ord := range a.guideOrders {
				if fq == "" {
					matchedOrders = append(matchedOrders, ord)
				} else if ord != "全部目" && strings.Contains(strings.ToLower(ord), fq) {
					matchedOrders = append(matchedOrders, ord)
				}
			}
			families := a.filteredFamiliesForOrder()
			var matchedFamilies []string
			for _, fam := range families {
				if fq == "" {
					matchedFamilies = append(matchedFamilies, fam)
				} else if fam != "全部科" && strings.Contains(strings.ToLower(fam), fq) {
					matchedFamilies = append(matchedFamilies, fam)
				}
			}

			ui.Scroll(c.Key("gf-scroll")).Grow(1).Basis(0).FillWidth().Scrollbars(false).Children(func() {
				ui.Column(c).FillWidth().Padding(10, 18, 36, 18).Gap(14).Children(func() {
					if len(matchedOrders) == 0 && len(matchedFamilies) == 0 {
						tactileCard(c, t).Padding(24).Center().Children(func() {
							ui.Text(c, "未找到匹配的目或科").FontSize(14).TextColor(t.TextMuted)
						})
						return
					}

					if len(matchedOrders) > 0 {
						tactileCard(c, t).Padding(16).Gap(10).Children(func() {
							ui.Text(c, "按目筛选").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
							ui.Row(c).FillWidth().Gap(8).Wrap().Children(func() {
								for i, ord := range matchedOrders {
									o := ord
									if hapticClick(filterPill(c, fmt.Sprintf("gfo-%d-%s", i, o), o, a.guideOrder == o, t)) {
										a.guideOrder = o
										a.guideFamily = "全部科"
										a.applyGuideFilter()
									}
								}
							})
						})
					}

					if len(matchedFamilies) > 0 {
						tactileCard(c, t).Padding(16).Gap(10).Children(func() {
							ui.Text(c, "按科筛选").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
							ui.Row(c).FillWidth().Gap(8).Wrap().Children(func() {
								for i, fam := range matchedFamilies {
									f := fam
									if hapticClick(filterPill(c, fmt.Sprintf("gff-%d-%s", i, f), f, a.guideFamily == f, t)) {
										a.guideFamily = f
										a.applyGuideFilter()
									}
								}
							})
						})
					}
				})
			})
		})
	})
}

func (a *AppState) filteredFamiliesForOrder() []string {
	if a.guideOrder == "" || a.guideOrder == "全部目" {
		return a.guideFamilies
	}
	seen := make(map[string]bool)
	out := []string{"全部科"}
	for _, tx := range a.allTaxa {
		if model.OrderZh(tx.Order) == a.guideOrder {
			if fam := model.FamilyZh(tx.FamilySciName, tx.FamilyComName); fam != "" && !seen[fam] {
				seen[fam] = true
				out = append(out, fam)
			}
		}
	}
	sort.Strings(out[1:])
	return out
}

func (a *AppState) viewGuideDetailPage(c *ui.Context, t *ui.Theme, code string) {
	tx, ok := a.taxonByCode[code]
	if !ok {
		ui.Column(c).Fill().Center().Padding(24).Gap(12).Children(func() {
			ui.Text(c, "未在名录中找到该鸟种")
			if hapticClick(filterPill(c, "gd-nf-back", "返回图鉴", false, t)) {
				a.goBack()
			}
		})
		return
	}

	media := a.guideMediaMap[tx.SpeciesCode]
	busy := a.guideMediaBusy[tx.SpeciesCode]
	if (media == nil || len(media.Photos) < 4) && !busy && a.win != nil {
		a.ensureGuideMediaLoaded(tx)
	}
	unlocked := a.store.IsUnlocked(tx.SpeciesCode)
	top, _, bottom, _ := c.SafeArea()
	topBtnY := max(float32(14), top+6)
	optA, _ := opticalAccentColors(t)

	ui.Scroll(c.Key("gd-scroll")).Fill().Scrollbars(false).Children(func() {
		ui.Column(c).FillWidth().Padding(0, 0, max(float32(44), bottom+28), 0).Gap(0).Children(func() {
			// 1. 全出血 Hero 摄影轮播区（顶部自研光学液态玻璃悬浮按钮 + 骨架扫光跳跃加载）
			count := 0
			if media != nil {
				count = len(media.Photos)
			}
			heroH := float32(352)
			if top > 0 {
				heroH = 340 + top
			}
			ui.Box(c).FillWidth().Height(heroH).Background(t.Surface).Children(func() {
				if count > 0 {
					mobile.PageView(c, &a.guideGallery, count, func(i int) {
						slide := ui.Box(c.Key(fmt.Sprintf("gslide-%d", i))).Fill().
							Role(ui.RoleButton).Label("点击放大图片")
						slide.Children(func() {
							photo := media.Photos[i]
							src, _ := netimage.Load(c, photo.URL, netimage.Options{MaxSide: 1200})
							if src == nil && photo.ThumbURL != "" {
								src, _ = netimage.Load(c, photo.ThumbURL, netimage.Options{MaxSide: 500})
							}
							if src != nil {
								ui.Image(c, src).Fill().Fit(ui.Cover).Role(ui.RoleImage)
							} else {
								skeletonBirdCard(c, 0, 46, t)
							}
						})
						if hapticClick(slide) {
							a.openNetworkPhotosViewer(tx.ComName, media.Photos, i)
						}
					}).Fill()
				} else if media == nil || busy {
					skeletonBirdCard(c, 0, 46, t)
				} else {
					ui.Box(c).Fill().Center().Children(func() {
						ui.Icon(c, iconBird).FontSize(48).TextColor(t.TextMuted.Alpha(0.28))
					})
				}

				if count > 0 {
					ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(84 + top).PassThrough().LinearGradient(ui.LinearGradient{
						From:  ui.RGBA(0, 0, 0, 0.34),
						To:    ui.RGBA(0, 0, 0, 0),
						Angle: 180,
					})
					ui.Box(c).Absolute().Left(0).Bottom(0).Right(0).Height(72).PassThrough().LinearGradient(ui.LinearGradient{
						From:  ui.RGBA(0, 0, 0, 0),
						To:    ui.RGBA(0, 0, 0, 0.48),
						Angle: 180,
					})
				}

				transitioning := a.router.Transitioning()
				if hapticClick(liquidGlassCircleBtnFast(c, "gd-back", "返回", iconBack, 38, transitioning).
					Absolute().Left(16).Top(topBtnY)) {
					a.goBack()
				}

				logBtn := applyPhotoOverlayGlass(
					ui.Row(c.Key("gd-top-log")).Height(36).PaddingX(14).Radius(18),
					transitioning, true,
				).
					Shadow(0, 4, 14, 0, ui.RGBA(0, 0, 0, 0.28)).
					Gap(5).AlignItems(ui.Center).
					Absolute().Right(16).Top(topBtnY).
					Role(ui.RoleButton).Label("记日志")
				logBtn.Children(func() {
					ui.Icon(c, iconPlus).FontSize(14).TextColor(ui.RGB(255, 255, 255))
					ui.Text(c, "记日志").FontSize(13).FontWeight(500).TextColor(ui.RGB(255, 255, 255))
				})
				if hapticClick(logBtn) {
					a.OpenNewJournalModal(tx.SpeciesCode, "")
				}

				if count > 0 {
					idx := max(0, min(a.guideGallery.Page, count-1))
					applyPhotoOverlayGlass(
						ui.Textf(c, "%d / %d", idx+1, count).
							Absolute().Left(18).Bottom(38).Padding(4, 10).Radius(999).
							FontSize(12).FontWeight(500).FontFeatures("tnum").
							TextColor(ui.RGB(255, 255, 255)),
						transitioning, false,
					)
					if count > 1 {
						ui.Box(c).Absolute().Right(16).Bottom(40).Children(func() {
							mobile.PageDots(c, idx, count)
						})
					}
				}
			})

			// 2. 重叠弧形立体详情承接面板（负外边距 + 顶部光影轮廓 + 立体质感卡片内容）
			sheetRim := ui.RGBA(255, 255, 255, 0.85)
			if t.Dark {
				sheetRim = ui.RGBA(255, 255, 255, 0.14)
			}
			sheet := applyCanvasBackground(
				ui.Column(c).FillWidth().Margin(-26, 0, 0, 0).Radius(26, 26, 0, 0).
					BorderWidth(1, 0, 0, 0).BorderColor(sheetRim).
					Shadow(0, -10, 28, 0, ui.RGBA(0, 0, 0, 0.16)).
					Padding(22, 18, 0, 18).Gap(16),
				t,
			)
			sheet.Children(func() {
				// 物种名片头（严格对齐下方实体卡片左缘）
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					ui.Text(c, tx.ComName).FontSize(24).FontWeight(600).LetterSpacing(-0.1)
					ui.Text(c, tx.SciName).FontSize(14).Italic().TextColor(t.TextMuted)

					var meta []metaItem
					if taxLine := model.FormatTaxonLine(tx); taxLine != "" {
						meta = append(meta, metaItem{Text: taxLine, Weight: 500})
					}
					if media != nil && media.INatObservationsCount > 0 {
						meta = append(meta, metaItem{Text: fmt.Sprintf("%d 条社区观测", media.INatObservationsCount), Color: optA, Weight: 500, Tnum: true})
					}
					metaAnnotationRow(c, t, meta...)

					if count > 0 {
						idx := max(0, min(a.guideGallery.Page, count-1))
						curPhoto := media.Photos[idx]
						ui.Textf(c, "图源：%s · %s", curPhoto.Source, curPhoto.Attribution).
							FontSize(12).TextColor(t.TextMuted).SingleLine()
					}

					if unlocked {
						var firstText string
						for _, ub := range a.store.UnlockedBirds() {
							if ub.SpeciesCode == tx.SpeciesCode {
								firstText = fmt.Sprintf("已解锁 · 首次于 %s（%s）",
									fallbackText(ub.FirstLocation, "野外现场"), model.FormatObsTime(ub.FirstSeenAt))
								break
							}
						}
						ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(4, 0, 0, 0).Children(func() {
							ui.Icon(c, iconCheck).FontSize(14).TextColor(t.Success)
							ui.Text(c, fallbackText(firstText, "已在个人图谱中解锁")).
								FontSize(13).FontWeight(500).TextColor(t.Success)
						})
					}
				})

				// 鸟鸣参考实体卡片
				tactileCard(c, t).Padding(16).Gap(12).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "鸟鸣参考").FontSize(16).FontWeight(600)
						if media != nil && len(media.Sounds) > 0 {
							ui.Textf(c, "%d 段", len(media.Sounds)).FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
						}
					})

					if media == nil && busy {
						skeletonAudioRows(c, 3, t)
					} else if media == nil || len(media.Sounds) == 0 {
						ui.Text(c, "暂无公开鸟鸣录音，可在「设置」补充 Xeno-canto Key 扩展曲库。").
							FontSize(13).TextColor(t.TextMuted).LineHeight(1.5)
					} else {
						pst := a.player.Status()
						ui.Column(c).FillWidth().Dividers(1, subtleHairlineColor(t)).Children(func() {
							for i, snd := range media.Sounds {
								snd := snd
								isThis := pst.SourceKey == snd.URL
								ui.Row(c.Key(fmt.Sprintf("gsnd-%d", i))).FillWidth().Padding(10, 0).Gap(12).AlignItems(ui.Center).Children(func() {
									btnIcon := iconPlay
									btnLbl := "播放"
									playBtn := ui.Box(c.Key(fmt.Sprintf("gsnd-b-%d", i))).Size(38, 38).Radius(19).Center().Shrink(0).
										Role(ui.RoleButton).Label(btnLbl)
									fg := t.Text
									if isThis && (pst.Playing || pst.Loading) {
										if pst.Playing {
											btnIcon = iconPause
											btnLbl = "暂停"
											playBtn.Label(btnLbl)
										}
										fg = ui.RGB(255, 255, 255)
										c1, c2 := opticalAccentColors(t)
										playBtn.LinearGradient(ui.LinearGradient{From: c1, To: c2, Angle: 180}).
											InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.35)).
											Shadow(0, 4, 12, 0, c1.Alpha(0.35))
									} else {
										applySculptedSecondaryPillStyle(playBtn, t)
									}
									playBtn.Children(func() {
										ui.Icon(c, btnIcon).FontSize(15).TextColor(fg)
									})
									if hapticClick(playBtn) {
										if isThis && (pst.Playing || pst.Loading) {
											a.player.TogglePause()
										} else {
											a.player.PlayURL(snd.URL, tx.ComName+" · "+snd.Type, snd.Source+" · "+snd.Place, a.cache, a.client, func() {
												a.updateUI(func() {})
											})
										}
									}

									ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
										ui.Textf(c, "%s · %s", snd.Type, snd.Source).FontSize(14).FontWeight(500).SingleLine()
										ui.Textf(c, "%s · %s", snd.Place, snd.Recordist).FontSize(12).TextColor(t.TextMuted).SingleLine()
									})
									if snd.Quality != "" && snd.Quality != "研究级鉴定" {
										statusPill(c, snd.Quality, softPillColor(t), secondaryTextColor(t))
									}
								})
							}
						})
					}
				})

				// 物种百科实体卡片
				tactileCard(c, t).Padding(16).Gap(10).Selectable().Children(func() {
					ui.Text(c, "物种简介").FontSize(16).FontWeight(600)
					if media == nil && busy {
						skeletonTextLines(c, t)
					} else if media != nil && media.Wiki != nil && media.Wiki.Extract != "" {
						ui.Text(c, media.Wiki.Extract).FontSize(15).LineHeight(1.75).TextColor(secondaryTextColor(t))
					} else {
						ui.Text(c, "中文维基百科暂未收录该鸟种的摘要说明。").FontSize(13).TextColor(t.TextMuted)
					}

					ui.Row(c).Gap(8).Wrap().Padding(4, 0, 0, 0).Children(func() {
						if media != nil && media.Wiki != nil && media.Wiki.PageURL != "" {
							externalLinkChip(c, "gd-link-wiki", "中文维基", iconWikipedia, media.Wiki.PageURL, t)
						}
						if media != nil && media.INatURL != "" {
							externalLinkChip(c, "gd-link-inat", "iNaturalist", iconINaturalist, media.INatURL, t)
						}
						externalLinkChip(c, "gd-link-ebird", "eBird 分布", iconEBird, fmt.Sprintf("https://ebird.org/species/%s", tx.SpeciesCode), t)
					})
				})

				if hapticClick(tactilePrimaryBtn(c, "gd-cta-log", "记入观鸟日志并解锁", t).FillWidth().Height(48).Radius(24)) {
					a.OpenNewJournalModal(tx.SpeciesCode, "")
				}
			})
		})
	})
}
