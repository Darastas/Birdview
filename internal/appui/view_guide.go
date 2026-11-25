package appui

import (
	"context"
	"fmt"
	"sort"
	"strings"
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

		key := a.store.LoadEbirdKey()
		defaultRegion := a.store.Settings().DefaultRegion
		chinaSet := make(map[string]bool)
		regionSet := make(map[string]bool)
		if key != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if codes, e := a.client.FetchSpeciesList(ctx, key, "CN"); e == nil {
				for _, code := range codes {
					chinaSet[code] = true
				}
			}
			if codes, e := a.client.FetchSpeciesList(ctx, key, defaultRegion); e == nil {
				for _, code := range codes {
					regionSet[code] = true
				}
			}
			cancel()
		}

		a.updateUI(func() {
			a.SetTaxonomy(taxa, chinaSet, regionSet, defaultRegion)
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
				strings.Contains(strings.ToLower(tx.SpeciesCode), q) ||
				strings.Contains(strings.ToLower(famZh), q) ||
				strings.Contains(strings.ToLower(ordZh), q)
			if !match {
				continue
			}
		}
		out = append(out, tx)
	}
	a.filteredTaxa = out
}

func (a *AppState) ensureGuideMediaLoaded(tx model.Taxon) {
	if tx.SpeciesCode == "" {
		return
	}
	if media, ok := a.guideMediaMap[tx.SpeciesCode]; ok {
		if len(media.Photos) > 0 && a.thumbMap[tx.SpeciesCode] == "" {
			a.thumbMap[tx.SpeciesCode] = fallbackText(media.Photos[0].ThumbURL, media.Photos[0].URL)
		}
		return
	}
	if a.guideMediaBusy[tx.SpeciesCode] {
		return
	}
	a.guideMediaBusy[tx.SpeciesCode] = true
	xcKey := a.store.LoadXenoKey()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
		// 顶部标题与搜索筛选区
			ui.Column(c).FillWidth().Padding(12, 16, 8, 16).Gap(10).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "鸟类图鉴").FontSize(26).Bold().Role(ui.RoleHeading).Level(1)
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if a.taxaLoading {
							ui.Spinner(c).Label("更新名录中")
						}
						btnBg := t.Surface
						btnFg := t.Text
						lbl := "目科筛选"
						if hasFilter {
							btnBg = t.Accent.Alpha(0.14)
							btnFg = t.Accent
							lbl = fallbackText(strings.TrimPrefix(a.guideFamily, "全部科"), a.guideOrder)
							if lbl == "" {
								lbl = a.guideOrder
							}
						}
						filterBtn := ui.Row(c.Key("guide-filter-open")).Height(34).PaddingX(12).Radius(17).
							Background(btnBg).Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("目科筛选")
						if !t.Dark && !hasFilter {
							filterBtn.Shadow(0, 1, 4, 0, ui.RGBA(15, 23, 42, 0.06))
						}
						filterBtn.Children(func() {
							ui.Icon(c, iconFilter).FontSize(13).TextColor(btnFg)
							ui.Text(c, lbl).FontSize(12.5).FontWeight(600).TextColor(btnFg).SingleLine()
						})
						if filterBtn.Clicked() {
							a.guideFilterSheet = true
						}
					})
				})

				if ui.SearchField(c.Key("guide-search"), &a.session.GuideQuery).
					Placeholder("搜索中文名、拉丁学名或科名…").Label("搜索鸟种").
					FillWidth().Height(40).Radius(12).Changed() {
					a.applyGuideFilter()
				}

				ui.ScrollHorizontal(c.Key("guide-scopes")).FillWidth().Gap(8).Children(func() {
					cnLabel := "中国鸟类"
					if len(a.chinaCodes) > 0 {
						cnLabel = fmt.Sprintf("中国 %d", len(a.chinaCodes))
					}
					if filterPill(c, "scope-cn", cnLabel, a.session.GuideScope == 0, t).Clicked() {
						a.session.GuideScope = 0
						a.applyGuideFilter()
					}
					provLabel := "当前省份"
					if a.regionCodesFor != "" {
						provLabel = regionShortNameByCode(a.regionCodesFor)
					}
					if filterPill(c, "scope-prov", provLabel, a.session.GuideScope == 1, t).Clicked() {
						a.session.GuideScope = 1
						a.applyGuideFilter()
					}
					ulLabel := fmt.Sprintf("已解锁 %d", len(a.store.UnlockedBirds()))
					if filterPill(c, "scope-ul", ulLabel, a.session.GuideScope == 3, t).Clicked() {
						a.session.GuideScope = 3
						a.applyGuideFilter()
					}
					allLabel := "全球"
					if len(a.allTaxa) > 0 {
						allLabel = fmt.Sprintf("全球 %d", len(a.allTaxa))
					}
					if filterPill(c, "scope-all", allLabel, a.session.GuideScope == 2, t).Clicked() {
						a.session.GuideScope = 2
						a.applyGuideFilter()
					}
				})

				if a.taxaErr != "" {
					ui.Row(c).FillWidth().Padding(8, 12).Radius(10).Background(t.Danger.Alpha(0.1)).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, a.taxaErr).FontSize(12.5).TextColor(t.Danger).Grow(1)
						if ui.Button(c.Key("taxa-retry"), "重试").Clicked() {
							a.loadTaxonomyAsync(true)
						}
					})
				} else {
					ui.Textf(c, "共 %d 个物种", len(a.filteredTaxa)).FontSize(12).TextColor(t.TextMuted)
				}
			})

			// 虚拟化物种卡片列表
			a.guideList.Key = func(i int) any { return a.filteredTaxa[i].SpeciesCode }
			a.guideList.Label = func(i int) string { return a.filteredTaxa[i].ComName }

			list := ui.List(c.Key("guide-taxa-list"), &a.guideList, len(a.filteredTaxa)).
				Grow(1).Basis(0).FillWidth()
			if list.PullToRefresh(&a.guideRefresh) {
				a.loadTaxonomyAsync(true)
			}
			list.Rows(func(row ui.ListRow) {
				tx := a.filteredTaxa[row.Index]
				rc := row.Context
				unlocked := a.store.IsUnlocked(tx.SpeciesCode)

				ui.Box(rc).Padding(5, 16).Children(func() {
					card := elevatedRowCard(rc, t).Padding(10, 12).Gap(12).AlignItems(ui.Center).
						Role(ui.RoleButton).Label(tx.ComName)
					if row.Pressed() {
						card.Background(t.SurfaceHover)
					}
					card.Children(func() {
						thumbURL := a.speciesThumbURL(tx.SpeciesCode)
						if thumbURL != "" {
							netimage.Image(rc.Key("gt-"+tx.SpeciesCode), thumbURL, netimage.Options{
								MaxSide:     180,
								Fit:         ui.Cover,
								Placeholder: softPillColor(t),
							}).Size(60, 60).Radius(12).Shrink(0)
						} else {
							a.queueSpeciesThumb(tx)
							ui.Box(rc).Size(60, 60).Radius(12).Background(softPillColor(t)).Center().Shrink(0).Children(func() {
								ui.Icon(rc, iconBook).FontSize(22).TextColor(t.TextMuted.Alpha(0.55))
							})
						}

						ui.Column(rc).Grow(1).Basis(0).Gap(3).Children(func() {
							ui.Row(rc).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Text(rc, tx.ComName).FontSize(16).FontWeight(600).SingleLine()
								if unlocked {
									ui.Box(rc).Size(17, 17).Radius(8.5).Background(t.Success).Center().Shrink(0).Children(func() {
										ui.Icon(rc, iconCheck).FontSize(10.5).TextColor(ui.RGB(255, 255, 255))
									})
								}
							})
							ui.Text(rc, tx.SciName).Italic().FontSize(12.5).TextColor(t.TextMuted).SingleLine()
							ui.Text(rc, model.FormatTaxonLine(tx)).FontSize(12).TextColor(secondaryTextColor(t)).SingleLine()
						})

						ui.Icon(rc, iconChevronRight).FontSize(16).TextColor(t.TextMuted.Alpha(0.5))
					})
					if card.Clicked() {
						a.OpenSpeciesInGuide(tx.SpeciesCode)
					}
				})
			}).Children(func() {
				if len(a.filteredTaxa) == 0 && !a.taxaLoading {
					ui.Column(c).Fill().Center().Padding(28).Gap(8).Children(func() {
						ui.Icon(c, iconBook).FontSize(34).TextColor(t.TextMuted.Alpha(0.6))
						ui.Text(c, "暂无匹配的鸟种").FontSize(16).FontWeight(600)
						ui.Text(c, "可尝试切换范围或清除筛选条件").FontSize(13).TextColor(t.TextMuted)
						if hasFilter || a.session.GuideQuery != "" {
							if ui.Button(c.Key("guide-reset-empty"), "清除筛选条件").Clicked() {
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
	mobile.Sheet(c, &a.guideFilterSheet, mobile.SheetOptions{Detents: []float32{0.68, 0.9}, Label: "目科筛选"}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 16, 12, 16).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				if ui.Button(c.Key("gf-reset"), "重置").Clicked() {
					a.guideOrder = "全部目"
					a.guideFamily = "全部科"
					a.applyGuideFilter()
				}
				ui.Text(c, "分类目科筛选").FontSize(16).FontWeight(600)
				if ui.PrimaryButton(c.Key("gf-done"), "完成").Radius(16).Clicked() {
					a.guideFilterSheet = false
				}
			})

			ui.Scroll(c.Key("gf-scroll")).Grow(1).Basis(0).FillWidth().Padding(0, 16, 24, 16).Gap(16).Children(func() {
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, "按目筛选").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
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

				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, "按科筛选").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
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
			ui.Text(c, "未在名录中找到该鸟种").Bold()
			if ui.Button(c.Key("gd-nf-back"), "返回图鉴").Clicked() {
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

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		// 顶部详情导航栏
		ui.Row(c).FillWidth().Padding(10, 16, 8, 16).Gap(10).AlignItems(ui.Center).Children(func() {
			if circleIconBtn(c, "gd-back", "返回", iconBack, t.Surface, t.Text, 38).Clicked() {
				a.goBack()
			}
			ui.Text(c, tx.ComName).FontSize(17).FontWeight(600).SingleLine().Grow(1)
			logBtn := ui.Row(c.Key("gd-top-log")).Height(34).PaddingX(14).Radius(17).
				Background(t.Accent).Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("记日志")
			logBtn.Children(func() {
				ui.Icon(c, iconPlus).FontSize(13).TextColor(t.AccentText)
				ui.Text(c, "记日志").FontSize(13).FontWeight(600).TextColor(t.AccentText).SingleLine()
			})
			if logBtn.Clicked() {
				a.OpenNewJournalModal(tx.SpeciesCode, "")
			}
		})

		ui.Scroll(c.Key("gd-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(0, 0, 28, 0).Gap(14).Children(func() {
				// 1. 全宽滑动摄影图集（4:3 比例）
				if media == nil && busy {
					ui.Box(c).FillWidth().Height(240).Background(softPillColor(t)).Center().Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Spinner(c).Label("获取图集")
							ui.Text(c, "正在加载高清图集…").FontSize(13).TextColor(t.TextMuted)
						})
					})
				} else if media != nil && len(media.Photos) > 0 {
					count := len(media.Photos)
					ui.Box(c).FillWidth().Height(264).Background(softPillColor(t)).Children(func() {
						mobile.PageView(c, &a.guideGallery, count, func(i int) {
							p := media.Photos[i]
							slide := ui.Box(c.Key(fmt.Sprintf("gslide-%s-%d", code, i))).Fill().
								Role(ui.RoleButton).Label(fmt.Sprintf("放大查看第 %d 张照片", i+1))
							slide.Children(func() {
								netimage.Image(c, p.URL, netimage.Options{
									MaxSide:     1200,
									Fit:         ui.Cover,
									Placeholder: softPillColor(t),
								}).Fill()
							})
							if slide.Clicked() {
								a.openNetworkPhotosViewer(tx.ComName, media.Photos, i)
							}
						}).Fill()

						ui.Textf(c, "%d / %d", a.guideGallery.Page+1, count).
							Absolute().Right(12).Bottom(12).Padding(4, 10).Radius(999).
							FontSize(11.5).FontWeight(600).
							Background(ui.RGBA(15, 23, 42, 0.58)).TextColor(ui.RGB(255, 255, 255))
					})

					idx := max(0, min(a.guideGallery.Page, count-1))
					curPhoto := media.Photos[idx]
					ui.Row(c).FillWidth().PaddingX(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Children(func() {
						ui.Textf(c, "%s · %s", curPhoto.Source, curPhoto.Attribution).
							FontSize(11.5).TextColor(t.TextMuted).SingleLine().Grow(1)
						if count > 1 {
							mobile.PageDots(c, a.guideGallery.Page, count)
						}
					})
				} else {
					ui.Box(c).PaddingX(16).Children(func() {
						ui.Box(c).FillWidth().Height(130).Radius(16).Background(softPillColor(t)).Center().Children(func() {
							ui.Text(c, "暂无公开标准图鉴照片").FontSize(13).TextColor(t.TextMuted)
						})
					})
				}

				// 2. 身份信息与解锁状态卡片
				ui.Box(c).PaddingX(16).Children(func() {
					elevatedCard(c, t).Padding(16).Gap(10).Children(func() {
						ui.Column(c).Gap(3).Children(func() {
							ui.Text(c, tx.ComName).FontSize(23).Bold()
							ui.Text(c, tx.SciName).Italic().FontSize(14).TextColor(t.TextMuted)
						})

						ui.Row(c).Gap(8).Wrap().Children(func() {
							if ord := model.OrderZh(tx.Order); ord != "" {
								statusPill(c, ord, softPillColor(t), secondaryTextColor(t))
							}
							if fam := model.FamilyZh(tx.FamilySciName, tx.FamilyComName); fam != "" {
								statusPill(c, fam, softPillColor(t), secondaryTextColor(t))
							}
							statusPill(c, tx.SpeciesCode, softPillColor(t), t.TextMuted)
							if media != nil && media.INatObservationsCount > 0 {
								statusPill(c, fmt.Sprintf("%d 条社区观测", media.INatObservationsCount), t.Accent.Alpha(0.12), t.Accent)
							}
						})

						if unlocked {
							var firstText string
							for _, ub := range a.store.UnlockedBirds() {
								if ub.SpeciesCode == tx.SpeciesCode {
									firstText = fmt.Sprintf("首次解锁于 %s · %s（累计 %d 次）",
										model.FormatObsTime(ub.FirstSeenAt), fallbackText(ub.FirstLocation, "野外现场"), ub.TotalSightings)
									break
								}
							}
							ui.Row(c).FillWidth().Padding(10, 12).Radius(12).Background(t.Success.Alpha(0.12)).Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, iconCheck).FontSize(15).TextColor(t.Success)
								ui.Text(c, fallbackText(firstText, "已在个人图谱中解锁")).FontSize(12.5).FontWeight(600).TextColor(t.Success).Grow(1)
							})
						}
					})
				})

				// 3. 鸟鸣参考录音区
				ui.Column(c).PaddingX(16).Gap(8).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "鸟鸣参考录音").FontSize(16).FontWeight(600)
						if media != nil && len(media.Sounds) > 0 {
							ui.Textf(c, "%d 段", len(media.Sounds)).FontSize(12.5).TextColor(t.TextMuted)
						}
					})

					if media == nil && busy {
						elevatedCard(c, t).Padding(14).Children(func() {
							ui.Text(c, "正在检索鸟鸣参考录音…").FontSize(13).TextColor(t.TextMuted)
						})
					} else if media == nil || len(media.Sounds) == 0 {
						elevatedCard(c, t).Padding(14).Children(func() {
							ui.Text(c, "暂无可播放的公开鸟鸣录音（可在「设置」补充 Xeno-canto API Key 扩展曲库）").
								FontSize(13).TextColor(t.TextMuted)
						})
					} else {
						pst := a.player.Status()
						for i, snd := range media.Sounds {
							snd := snd
							isThis := pst.SourceKey == snd.URL
							elevatedRowCard(c.Key(fmt.Sprintf("gsnd-%d", i)), t).Padding(12).Gap(12).AlignItems(ui.Center).Children(func() {
								btnIcon := iconPlay
								btnLbl := "播放"
								if isThis && pst.Playing {
									btnIcon = iconPause
									btnLbl = "暂停"
								}
								if circleIconBtn(c, fmt.Sprintf("gsnd-b-%d", i), btnLbl, btnIcon, t.Accent, t.AccentText, 40).Clicked() {
									if isThis && (pst.Playing || pst.Loading) {
										a.player.TogglePause()
									} else {
										a.player.PlayURL(snd.URL, tx.ComName+" · "+snd.Type, snd.Source+" · "+snd.Place, a.cache, a.client, func() {
											a.updateUI(func() {})
										})
									}
								}
								ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
									ui.Textf(c, "%s · %s", snd.Type, snd.Source).FontSize(14).FontWeight(600).SingleLine()
									ui.Textf(c, "%s · 录音师：%s", snd.Place, snd.Recordist).FontSize(12).TextColor(t.TextMuted).SingleLine()
								})
								if snd.Quality != "" {
									statusPill(c, snd.Quality, softPillColor(t), secondaryTextColor(t))
								}
							})
						}
					}
				})

				// 4. 中文百科简介
				ui.Column(c).PaddingX(16).Gap(8).Children(func() {
					ui.Text(c, "物种百科").FontSize(16).FontWeight(600)
					elevatedCard(c, t).Padding(16).Gap(10).Selectable().Children(func() {
						if media == nil && busy {
							ui.Text(c, "正在加载中文维基百科简介…").FontSize(13).TextColor(t.TextMuted)
						} else if media != nil && media.Wiki != nil && media.Wiki.Extract != "" {
							ui.Text(c, media.Wiki.Extract).FontSize(14.5).LineHeight(1.62)
						} else {
							ui.Text(c, "中文维基百科暂未收录该鸟种的摘要说明。").FontSize(13.5).TextColor(t.TextMuted)
						}

						ui.Row(c).Gap(14).Wrap().Padding(4, 0, 0, 0).Children(func() {
							if media != nil && media.Wiki != nil && media.Wiki.PageURL != "" {
								ui.Link(c, "中文维基全文 ↗", media.Wiki.PageURL)
							}
							if media != nil && media.INatURL != "" {
								ui.Link(c, "iNaturalist ↗", media.INatURL)
							}
							ui.Link(c, "eBird 分布 ↗", fmt.Sprintf("https://ebird.org/species/%s", tx.SpeciesCode))
						})
					})
				})

				// 5. 底部记录并解锁 CTA
				ui.Box(c).Padding(4, 16, 8, 16).Children(func() {
					cta := ui.PrimaryButton(c.Key("gd-cta-log"), "记入观鸟日志并解锁").
						FillWidth().Height(48).Radius(24)
					if cta.Clicked() {
						a.OpenNewJournalModal(tx.SpeciesCode, "")
					}
				})
			})
		})
	})
}
