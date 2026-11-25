package appui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *AppState) loadInitialData() {
	a.loadTaxonomyAsync(false)
}

// loadTaxonomyAsync 加载全球简体中文名录（优先读 24 小时缓存），并在有 Key 时拉取中国及默认省份鸟种列表。
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
				a.taxaErr = fmt.Sprintf("加载鸟类名录失败: %v", err)
			})
			return
		}

		// 若已配置 eBird Key，顺便拉取中国与当前默认省份物种代码集合
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
			a.refreshStorageStats()
		})
	}()
}

// SetTaxonomy 写入鸟类名录并重建目/科索引及筛选列表（同时供测试直接注入数据）。
func (a *AppState) SetTaxonomy(taxa []model.Taxon, chinaSet, regionSet map[string]bool, regionCode string) {
	a.allTaxa = taxa
	a.taxonByCode = make(map[string]model.Taxon, len(taxa))
	orderMap := make(map[string]bool)
	familyMap := make(map[string]bool)

	for _, tx := range taxa {
		a.taxonByCode[tx.SpeciesCode] = tx
		if tx.Order != "" {
			orderMap[tx.Order] = true
		}
		if tx.FamilyComName != "" {
			familyMap[tx.FamilyComName] = true
		}
	}

	if len(chinaSet) > 0 {
		a.chinaCodes = chinaSet
	} else if len(a.chinaCodes) == 0 {
		// 未配置 Key 时不阻塞图鉴使用，默认自动切到全球名录
		if a.guideScope == 0 || a.guideScope == 1 {
			a.guideScope = 2
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
	if len(a.filteredTaxa) > 0 && a.guideActiveCode == "" {
		a.selectGuideSpecies(0)
	}
}

// OpenSpeciesInGuide 从其他模块（如区域动态、热点、已解锁鸟种）跳转至图鉴并选中指定物种。
func (a *AppState) OpenSpeciesInGuide(speciesCode string) {
	a.page = "guide"
	tx, ok := a.taxonByCode[speciesCode]
	if !ok {
		a.invalidateUI()
		return
	}
	// 如果当前筛选范围内找不到，切到全球名录并清空搜索条件以确保命中
	foundIdx := -1
	for i, item := range a.filteredTaxa {
		if item.SpeciesCode == speciesCode {
			foundIdx = i
			break
		}
	}
	if foundIdx < 0 {
		a.guideScope = 2
		a.guideQuery = ""
		a.guideOrder = "全部目"
		a.guideFamily = "全部科"
		a.applyGuideFilter()
		for i, item := range a.filteredTaxa {
			if item.SpeciesCode == speciesCode {
				foundIdx = i
				break
			}
		}
	}
	if foundIdx >= 0 {
		a.selectGuideSpecies(foundIdx)
		a.guideList.ScrollIntoView(foundIdx)
	} else {
		a.guideActiveCode = tx.SpeciesCode
		a.ensureGuideMediaLoaded(tx)
	}
	a.invalidateUI()
}

func (a *AppState) applyGuideFilter() {
	q := strings.ToLower(strings.TrimSpace(a.guideQuery))
	out := make([]model.Taxon, 0, 256)

	for _, tx := range a.allTaxa {
		switch a.guideScope {
		case 0: // 中国鸟类
			if len(a.chinaCodes) > 0 && !a.chinaCodes[tx.SpeciesCode] {
				continue
			}
		case 1: // 当前省份
			if len(a.regionCodes) > 0 && !a.regionCodes[tx.SpeciesCode] {
				continue
			}
		case 3: // 已解锁
			if !a.store.IsUnlocked(tx.SpeciesCode) {
				continue
			}
		}
		if a.guideOrder != "" && a.guideOrder != "全部目" && tx.Order != a.guideOrder {
			continue
		}
		if a.guideFamily != "" && a.guideFamily != "全部科" && tx.FamilyComName != a.guideFamily {
			continue
		}
		if q != "" {
			match := strings.Contains(strings.ToLower(tx.ComName), q) ||
				strings.Contains(strings.ToLower(tx.SciName), q) ||
				strings.Contains(strings.ToLower(tx.SpeciesCode), q) ||
				strings.Contains(strings.ToLower(tx.FamilyComName), q)
			if !match {
				continue
			}
		}
		out = append(out, tx)
	}

	a.filteredTaxa = out
	a.guideSelected = -1
	for i, tx := range a.filteredTaxa {
		if tx.SpeciesCode == a.guideActiveCode {
			a.guideSelected = i
			break
		}
	}
}

func (a *AppState) selectGuideSpecies(idx int) {
	if idx < 0 || idx >= len(a.filteredTaxa) {
		return
	}
	tx := a.filteredTaxa[idx]
	a.guideSelected = idx
	a.guideActiveCode = tx.SpeciesCode
	a.guidePhotoIndex = 0
	a.ensureGuideMediaLoaded(tx)
}

func (a *AppState) ensureGuideMediaLoaded(tx model.Taxon) {
	if tx.SpeciesCode == "" {
		return
	}
	if media, ok := a.guideMediaMap[tx.SpeciesCode]; ok {
		for _, p := range media.Photos {
			a.loadImageAsync("guide:"+p.URL, p.URL, false)
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
			for _, p := range media.Photos {
				a.loadImageAsync("guide:"+p.URL, p.URL, false)
			}
			a.refreshStorageStats()
		})
	}()
}

func (a *AppState) viewGuide(c *ui.Context, t *ui.Theme) {
	ui.Split(c, &a.guideSplit,
		func() { a.viewGuideListPane(c, t) },
		func() { a.viewGuideDetailPane(c, t) },
	).SplitReset(380).Fill()
}

func (a *AppState) viewGuideListPane(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Fill().Children(func() {
		// 顶部筛选栏
		ui.Column(c).Padding(14, 14, 10, 14).Gap(10).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Text(c, "鸟类图鉴").FontSize(17).Bold()
				if a.taxaLoading {
					ui.Spinner(c).Label("正在更新名录")
				} else if ui.Button(c.Key("guide-refresh"), "").Label("刷新名录").Tooltip("刷新 eBird 鸟类名录").Children(func() {
					ui.Icon(c, iconRefresh).FontSize(14)
				}).Clicked() {
					a.loadTaxonomyAsync(true)
				}
			})

			if ui.Segmented(c.Key("guide-scope"), &a.guideScope, "中国鸟类", "当前省份", "全球名录", "已解锁").
				FillEqually().Label("名录范围").Changed() {
				a.applyGuideFilter()
			}

			if ui.SearchField(c.Key("guide-search"), &a.guideQuery).
				Placeholder("搜索中文名、拉丁学名或代码…").Label("搜索鸟种").Changed() {
				a.applyGuideFilter()
			}

			ui.Row(c).Gap(8).Children(func() {
				if len(a.guideOrders) > 0 {
					if ui.Select(c.Key("guide-order"), &a.guideOrder, a.guideOrders).Label("按目筛选").Grow(1).Changed() {
						a.applyGuideFilter()
					}
				}
				if len(a.guideFamilies) > 0 {
					if ui.Select(c.Key("guide-family"), &a.guideFamily, a.guideFamilies).Label("按科筛选").Grow(1).Changed() {
						a.applyGuideFilter()
					}
				}
			})

			if a.taxaErr != "" {
				ui.Text(c, a.taxaErr).FontSize(12).TextColor(t.Danger)
			} else {
				ui.Textf(c, "匹配 %d 种（共 %d 种）", len(a.filteredTaxa), len(a.allTaxa)).
					FontSize(12).TextColor(t.TextMuted)
			}
		})

		// 虚拟化物种列表
		a.guideList.Key = func(i int) any { return a.filteredTaxa[i].SpeciesCode }
		a.guideList.Label = func(i int) string { return a.filteredTaxa[i].ComName }
		a.guideList.Selected = &a.guideSelected

		list := ui.List(c.Key("guide-taxa-list"), &a.guideList, len(a.filteredTaxa)).Grow(1).FillWidth()
		list.Rows(func(row ui.ListRow) {
			tx := a.filteredTaxa[row.Index]
			rc := row.Context
			bg := ui.Transparent
			switch {
			case row.Selected():
				bg = t.Accent
			case row.Pressed():
				bg = t.SurfacePressed
			case row.Hovered():
				bg = t.SurfaceHover
			}

			textColor := t.Text
			mutedColor := t.TextMuted
			if row.Selected() {
				textColor = t.AccentText
				mutedColor = t.AccentText.Alpha(0.85)
			}

			ui.Row(rc).Padding(10, 14).Gap(8).AlignItems(ui.Center).Background(bg).Children(func() {
				ui.Column(rc).Grow(1).Gap(2).Children(func() {
					ui.Text(rc, tx.ComName).Bold().TextColor(textColor).SingleLine()
					ui.Textf(rc, "%s · %s", tx.SciName, fallbackText(tx.FamilyComName, tx.Order)).
						FontSize(11.5).TextColor(mutedColor).SingleLine()
				})
				if a.store.IsUnlocked(tx.SpeciesCode) {
					badge := ui.Badge(rc, "已解锁")
					if row.Selected() {
						badge.Background(ui.RGBA(255, 255, 255, 0.2)).TextColor(t.AccentText)
					} else {
						badge.Background(t.Success.Alpha(0.14)).TextColor(t.Success)
					}
				}
			})
		}).Children(func() {
			if len(a.filteredTaxa) == 0 && !a.taxaLoading {
				ui.Column(c).Fill().Center().Padding(24).Gap(6).Children(func() {
					ui.Text(c, "暂无匹配的鸟种").Bold()
					ui.Text(c, "可尝试切换名录范围或清除搜索关键词").FontSize(12.5).TextColor(t.TextMuted)
				})
			}
		})

		if list.Changed() && a.guideSelected >= 0 {
			a.selectGuideSpecies(a.guideSelected)
		}
	})
}

func (a *AppState) viewGuideDetailPane(c *ui.Context, t *ui.Theme) {
	if a.guideActiveCode == "" {
		ui.Column(c).Fill().Center().Gap(8).Padding(32).Children(func() {
			ui.Icon(c, iconBook).FontSize(32).TextColor(t.TextMuted)
			ui.Text(c, "从左侧列表选择一个鸟种").FontSize(15).Bold()
			ui.Text(c, "将按需在线加载高清实拍图、鸟鸣音频流与中文物种百科").FontSize(12.5).TextColor(t.TextMuted)
		})
		return
	}

	tx, ok := a.taxonByCode[a.guideActiveCode]
	if !ok {
		return
	}
	media := a.guideMediaMap[tx.SpeciesCode]
	busy := a.guideMediaBusy[tx.SpeciesCode]
	unlocked := a.store.IsUnlocked(tx.SpeciesCode)

	ui.Scroll(c.Key("guide-detail-scroll")).Fill().Padding(20).Gap(16).Children(func() {
		// 标题栏与快捷动作
		ui.Row(c).FillWidth().AlignItems(ui.Start).Justify(ui.SpaceBetween).Gap(12).Children(func() {
			ui.Column(c).Grow(1).Gap(4).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tx.ComName).FontSize(22).Bold()
					if unlocked {
						ui.Badge(c, "已解锁").Background(t.Success.Alpha(0.15)).TextColor(t.Success)
					}
				})
				ui.Text(c, tx.SciName).Italic().FontSize(14).TextColor(t.TextMuted)
			})

			ui.Row(c).Gap(8).Children(func() {
				if ui.PrimaryButton(c.Key("guide-add-journal"), "").Children(func() {
					ui.Icon(c, iconPlus).FontSize(14)
					ui.Text(c, "记入观鸟日志").SingleLine()
				}).Clicked() {
					a.openNewJournalForSpecies(tx.SpeciesCode)
				}
			})
		})

		// 分类阶元卡片
		ui.Row(c).Gap(8).Wrap().Children(func() {
			infoChip(c, t, "物种代码", tx.SpeciesCode)
			if tx.Order != "" {
				infoChip(c, t, "目", tx.Order)
			}
			if tx.FamilyComName != "" {
				infoChip(c, t, "科", tx.FamilyComName)
			} else if tx.FamilySciName != "" {
				infoChip(c, t, "科", tx.FamilySciName)
			}
			if media != nil && media.INatObservationsCount > 0 {
				infoChip(c, t, "iNaturalist 记录", fmt.Sprintf("%d 条", media.INatObservationsCount))
			}
		})

		// 高清图鉴画廊（在线按需加载）
		ui.Column(c).FillWidth().Padding(14).Gap(10).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "物种图鉴照片").Bold()
				if busy && media == nil {
					ui.Spinner(c).Label("加载照片中")
				} else if media != nil {
					ui.Textf(c, "在线加载 · %d 张", len(media.Photos)).FontSize(12).TextColor(t.TextMuted)
				}
			})

			if media == nil && busy {
				ui.Box(c).FillWidth().Height(220).Center().Radius(8).Background(t.Background).Children(func() {
					ui.Text(c, "正在按需获取照片外链…").TextColor(t.TextMuted)
				})
			} else if media == nil || len(media.Photos) == 0 {
				ui.Box(c).FillWidth().Height(140).Center().Radius(8).Background(t.Background).Children(func() {
					ui.Text(c, "暂无公开标准图鉴照片").TextColor(t.TextMuted)
				})
			} else {
				if a.guidePhotoIndex >= len(media.Photos) {
					a.guidePhotoIndex = 0
				}
				curPhoto := media.Photos[a.guidePhotoIndex]
				imgKey := "guide:" + curPhoto.URL
				a.loadImageAsync(imgKey, curPhoto.URL, false)

				if src, ok := a.imgCache[imgKey]; ok {
					ui.Image(c, src).FillWidth().Height(280).Fit(ui.Contain).Radius(8).
						Background(t.Background).Label(tx.ComName + "图鉴照片")
				} else if errMsg := a.imgErr[imgKey]; errMsg != "" {
					ui.Box(c).FillWidth().Height(200).Center().Radius(8).Background(t.Background).Children(func() {
						ui.Textf(c, "图片加载失败: %s", errMsg).FontSize(12).TextColor(t.TextMuted)
					})
				} else {
					ui.Box(c).FillWidth().Height(280).Center().Radius(8).Background(t.Background).Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Spinner(c).Label("正在下载图片")
							ui.Text(c, "正在加载高清图…").TextColor(t.TextMuted)
						})
					})
				}

				ui.Textf(c, "来源：%s · 署名：%s", curPhoto.Source, curPhoto.Attribution).
					FontSize(11.5).TextColor(t.TextMuted)

				if len(media.Photos) > 1 {
					ui.Row(c).Gap(6).Wrap().Children(func() {
						for i := range media.Photos {
							btn := ui.Button(c.Key(fmt.Sprintf("photo-tab-%d", i)), fmt.Sprintf("图 %d", i+1))
							if i == a.guidePhotoIndex {
								btn.Background(t.Accent).TextColor(t.AccentText)
							}
							if btn.Clicked() {
								a.guidePhotoIndex = i
							}
						}
					})
				}
			}
		})

		// 鸟鸣录音在线流式播放
		ui.Column(c).FillWidth().Padding(14).Gap(10).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "鸟鸣参考录音").Bold()
				if media != nil {
					ui.Textf(c, "%d 段在线音频", len(media.Sounds)).FontSize(12).TextColor(t.TextMuted)
				}
			})

			if media == nil && busy {
				ui.Text(c, "正在检索鸟鸣录音…").FontSize(12.5).TextColor(t.TextMuted)
			} else if media == nil || len(media.Sounds) == 0 {
				ui.Text(c, "暂无可直连播放的公开鸟鸣录音（可在「设置」中配置 Xeno-canto v3 Key 扩展曲库）").
					FontSize(12.5).TextColor(t.TextMuted)
			} else {
				playerStatus := a.player.Status()
				for i, snd := range media.Sounds {
					isThis := playerStatus.SourceKey == snd.URL
					ui.Row(c.Key(fmt.Sprintf("snd-%d", i))).FillWidth().Padding(10, 12).Gap(10).
						AlignItems(ui.Center).Radius(8).Background(t.Background).Children(func() {
						btnIcon := iconPlay
						btnLabel := "播放"
						if isThis && playerStatus.Playing {
							btnIcon = iconPause
							btnLabel = "暂停"
						}
						if ui.Button(c.Key(fmt.Sprintf("snd-btn-%d", i)), "").Label(btnLabel).Children(func() {
							ui.Icon(c, btnIcon).FontSize(13)
						}).Clicked() {
							if isThis && (playerStatus.Playing || playerStatus.Loading) {
								a.player.TogglePause()
							} else {
								a.player.PlayURL(snd.URL, tx.ComName+" · "+snd.Type, snd.Source+" · "+snd.Place, a.cache, a.client, func() {
									a.updateUI(func() {})
								})
							}
						}

						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Textf(c, "%s · %s", snd.Type, snd.Source).Bold().FontSize(13)
							ui.Textf(c, "%s · 录音：%s", snd.Place, snd.Recordist).
								FontSize(11.5).TextColor(t.TextMuted).SingleLine()
						})
						if snd.Quality != "" {
							ui.Badge(c, snd.Quality)
						}
					})
				}
			}
		})

		// 中文维基百科摘要
		ui.Column(c).FillWidth().Padding(14).Gap(8).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Selectable().Children(func() {
			ui.Text(c, "物种百科简介").Bold()
			if media == nil && busy {
				ui.Text(c, "正在获取中文维基百科摘要…").FontSize(12.5).TextColor(t.TextMuted)
			} else if media != nil && media.Wiki != nil && media.Wiki.Extract != "" {
				ui.Text(c, media.Wiki.Extract).LineHeight(1.6)
				ui.Row(c).Gap(16).Padding(4, 0, 0, 0).Children(func() {
					if media.Wiki.PageURL != "" {
						ui.Link(c, "打开中文维基完整词条", media.Wiki.PageURL)
					}
					if media.INatURL != "" {
						ui.Link(c, "在 iNaturalist 查看物种页", media.INatURL)
					}
					if ui.Link(c, "在 eBird 查看分布图", fmt.Sprintf("https://ebird.org/species/%s", tx.SpeciesCode)).Clicked() {
						// Link 自身会打开外部浏览器
					}
				})
			} else {
				ui.Text(c, "中文维基暂未收录该物种摘要。").FontSize(12.5).TextColor(t.TextMuted)
				if ui.Button(c.Key("open-ebird-sp"), "在浏览器打开 eBird 物种页").Clicked() {
					_ = mygo.Shell.OpenExternal(fmt.Sprintf("https://ebird.org/species/%s", tx.SpeciesCode))
				}
			}
		})
	})
}

func (a *AppState) openNewJournalForSpecies(speciesCode string) {
	a.journalPrefillCode = speciesCode
	a.page = "journal"
}

func infoChip(c *ui.Context, t *ui.Theme, label, val string) {
	ui.Row(c).Gap(6).Padding(5, 10).Radius(6).Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Text(c, label).FontSize(11.5).TextColor(t.TextMuted)
		ui.Text(c, val).FontSize(12).Bold()
	})
}

func fallbackText(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
