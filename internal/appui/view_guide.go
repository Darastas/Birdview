package appui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

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
		var taxa []model.Taxon
		var err error

		if !forceRefresh && a.cache != nil {
			if cached, ok := a.cache.LoadTaxonomy("zh_SIM", 24*time.Hour); ok {
				taxa = cached
			}
		}
		if len(taxa) == 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			taxa, err = a.client.FetchTaxonomy(ctx, "zh_SIM")
			cancel()
			if err == nil && a.cache != nil && len(taxa) > 0 {
				_ = a.cache.SaveTaxonomy("zh_SIM", taxa)
			}
		}

		if err != nil {
			a.updateUI(func() {
				a.taxaLoading = false
				a.guideRefresh.Refreshing = false
				a.taxaErr = fmt.Sprintf("加载鸟类名录失败: %v", err)
			})
			return
		}

		defaultRegion := a.store.Settings().DefaultRegion
		cachedChina := make(map[string]bool)
		cachedRegion := make(map[string]bool)
		if !forceRefresh && a.cache != nil {
			if codes, ok := a.cache.LoadSpeciesList("CN", 24*time.Hour); ok {
				for _, c := range codes {
					cachedChina[c] = true
				}
			}
			if codes, ok := a.cache.LoadSpeciesList(defaultRegion, 24*time.Hour); ok {
				for _, c := range codes {
					cachedRegion[c] = true
				}
			}
		}

		// 先立即呈现名录，避免等待区域物种筛选接口时出现空列表
		a.updateUI(func() {
			a.SetTaxonomy(taxa, cachedChina, cachedRegion, defaultRegion)
			key := a.store.LoadEbirdKey()
			if key == "" || (len(cachedChina) > 0 && len(cachedRegion) > 0) {
				a.taxaLoading = false
				a.guideRefresh.Refreshing = false
				a.refreshStorageStats()
			}
		})

		key := a.store.LoadEbirdKey()
		if key == "" || (!forceRefresh && len(cachedChina) > 0 && len(cachedRegion) > 0) {
			return
		}

		var wg sync.WaitGroup
		chinaSet := make(map[string]bool)
		regionSet := make(map[string]bool)
		wg.Add(2)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			if codes, e := a.client.FetchSpeciesList(ctx, key, "CN"); e == nil && len(codes) > 0 {
				if a.cache != nil {
					_ = a.cache.SaveSpeciesList("CN", codes)
				}
				for _, code := range codes {
					chinaSet[code] = true
				}
			}
			cancel()
		}()
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			if codes, e := a.client.FetchSpeciesList(ctx, key, defaultRegion); e == nil && len(codes) > 0 {
				if a.cache != nil {
					_ = a.cache.SaveSpeciesList(defaultRegion, codes)
				}
				for _, code := range codes {
					regionSet[code] = true
				}
			}
			cancel()
		}()
		wg.Wait()

		a.updateUI(func() {
			if len(chinaSet) > 0 || len(regionSet) > 0 {
				if len(chinaSet) > 0 && a.session.GuideScope == 2 && a.session.GuideQuery == "" {
					a.session.GuideScope = 0
				}
				a.SetTaxonomy(taxa, chinaSet, regionSet, defaultRegion)
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

// OpenSpeciesInGuide 跳转至图鉴页并推入指定物种的详情屏。
func (a *AppState) OpenSpeciesInGuide(speciesCode string) {
	tx, ok := a.taxonByCode[speciesCode]
	a.session.Tab = 0
	if !ok {
		a.router.Replace("/guide")
		a.invalidateUI()
		return
	}
	a.guideActiveCode = tx.SpeciesCode
	a.guideGallery.Page = 0
	a.ensureGuideMediaLoaded(tx)
	a.router.Push("/guide/" + tx.SpeciesCode)
	a.invalidateUI()
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
		case 3:
			if !a.store.IsUnlocked(tx.SpeciesCode) {
				continue
			}
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
}

func (a *AppState) ensureGuideMediaLoaded(tx model.Taxon) {
	if _, ok := a.guideMediaMap[tx.SpeciesCode]; ok {
		return
	}
	if a.guideMediaBusy[tx.SpeciesCode] {
		return
	}
	a.guideMediaBusy[tx.SpeciesCode] = true
	xcKey := a.store.LoadXenoKey()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 22*time.Second)
		media := a.client.FetchSpeciesGuideMedia(ctx, tx.SciName, tx.ComName, xcKey)
		cancel()

		a.updateUI(func() {
			delete(a.guideMediaBusy, tx.SpeciesCode)
			a.guideMediaMap[tx.SpeciesCode] = media
			if len(media.Photos) > 0 {
				a.thumbMap[tx.SpeciesCode] = fallbackText(media.Photos[0].ThumbURL, media.Photos[0].URL)
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
		return u
	}
	if m, ok := a.guideMediaMap[code]; ok && len(m.Photos) > 0 {
		return fallbackText(m.Photos[0].ThumbURL, m.Photos[0].URL)
	}
	return ""
}

func (a *AppState) viewGuideHome(c *ui.Context, t *ui.Theme) {
	hasFilter := (a.guideOrder != "" && a.guideOrder != "全部目") || (a.guideFamily != "" && a.guideFamily != "全部科")

	ui.Column(c).Fill().Children(func() {
		// 极简留白顶部导航与搜索筛选
		ui.Column(c).FillWidth().Padding(16, 20, 10, 20).Gap(12).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "图鉴").FontSize(24).FontWeight(700).LetterSpacing(-0.4).Role(ui.RoleHeading).Level(1)

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if a.taxaLoading {
						ui.Spinner(c).Label("更新名录中")
					}
					btnBg := t.Surface
					btnFg := t.Text
					lbl := "目科"
					if hasFilter {
						btnBg = t.Accent
						btnFg = t.AccentText
						lbl = fallbackText(strings.TrimPrefix(a.guideFamily, "全部科"), a.guideOrder)
						if lbl == "" {
							lbl = a.guideOrder
						}
					}
					filterBtn := ui.Row(c.Key("guide-filter-open")).Height(32).PaddingX(12).Radius(16).
						Background(btnBg).Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("目科筛选")
					if !hasFilter && t.Dark {
						filterBtn.Border(1, t.Border)
					}
					filterBtn.Children(func() {
						ui.Icon(c, iconFilter).FontSize(13).TextColor(btnFg)
						ui.Text(c, lbl).FontSize(12.5).FontWeight(500).TextColor(btnFg).SingleLine()
					})
					if filterBtn.Clicked() {
						a.guideFilterSheet = true
					}
				})
			})

			if minimalSearchBar(c, "guide-search", &a.session.GuideQuery, "搜索鸟种、学名或科名", "搜索鸟种", t) {
				a.applyGuideFilter()
			}

			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(10).Children(func() {
				ui.ScrollHorizontal(c.Key("guide-scopes")).Grow(1).Basis(0).Gap(6).Children(func() {
					if filterPill(c, "scope-cn", "中国", a.session.GuideScope == 0, t).Clicked() {
						a.session.GuideScope = 0
						a.applyGuideFilter()
					}
					provLabel := "省区"
					if a.regionCodesFor != "" {
						provLabel = regionShortNameByCode(a.regionCodesFor)
					}
					if filterPill(c, "scope-prov", provLabel, a.session.GuideScope == 1, t).Clicked() {
						a.session.GuideScope = 1
						a.applyGuideFilter()
					}
					if filterPill(c, "scope-ul", "已解锁", a.session.GuideScope == 3, t).Clicked() {
						a.session.GuideScope = 3
						a.applyGuideFilter()
					}
					if filterPill(c, "scope-all", "全球", a.session.GuideScope == 2, t).Clicked() {
						a.session.GuideScope = 2
						a.applyGuideFilter()
					}
				})

				if a.taxaErr == "" {
					ui.Textf(c, "%d 种", len(a.filteredTaxa)).FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum").Shrink(0)
				}
			})

			if a.taxaErr != "" {
				ui.Row(c).FillWidth().Padding(10, 14).Radius(12).Background(t.Danger.Alpha(0.08)).
					Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(10).Children(func() {
					ui.Text(c, a.taxaErr).FontSize(12.5).TextColor(t.Danger).Grow(1)
					if filterPill(c, "taxa-retry", "重试", false, t).Clicked() {
						a.loadTaxonomyAsync(true)
					}
				})
			}
		})

		// 虚拟化双列摄影画册网格（无边框、无阴影、照片为主角）
		rowCount := (len(a.filteredTaxa) + 1) / 2
		a.guideList.Key = func(r int) any {
			idx := r * 2
			if idx < len(a.filteredTaxa) {
				return "gr-" + a.filteredTaxa[idx].SpeciesCode
			}
			return r
		}
		a.guideList.Label = nil

		list := ui.List(c.Key("guide-taxa-list"), &a.guideList, rowCount).
			Grow(1).Basis(0).FillWidth()
		if list.PullToRefresh(&a.guideRefresh) {
			a.loadTaxonomyAsync(true)
		}
		list.Rows(func(row ui.ListRow) {
			rc := row.Context
			baseIdx := row.Index * 2
			ui.Row(rc).FillWidth().Padding(4, 20, 14, 20).Gap(12).AlignItems(ui.Start).Children(func() {
				for col := 0; col < 2; col++ {
					idx := baseIdx + col
					if idx >= len(a.filteredTaxa) {
						ui.Box(rc).Grow(1).Basis(0)
						continue
					}
					tx := a.filteredTaxa[idx]
					unlocked := a.store.IsUnlocked(tx.SpeciesCode)

					tile := ui.Column(rc.Key("gtile-"+tx.SpeciesCode)).Grow(1).Basis(0).
						Role(ui.RoleButton).Label(tx.ComName)
					tile.Children(func() {
						ui.Box(rc).FillWidth().Height(148).Radius(14).Clip().Background(t.Surface).Children(func() {
							thumbURL := a.speciesThumbURL(tx.SpeciesCode)
							if thumbURL != "" {
								netimage.Image(rc.Key("gt-"+tx.SpeciesCode), thumbURL, netimage.Options{
									MaxSide:     360,
									Fit:         ui.Cover,
									Placeholder: t.Surface,
								}).Fill()
							} else {
								a.queueSpeciesThumb(tx)
								ui.Box(rc).Fill().Center().Children(func() {
									ui.Icon(rc, iconBird).FontSize(26).TextColor(t.TextMuted.Alpha(0.28))
								})
							}
							if unlocked {
								ui.Box(rc).Absolute().Top(8).Right(8).Size(20, 20).Radius(10).
									Background(t.Success).Center().Children(func() {
									ui.Icon(rc, iconCheck).FontSize(11).TextColor(ui.RGB(255, 255, 255))
								})
							}
						})

						ui.Column(rc).FillWidth().Padding(7, 2, 0, 2).Gap(1).Children(func() {
							ui.Text(rc, tx.ComName).FontSize(14.5).FontWeight(600).TextColor(t.Text).SingleLine()
							ui.Text(rc, model.FormatTaxonLine(tx)).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
						})
					})
					if tile.Clicked() {
						a.OpenSpeciesInGuide(tx.SpeciesCode)
					}
				}
			})
		}).Children(func() {
			if len(a.filteredTaxa) == 0 && !a.taxaLoading {
				ui.Column(c).Fill().Center().Padding(32).Gap(10).Children(func() {
					ui.Icon(c, iconBird).FontSize(34).TextColor(t.TextMuted.Alpha(0.4))
					ui.Text(c, "暂无匹配的鸟种").FontSize(15.5).FontWeight(600)
					ui.Text(c, "可切换浏览范围或清除关键词筛选").FontSize(13).TextColor(t.TextMuted)
					if hasFilter || a.session.GuideQuery != "" {
						if filterPill(c, "guide-reset-empty", "清除筛选条件", false, t).Clicked() {
							a.session.GuideQuery = ""
							a.guideOrder = "全部目"
							a.guideFamily = "全部科"
							a.applyGuideFilter()
						}
					}
				})
			}
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
					ui.Text(c, "重置").FontSize(14.5).TextColor(t.TextMuted)
				})
				if rst.Clicked() {
					a.guideOrder = "全部目"
					a.guideFamily = "全部科"
					a.applyGuideFilter()
				}
				ui.Text(c, "目科筛选").FontSize(16).FontWeight(600)
				if filterPill(c, "gf-done", "完成", true, t).Clicked() {
					a.guideFilterSheet = false
				}
			})
			minimalDivider(c, t)

			ui.Scroll(c.Key("gf-scroll")).Grow(1).Basis(0).FillWidth().Padding(16, 20, 28, 20).Gap(18).Children(func() {
				ui.Column(c).FillWidth().Gap(10).Children(func() {
					ui.Text(c, "按目筛选").FontSize(12.5).FontWeight(600).TextColor(t.TextMuted)
					ui.Row(c).FillWidth().Gap(8).Wrap().Children(func() {
						for i, ord := range a.guideOrders {
							o := ord
							if filterPill(c, fmt.Sprintf("gfo-%d", i), o, a.guideOrder == o, t).Clicked() {
								a.guideOrder = o
								a.guideFamily = "全部科"
								a.applyGuideFilter()
							}
						}
					})
				})

				minimalDivider(c, t)

				ui.Column(c).FillWidth().Gap(10).Children(func() {
					ui.Text(c, "按科筛选").FontSize(12.5).FontWeight(600).TextColor(t.TextMuted)
					families := a.filteredFamiliesForOrder()
					ui.Row(c).FillWidth().Gap(8).Wrap().Children(func() {
						for i, fam := range families {
							f := fam
							if filterPill(c, fmt.Sprintf("gff-%d", i), f, a.guideFamily == f, t).Clicked() {
								a.guideFamily = f
								a.applyGuideFilter()
							}
						}
					})
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
			if filterPill(c, "gd-nf-back", "返回图鉴", false, t).Clicked() {
				a.goBack()
			}
		})
		return
	}

	media := a.guideMediaMap[tx.SpeciesCode]
	busy := a.guideMediaBusy[tx.SpeciesCode]
	if media == nil && !busy && a.win != nil {
		a.ensureGuideMediaLoaded(tx)
	}
	unlocked := a.store.IsUnlocked(tx.SpeciesCode)
	top, _, _, _ := c.SafeArea()
	topBtnY := max(float32(12), top+6)

	ui.Scroll(c.Key("gd-scroll")).Fill().Background(t.Background).Children(func() {
		ui.Column(c).FillWidth().Padding(0, 0, 36, 0).Gap(0).Children(func() {
			// 1. 全出血 Hero 摄影轮播区（顶部浮动玻璃控制按钮）
			count := 0
			if media != nil {
				count = len(media.Photos)
			}
			heroH := float32(320)
			if top > 0 {
				heroH = 310 + top
			}
			ui.Box(c).FillWidth().Height(heroH).Background(t.Surface).Children(func() {
				if count > 0 {
					mobile.PageView(c, &a.guideGallery, count, func(i int) {
						slide := ui.Box(c.Key(fmt.Sprintf("gslide-%d", i))).Fill().
							Role(ui.RoleButton).Label("点击放大图片")
						slide.Children(func() {
							netimage.Image(c, media.Photos[i].URL, netimage.Options{
								MaxSide:     1200,
								Fit:         ui.Cover,
								Placeholder: t.Surface,
							}).Fill()
						})
						if slide.Clicked() {
							a.openNetworkPhotosViewer(tx.ComName, media.Photos, i)
						}
					}).Fill()
				} else if busy {
					ui.Box(c).Fill().Center().Children(func() {
						ui.Spinner(c).Label("加载图集中")
					})
				} else {
					ui.Box(c).Fill().Center().Children(func() {
						ui.Icon(c, iconBird).FontSize(44).TextColor(t.TextMuted.Alpha(0.28))
					})
				}

				// 顶部/底部柔和渐隐蒙版确保控件与页码清晰
				ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(78).LinearGradient(ui.LinearGradient{
					From:  ui.RGBA(0, 0, 0, 0.36),
					To:    ui.RGBA(0, 0, 0, 0),
					Angle: 180,
				})
				if count > 0 {
					ui.Box(c).Absolute().Left(0).Bottom(0).Right(0).Height(56).LinearGradient(ui.LinearGradient{
						From:  ui.RGBA(0, 0, 0, 0),
						To:    ui.RGBA(0, 0, 0, 0.44),
						Angle: 180,
					})
				}

				// 浮动返回按钮
				if glassCircleBtn(c, "gd-back", "返回", iconBack, 36).
					Absolute().Left(16).Top(topBtnY).Clicked() {
					a.goBack()
				}

				// 浮动右侧记日志胶囊
				logBtn := ui.Row(c.Key("gd-top-log")).Height(34).PaddingX(13).Radius(17).
					Background(ui.RGBA(0, 0, 0, 0.42)).Gap(5).AlignItems(ui.Center).
					Absolute().Right(16).Top(topBtnY).
					Role(ui.RoleButton).Label("记日志")
				logBtn.Children(func() {
					ui.Icon(c, iconPlus).FontSize(13.5).TextColor(ui.RGB(255, 255, 255))
					ui.Text(c, "记日志").FontSize(12.5).FontWeight(600).TextColor(ui.RGB(255, 255, 255))
				})
				if logBtn.Clicked() {
					a.OpenNewJournalModal(tx.SpeciesCode, "")
				}

				// 页码与指示点
				if count > 0 {
					idx := max(0, min(a.guideGallery.Page, count-1))
					ui.Textf(c, "%d / %d", idx+1, count).
						Absolute().Left(18).Bottom(14).
						FontSize(12).FontWeight(600).FontFeatures("tnum").
						TextColor(ui.RGBA(255, 255, 255, 0.92))
					if count > 1 {
						ui.Box(c).Absolute().Right(16).Bottom(14).Children(func() {
							mobile.PageDots(c, idx, count)
						})
					}
				}
			})

			// 2. 杂志社论风纯平排版正文
			ui.Column(c).FillWidth().Padding(20, 20, 0, 20).Gap(20).Children(func() {
				// 标题与分类信息
				ui.Column(c).FillWidth().Gap(5).Children(func() {
					ui.Text(c, tx.ComName).FontSize(26).FontWeight(700).LetterSpacing(-0.4)
					ui.Text(c, tx.SciName).FontSize(14).TextColor(t.TextMuted)

					meta := model.FormatTaxonLine(tx)
					if media != nil && media.INatObservationsCount > 0 {
						obsMeta := fmt.Sprintf("%d 条社区观测", media.INatObservationsCount)
						if meta != "" {
							meta += " · " + obsMeta
						} else {
							meta = obsMeta
						}
					}
					if meta != "" {
						ui.Text(c, meta).FontSize(13).TextColor(secondaryTextColor(t)).Padding(2, 0, 0, 0)
					}

					if count > 0 {
						idx := max(0, min(a.guideGallery.Page, count-1))
						curPhoto := media.Photos[idx]
						ui.Textf(c, "图源：%s · %s", curPhoto.Source, curPhoto.Attribution).
							FontSize(11.5).TextColor(t.TextMuted).SingleLine()
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
							ui.Icon(c, iconCheck).FontSize(13.5).TextColor(t.Success)
							ui.Text(c, fallbackText(firstText, "已在个人图谱中解锁")).
								FontSize(12.5).FontWeight(500).TextColor(t.Success)
						})
					}
				})

				minimalDivider(c, t)

				// 鸟鸣参考区（干净细线列表）
				ui.Column(c).FillWidth().Gap(10).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "鸟鸣参考").FontSize(16).FontWeight(600)
						if media != nil && len(media.Sounds) > 0 {
							ui.Textf(c, "%d 段", len(media.Sounds)).FontSize(12.5).TextColor(t.TextMuted)
						}
					})

					if media == nil && busy {
						ui.Text(c, "正在检索鸟鸣参考录音…").FontSize(13).TextColor(t.TextMuted)
					} else if media == nil || len(media.Sounds) == 0 {
						ui.Text(c, "暂无公开鸟鸣录音，可在「设置」补充 Xeno-canto Key 扩展曲库。").
							FontSize(13).TextColor(t.TextMuted).LineHeight(1.5)
					} else {
						pst := a.player.Status()
						ui.Column(c).FillWidth().Dividers(1, t.Border).Children(func() {
							for i, snd := range media.Sounds {
								snd := snd
								isThis := pst.SourceKey == snd.URL
								ui.Row(c.Key(fmt.Sprintf("gsnd-%d", i))).FillWidth().Padding(11, 0).Gap(12).AlignItems(ui.Center).Children(func() {
									btnIcon := iconPlay
									btnLbl := "播放"
									btnBg := t.Surface
									btnFg := t.Text
									if isThis && (pst.Playing || pst.Loading) {
										if pst.Playing {
											btnIcon = iconPause
											btnLbl = "暂停"
										}
										btnBg = t.Accent
										btnFg = t.AccentText
									}
									if circleIconBtn(c, fmt.Sprintf("gsnd-b-%d", i), btnLbl, btnIcon, btnBg, btnFg, 36).Clicked() {
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
										ui.Text(c, snd.Quality).FontSize(11.5).FontWeight(600).TextColor(t.TextMuted)
									}
								})
							}
						})
					}
				})

				minimalDivider(c, t)

				// 中文百科简介（纯留白排版，无多余边框卡片）
				ui.Column(c).FillWidth().Gap(10).Selectable().Children(func() {
					ui.Text(c, "物种简介").FontSize(16).FontWeight(600)
					if media == nil && busy {
						ui.Text(c, "正在加载中文维基百科简介…").FontSize(13).TextColor(t.TextMuted)
					} else if media != nil && media.Wiki != nil && media.Wiki.Extract != "" {
						ui.Text(c, media.Wiki.Extract).FontSize(15).LineHeight(1.75).TextColor(secondaryTextColor(t))
					} else {
						ui.Text(c, "中文维基百科暂未收录该鸟种的摘要说明。").FontSize(13.5).TextColor(t.TextMuted)
					}

					ui.Row(c).Gap(16).Wrap().Padding(4, 0, 0, 0).Children(func() {
						if media != nil && media.Wiki != nil && media.Wiki.PageURL != "" {
							ui.Link(c, "中文维基全文 ↗", media.Wiki.PageURL)
						}
						if media != nil && media.INatURL != "" {
							ui.Link(c, "iNaturalist ↗", media.INatURL)
						}
						ui.Link(c, "eBird 分布 ↗", fmt.Sprintf("https://ebird.org/species/%s", tx.SpeciesCode))
					})
				})

				if monochromeButton(c, "gd-cta-log", "记入观鸟日志并解锁", true, t).FillWidth().Height(48).Radius(24).Clicked() {
					a.OpenNewJournalModal(tx.SpeciesCode, "")
				}
			})
		})
	})
}
