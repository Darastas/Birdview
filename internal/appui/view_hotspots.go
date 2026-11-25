package appui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// LoadHotspots 拉取指定省市的 eBird 观鸟热点列表。
func (a *AppState) LoadHotspots() {
	regionCode := regionCodeByLabel(a.hsRegionLabel)
	if regionCode == "" {
		regionCode = "CN-11"
	}
	key := a.store.LoadEbirdKey()
	if key == "" {
		a.hsErr = "查询观鸟热点需要配置 eBird API Key，请前往「设置」页面填写"
		a.hsAll = nil
		a.hsFiltered = nil
		a.invalidateUI()
		return
	}

	a.hsBusy = true
	a.hsErr = ""
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		list, err := a.client.FetchHotspots(ctx, key, regionCode, 30)
		cancel()

		a.updateUI(func() {
			a.hsBusy = false
			if err != nil {
				a.hsErr = fmt.Sprintf("加载观鸟热点失败: %v", err)
				a.hsAll = nil
				a.hsFiltered = nil
				return
			}
			a.hsAll = list
			a.applyHotspotFilterAndSort()
			if len(a.hsFiltered) > 0 {
				a.selectHotspot(0)
			} else {
				a.hsSelected = -1
				a.hsActiveLocID = ""
			}
		})
	}()
}

func (a *AppState) applyHotspotFilterAndSort() {
	q := strings.ToLower(strings.TrimSpace(a.hsQuery))
	out := make([]model.Hotspot, 0, len(a.hsAll))
	for _, h := range a.hsAll {
		if q != "" && !strings.Contains(strings.ToLower(h.LocName), q) && !strings.Contains(strings.ToLower(h.LocID), q) {
			continue
		}
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		if a.hsSortMode == 1 {
			return out[i].LatestObsDt > out[j].LatestObsDt
		}
		if out[i].NumSpeciesAllTime != out[j].NumSpeciesAllTime {
			return out[i].NumSpeciesAllTime > out[j].NumSpeciesAllTime
		}
		return out[i].LatestObsDt > out[j].LatestObsDt
	})
	a.hsFiltered = out
	a.hsSelected = -1
	for i, h := range out {
		if h.LocID == a.hsActiveLocID {
			a.hsSelected = i
			break
		}
	}
}

func (a *AppState) selectHotspot(idx int) {
	if idx < 0 || idx >= len(a.hsFiltered) {
		return
	}
	h := a.hsFiltered[idx]
	a.hsSelected = idx
	a.hsActiveLocID = h.LocID

	if _, ok := a.hsObsMap[h.LocID]; ok || a.hsObsBusy[h.LocID] {
		return
	}
	key := a.store.LoadEbirdKey()
	if key == "" {
		return
	}
	a.hsObsBusy[h.LocID] = true

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		obs, err := a.client.FetchRecentObservations(ctx, key, h.LocID, 30, 50, false)
		cancel()

		a.updateUI(func() {
			delete(a.hsObsBusy, h.LocID)
			if err == nil {
				a.hsObsMap[h.LocID] = obs
			}
		})
	}()
}

func (a *AppState) viewHotspots(c *ui.Context, t *ui.Theme) {
	ui.Split(c, &a.hotspotSplit,
		func() { a.viewHotspotsListPane(c, t) },
		func() { a.viewHotspotsDetailPane(c, t) },
	).SplitReset(380).Fill()
}

func (a *AppState) viewHotspotsListPane(c *ui.Context, t *ui.Theme) {
	regionLabels := make([]string, 0, len(api.ChinaRegions))
	for _, r := range api.ChinaRegions {
		regionLabels = append(regionLabels, r.Name)
	}

	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(14, 14, 10, 14).Gap(10).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "观鸟热点探索").FontSize(17).Bold()
				if a.hsBusy {
					ui.Spinner(c).Label("加载热点中")
				} else if ui.Button(c.Key("hs-refresh"), "").Label("刷新热点").Children(func() {
					ui.Icon(c, iconRefresh).FontSize(14)
				}).Clicked() {
					a.LoadHotspots()
				}
			})

			if ui.Select(c.Key("hs-region"), &a.hsRegionLabel, regionLabels).Label("选择地区").FillWidth().Changed() {
				a.LoadHotspots()
			}

			if ui.Segmented(c.Key("hs-sort"), &a.hsSortMode, "按累计鸟种数", "按最近活跃日期").
				FillEqually().Label("排序方式").Changed() {
				a.applyHotspotFilterAndSort()
			}

			if ui.SearchField(c.Key("hs-search"), &a.hsQuery).
				Placeholder("搜索热点名称或 ID…").Label("搜索热点").Changed() {
				a.applyHotspotFilterAndSort()
			}

			if a.hsErr == "" {
				ui.Textf(c, "共 %d 个近 30 天活跃热点", len(a.hsFiltered)).FontSize(12).TextColor(t.TextMuted)
			}
		})

		if a.hsErr != "" {
			ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
				ui.Text(c, a.hsErr).FontSize(13).TextColor(t.Danger)
				if a.store.LoadEbirdKey() == "" {
					if ui.PrimaryButton(c.Key("hs-goto-set"), "前往设置 API Key").Clicked() {
						a.SetPage("settings")
					}
				}
			})
			return
		}

		a.hsList.Key = func(i int) any { return a.hsFiltered[i].LocID }
		a.hsList.Label = func(i int) string { return a.hsFiltered[i].LocName }
		a.hsList.Selected = &a.hsSelected

		list := ui.List(c.Key("hs-list"), &a.hsList, len(a.hsFiltered)).Grow(1).FillWidth()
		list.Rows(func(row ui.ListRow) {
			h := a.hsFiltered[row.Index]
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
					ui.Text(rc, h.LocName).Bold().TextColor(textColor).SingleLine()
					ui.Textf(rc, "%s · 最近记录：%s", h.LocID, fallbackText(h.LatestObsDt, "-")).
						FontSize(11.5).TextColor(mutedColor).SingleLine()
				})
				badge := ui.Badge(rc, fmt.Sprintf("%d 种", h.NumSpeciesAllTime))
				if row.Selected() {
					badge.Background(ui.RGBA(255, 255, 255, 0.2)).TextColor(t.AccentText)
				}
			})
		}).Children(func() {
			if len(a.hsFiltered) == 0 && !a.hsBusy {
				ui.Column(c).Fill().Center().Padding(24).Children(func() {
					ui.Text(c, "暂无符合条件的观鸟热点").TextColor(t.TextMuted)
				})
			}
		})

		if list.Changed() && a.hsSelected >= 0 {
			a.selectHotspot(a.hsSelected)
		}
	})
}

func (a *AppState) viewHotspotsDetailPane(c *ui.Context, t *ui.Theme) {
	if a.hsActiveLocID == "" || a.hsSelected < 0 || a.hsSelected >= len(a.hsFiltered) {
		ui.Column(c).Fill().Center().Gap(8).Padding(32).Children(func() {
			ui.Icon(c, iconMapPin).FontSize(32).TextColor(t.TextMuted)
			ui.Text(c, "选择一个观鸟热点").FontSize(15).Bold()
			ui.Text(c, "可查看该地点历史累计鸟种数、近期目击鸟种并快速在此写观鸟日志").FontSize(12.5).TextColor(t.TextMuted)
		})
		return
	}

	h := a.hsFiltered[a.hsSelected]
	obsList := a.hsObsMap[h.LocID]
	busy := a.hsObsBusy[h.LocID]

	ui.Scroll(c.Key("hs-detail-scroll")).Fill().Padding(20).Gap(16).Children(func() {
		ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Start).Gap(12).Children(func() {
			ui.Column(c).Grow(1).Gap(4).Children(func() {
				ui.Text(c, h.LocName).FontSize(20).Bold()
				ui.Textf(c, "eBird 热点编号：%s · 坐标 %.4f, %.4f", h.LocID, h.Lat, h.Lng).
					FontSize(13).TextColor(t.TextMuted)
			})
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c.Key("hs-open-web"), "浏览器打开热点").Clicked() {
					_ = mygo.Shell.OpenExternal("https://ebird.org/hotspot/" + h.LocID)
				}
				if ui.PrimaryButton(c.Key("hs-new-log"), "在此处写日志").Clicked() {
					a.OpenNewJournalModal("", h.LocName)
				}
			})
		})

		ui.Row(c).Gap(8).Wrap().Children(func() {
			infoChip(c, t, "历史累计鸟种", fmt.Sprintf("%d 种", h.NumSpeciesAllTime))
			infoChip(c, t, "最近观测日期", fallbackText(h.LatestObsDt, "-"))
			infoChip(c, t, "地区代码", h.Subnational1Code)
			if ui.Button(c.Key("hs-to-regional"), "在区域动态中查看该热点明细 →").Clicked() {
				a.regCustomCode = h.LocID
				a.regSourceTab = 0
				a.page = "regional"
				a.LoadRegionalData()
			}
		})

		ui.Column(c).FillWidth().Padding(14).Gap(10).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "该热点近 30 天目击鸟种").Bold()
				if busy {
					ui.Spinner(c).Label("加载热点近期鸟种")
				} else {
					ui.Textf(c, "%d 条近期记录", len(obsList)).FontSize(12).TextColor(t.TextMuted)
				}
			})

			if busy && len(obsList) == 0 {
				ui.Text(c, "正在加载该热点的近期目击记录…").FontSize(12.5).TextColor(t.TextMuted)
			} else if len(obsList) == 0 {
				ui.Text(c, "该热点近 30 天暂无公开鸟种记录。").FontSize(12.5).TextColor(t.TextMuted)
			} else {
				for i, o := range obsList {
					obs := o
					ui.Row(c.Key(fmt.Sprintf("hs-obs-%d", i))).FillWidth().Padding(10, 12).Gap(10).
						AlignItems(ui.Center).Radius(8).Background(t.Background).Children(func() {
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Text(c, obs.ComName).Bold().FontSize(13.5)
								ui.Text(c, obs.SciName).Italic().FontSize(12).TextColor(t.TextMuted)
							})
							cnt := "在场"
							if obs.HowMany > 0 {
								cnt = fmt.Sprintf("%d 只", obs.HowMany)
							}
							ui.Textf(c, "%s · 数量：%s", obs.ObsDt, cnt).FontSize(11.5).TextColor(t.TextMuted)
						})

						ui.Row(c).Gap(6).Children(func() {
							if ui.Button(c.Key(fmt.Sprintf("hs-log-%d", i)), "记日志").Clicked() {
								a.OpenNewJournalModal(obs.SpeciesCode, h.LocName)
							}
							if ui.Button(c.Key(fmt.Sprintf("hs-sp-%d", i)), "图鉴").Clicked() {
								a.OpenSpeciesInGuide(obs.SpeciesCode)
							}
						})
					})
				}
			}
		})
	})
}
