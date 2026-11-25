package appui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"
	"github.com/egoist/mygo/ui/netimage"
)

func parseLatLng(raw string) (float64, float64, bool) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), "，", ",")
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, false
	}
	return lat, lng, true
}

func (a *AppState) currentExploreRegionCode() (code string, lat, lng float64, isGeo bool) {
	custom := strings.TrimSpace(a.regCustomCode)
	lat, lng, isGeo = parseLatLng(custom)
	code = custom
	if code == "" || isGeo {
		code = regionCodeByLabel(a.regRegionLabel)
	}
	if code == "" {
		code = "CN-11"
	}
	a.regionalRegion = code
	return code, lat, lng, isGeo
}

func (a *AppState) exploreRegionButtonText() string {
	custom := strings.TrimSpace(a.regCustomCode)
	if _, _, isGeo := parseLatLng(custom); isGeo {
		return "附近定位"
	}
	if custom != "" {
		return custom
	}
	return regionShortNameByCode(a.regionalRegion)
}

// LoadExploreData 根据当前「探索」子分页加载现场实拍、eBird 近期/珍稀记录或观鸟热点。
func (a *AppState) LoadExploreData() {
	switch a.session.ExploreSubTab {
	case 0:
		a.loadExploreFieldPhotos()
	case 1, 2:
		a.loadExploreObservations(a.session.ExploreSubTab == 2)
	case 3:
		a.loadExploreHotspots()
	}
}

func (a *AppState) loadExploreFieldPhotos() {
	regionCode, lat, lng, isGeo := a.currentExploreRegionCode()
	if !isGeo {
		lat, lng = 39.9042, 116.4074
		if opt, ok := api.LookupRegionOption(regionCode); ok {
			lat, lng = opt.Lat, opt.Lng
		}
	}
	a.regPhotoBusy = true
	a.regPhotoErr = ""
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
		items, err := a.client.FetchRegionalPhotoObservations(ctx, lat, lng, 50, 30)
		cancel()

		a.updateUI(func() {
			a.regPhotoBusy = false
			a.exploreRefresh.Refreshing = false
			if err != nil {
				a.regPhotoErr = fmt.Sprintf("加载现场实拍记录失败: %v", err)
				a.regFieldPhotos = nil
				return
			}
			a.regFieldPhotos = items
		})
	}()
}

func (a *AppState) loadExploreObservations(notable bool) {
	if a.win == nil && len(a.regObsList) > 0 {
		return
	}
	regionCode, lat, lng, isGeo := a.currentExploreRegionCode()
	key := a.store.LoadEbirdKey()
	if key == "" {
		a.regObsErr = "查询 eBird 观测动态需要配置 API Key，请前往「设置」填入（或切换至免 Key 的“现场实拍”）"
		a.regObsList = nil
		a.exploreRefresh.Refreshing = false
		a.invalidateUI()
		return
	}

	backDays := 14
	switch a.regBackLabel {
	case "7天":
		backDays = 7
	case "30天":
		backDays = 30
	}

	a.regObsBusy = true
	a.regObsErr = ""
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
		var obs []model.Observation
		var err error
		if isGeo && !notable {
			obs, err = a.client.FetchNearbyObservations(ctx, key, lat, lng, 25, backDays, 100)
		} else {
			obs, err = a.client.FetchRecentObservations(ctx, key, regionCode, backDays, 100, notable)
		}
		stat, statErr := a.client.FetchRegionStats(ctx, key, regionCode, time.Now().AddDate(0, 0, -1))
		cancel()

		a.updateUI(func() {
			a.regObsBusy = false
			a.exploreRefresh.Refreshing = false
			if err != nil {
				a.regObsErr = fmt.Sprintf("加载观测记录失败: %v", err)
				a.regObsList = nil
				return
			}
			a.regObsList = obs
			if statErr == nil {
				a.regStat = stat
			} else {
				a.regStat = nil
			}
		})
	}()
}

func (a *AppState) loadExploreHotspots() {
	if a.win == nil && len(a.hsAll) > 0 {
		return
	}
	regionCode, _, _, _ := a.currentExploreRegionCode()
	key := a.store.LoadEbirdKey()
	if key == "" {
		a.hsErr = "查询观鸟热点需要配置 eBird API Key，请前往「设置」填写"
		a.hsAll = nil
		a.hsFiltered = nil
		a.exploreRefresh.Refreshing = false
		a.invalidateUI()
		return
	}

	a.hsBusy = true
	a.hsErr = ""
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 22*time.Second)
		list, err := a.client.FetchHotspots(ctx, key, regionCode, 30)
		cancel()

		a.updateUI(func() {
			a.hsBusy = false
			a.exploreRefresh.Refreshing = false
			if err != nil {
				a.hsErr = fmt.Sprintf("加载观鸟热点失败: %v", err)
				a.hsAll = nil
				a.hsFiltered = nil
				return
			}
			a.hsAll = list
			a.applyHotspotFilterAndSort()
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
}

func (a *AppState) OpenExploreChecklist(obs model.Observation) {
	copyObs := obs
	a.regSelectedObs = &copyObs
	a.regActiveSubID = obs.SubID
	a.regActiveObsID = obs.ObsID
	a.fetchChecklistDetailAsync(obs.SubID)
	a.router.Push("/explore/check/" + obs.SubID)
}

func (a *AppState) fetchChecklistDetailAsync(subID string) {
	if subID == "" {
		return
	}
	if _, ok := a.regChecklistMap[subID]; ok || a.regChecklistBusy[subID] {
		return
	}
	key := a.store.LoadEbirdKey()
	if key == "" {
		return
	}
	a.regChecklistBusy[subID] = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		detail, err := a.client.FetchChecklistDetail(ctx, key, subID)
		cancel()
		a.updateUI(func() {
			delete(a.regChecklistBusy, subID)
			if err == nil && detail != nil {
				a.regChecklistMap[subID] = detail
			}
		})
	}()
}

func (a *AppState) OpenExploreHotspot(h model.Hotspot) {
	copySpot := h
	a.hsActiveSpot = &copySpot
	if _, ok := a.hsObsMap[h.LocID]; !ok && !a.hsObsBusy[h.LocID] {
		if key := a.store.LoadEbirdKey(); key != "" {
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
	}
	a.router.Push("/explore/spot/" + h.LocID)
}

func (a *AppState) viewExploreHome(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Fill().Children(func() {
		// 顶部标题与区域选择胶囊
		ui.Column(c).FillWidth().Padding(12, 16, 10, 16).Gap(10).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "区域探索").FontSize(26).Bold().Role(ui.RoleHeading).Level(1)

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if a.regPhotoBusy || a.regObsBusy || a.hsBusy {
						ui.Spinner(c).Label("加载中")
					}
					locBtn := ui.Row(c.Key("exp-region-btn")).Height(34).PaddingX(12).Radius(17).
						Background(t.Surface).Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("切换探索地区")
					if !t.Dark {
						locBtn.Shadow(0, 1, 4, 0, ui.RGBA(15, 23, 42, 0.06))
					}
					locBtn.Children(func() {
						ui.Icon(c, iconMapPin).FontSize(13).TextColor(t.Accent)
						ui.Text(c, a.exploreRegionButtonText()+" ▾").FontSize(12.5).FontWeight(600).SingleLine()
					})
					if locBtn.Clicked() {
						a.regRegionTarget = "explore"
						a.regGPSNote = ""
						a.regRegionSheetOpen = true
					}
				})
			})

			if segmentedPillStrip(c, "exp-seg", &a.session.ExploreSubTab, t, "现场实拍", "近期目击", "珍稀记录", "观鸟热点") {
				a.LoadExploreData()
			}
		})

		// 主内容区（支持下拉刷新）
		scroll := ui.Scroll(c.Key(fmt.Sprintf("exp-scroll-%d", a.session.ExploreSubTab))).Grow(1).Basis(0).FillWidth()
		if scroll.PullToRefresh(&a.exploreRefresh) {
			a.LoadExploreData()
		}
		scroll.Children(func() {
			ui.Column(c).FillWidth().Padding(2, 16, 24, 16).Gap(12).Children(func() {
				switch a.session.ExploreSubTab {
				case 0:
					a.buildExplorePhotoFeed(c, t)
				case 1, 2:
					a.buildExploreEbirdObsFeed(c, t)
				case 3:
					a.buildExploreHotspotsFeed(c, t)
				}
			})
		})
	})
}

func (a *AppState) buildExplorePhotoFeed(c *ui.Context, t *ui.Theme) {
	if a.regPhotoErr != "" {
		elevatedCard(c, t).Padding(16).Gap(10).Children(func() {
			ui.Text(c, a.regPhotoErr).FontSize(13).TextColor(t.Danger)
			if ui.Button(c.Key("fp-retry"), "重新加载").Clicked() {
				a.loadExploreFieldPhotos()
			}
		})
		return
	}
	if len(a.regFieldPhotos) == 0 && !a.regPhotoBusy {
		elevatedCard(c, t).Padding(28).Center().Gap(8).Children(func() {
			ui.Icon(c, iconImage).FontSize(30).TextColor(t.TextMuted.Alpha(0.6))
			ui.Text(c, "周边暂无带图现场实拍").FontSize(15).FontWeight(600)
			ui.Text(c, "可下拉刷新或右上角切换其他省份/GPS定位").FontSize(12.5).TextColor(t.TextMuted)
			if ui.PrimaryButton(c.Key("fp-load-btn"), "立即获取周边实拍").Radius(18).Clicked() {
				a.loadExploreFieldPhotos()
			}
		})
		return
	}

	for i, item := range a.regFieldPhotos {
		obsItem := item
		card := elevatedCard(c.Key(fmt.Sprintf("fpcard-%d-%d", obsItem.ID, i)), t).Clip().
			Role(ui.RoleButton).Label(obsItem.ComName + " 现场实拍")
		card.Children(func() {
			// 当次观测者真实拍摄的现场照片（严禁与图鉴混用）
			ui.Box(c).FillWidth().Height(204).Background(softPillColor(t)).Children(func() {
				netimage.Image(c, obsItem.Photos[0].URL, netimage.Options{
					MaxSide:     900,
					Fit:         ui.Cover,
					Placeholder: softPillColor(t),
				}).Fill()

				ui.Row(c).Absolute().Right(10).Top(10).Gap(6).Children(func() {
					if len(obsItem.Sounds) > 0 {
						ui.Text(c, "含现场录音").FontSize(11).FontWeight(600).Padding(3, 9).Radius(999).
							Background(ui.RGBA(15, 23, 42, 0.62)).TextColor(ui.RGB(255, 255, 255))
					}
					if len(obsItem.Photos) > 1 {
						ui.Textf(c, "1/%d", len(obsItem.Photos)).FontSize(11).FontWeight(600).Padding(3, 9).Radius(999).
							Background(ui.RGBA(15, 23, 42, 0.62)).TextColor(ui.RGB(255, 255, 255))
					}
				})
			})

			ui.Column(c).FillWidth().Padding(14).Gap(6).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, obsItem.ComName).FontSize(17).FontWeight(600).SingleLine().Grow(1)
					statusPill(c, model.QualityGradeZh(obsItem.QualityGrade), t.Accent.Alpha(0.12), t.Accent)
				})
				ui.Text(c, obsItem.SciName).Italic().FontSize(12.5).TextColor(t.TextMuted).SingleLine()
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).Gap(8).Children(func() {
					ui.Text(c, obsItem.PlaceGuess).FontSize(12.5).TextColor(secondaryTextColor(t)).SingleLine().Grow(1)
					ui.Text(c, model.FormatObsTime(obsItem.ObservedOn)).FontSize(12).TextColor(t.TextMuted).SingleLine()
				})
				ui.Textf(c, "拍摄者：%s", obsItem.Observer).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
			})
		})
		if card.Clicked() {
			a.regActivePhoto = &obsItem
			a.regPhotoPager.Page = 0
			a.router.Push(fmt.Sprintf("/explore/photo/%d", obsItem.ID))
		}
	}
}

func (a *AppState) buildExploreEbirdObsFeed(c *ui.Context, t *ui.Theme) {
	// 昨日统计三列卡片
	if a.regStat != nil {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			metricMiniCard(c, t, "昨日清单", fmt.Sprintf("%d", a.regStat.NumChecklists))
			metricMiniCard(c, t, "目击鸟种", fmt.Sprintf("%d", a.regStat.NumSpecies))
			metricMiniCard(c, t, "贡献人数", fmt.Sprintf("%d", a.regStat.NumContributors))
		})
	}

	// 回溯天数切换
	ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			for _, label := range []string{"7天", "14天", "30天"} {
				l := label
				if filterPill(c, "back-"+l, l, a.regBackLabel == l, t).Clicked() {
					a.regBackLabel = l
					a.LoadExploreData()
				}
			}
		})
		if len(a.regObsList) > 0 {
			ui.Textf(c, "%d 条记录", len(a.regObsList)).FontSize(12).TextColor(t.TextMuted)
		}
	})

	if a.regObsErr != "" {
		elevatedCard(c, t).Padding(16).Gap(10).Children(func() {
			ui.Text(c, a.regObsErr).FontSize(13).TextColor(t.Danger)
			if a.store.LoadEbirdKey() == "" {
				if ui.PrimaryButton(c.Key("obs-goto-set"), "前往设置 API Key").Radius(18).Clicked() {
					a.SelectTab(3)
				}
			}
		})
		return
	}

	if len(a.regObsList) == 0 && !a.regObsBusy {
		elevatedCard(c, t).Padding(28).Center().Gap(8).Children(func() {
			ui.Icon(c, iconGlobe).FontSize(30).TextColor(t.TextMuted.Alpha(0.6))
			ui.Text(c, "近期暂无公开观测记录").FontSize(15).FontWeight(600)
			if ui.Button(c.Key("obs-load-now"), "刷新尝试").Clicked() {
				a.LoadExploreData()
			}
		})
		return
	}

	for i, o := range a.regObsList {
		obs := o
		card := elevatedRowCard(c.Key(fmt.Sprintf("ebobs-%s-%d", obs.SubID, i)), t).
			Padding(14).Gap(10).AlignItems(ui.Center).
			Role(ui.RoleButton).Label(obs.ComName)
		card.Children(func() {
			ui.Column(c).Grow(1).Basis(0).Gap(4).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, obs.ComName).FontSize(16).FontWeight(600).SingleLine()
					if obs.HowMany > 0 {
						statusPill(c, fmt.Sprintf("%d 只", obs.HowMany), softPillColor(t), secondaryTextColor(t))
					}
					if obs.HasRichMedia {
						statusPill(c, "含媒体", t.Accent.Alpha(0.12), t.Accent)
					}
				})
				ui.Text(c, fallbackText(obs.LocName, "未命名观鸟点")).FontSize(13).TextColor(secondaryTextColor(t)).SingleLine()
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, model.FormatObsTime(obs.ObsDt)).FontSize(12).TextColor(t.TextMuted)
					if obs.ObsValid {
						ui.Text(c, "已核验").FontSize(11.5).FontWeight(600).TextColor(t.Success)
					}
				})
			})
			ui.Icon(c, iconChevronRight).FontSize(16).TextColor(t.TextMuted.Alpha(0.5))
		})
		if card.Clicked() {
			a.OpenExploreChecklist(obs)
		}
	}
}

func (a *AppState) buildExploreHotspotsFeed(c *ui.Context, t *ui.Theme) {
	if ui.SearchField(c.Key("hs-search"), &a.hsQuery).
		Placeholder("搜索热点名称或代码…").Label("搜索热点").
		FillWidth().Height(40).Radius(12).Changed() {
		a.applyHotspotFilterAndSort()
	}

	ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			if filterPill(c, "hsort-cnt", "鸟种最多", a.hsSortMode == 0, t).Clicked() {
				a.hsSortMode = 0
				a.applyHotspotFilterAndSort()
			}
			if filterPill(c, "hsort-dt", "最近活跃", a.hsSortMode == 1, t).Clicked() {
				a.hsSortMode = 1
				a.applyHotspotFilterAndSort()
			}
		})
		if len(a.hsFiltered) > 0 {
			ui.Textf(c, "%d 个热点", len(a.hsFiltered)).FontSize(12).TextColor(t.TextMuted)
		}
	})

	if a.hsErr != "" {
		elevatedCard(c, t).Padding(16).Gap(10).Children(func() {
			ui.Text(c, a.hsErr).FontSize(13).TextColor(t.Danger)
			if a.store.LoadEbirdKey() == "" {
				if ui.PrimaryButton(c.Key("hs-goto-set"), "前往设置 API Key").Radius(18).Clicked() {
					a.SelectTab(3)
				}
			}
		})
		return
	}

	if len(a.hsFiltered) == 0 && !a.hsBusy {
		elevatedCard(c, t).Padding(28).Center().Gap(8).Children(func() {
			ui.Icon(c, iconMapPin).FontSize(30).TextColor(t.TextMuted.Alpha(0.6))
			ui.Text(c, "暂无符合条件的观鸟热点").FontSize(15).FontWeight(600)
			if ui.Button(c.Key("hs-load-now"), "刷新尝试").Clicked() {
				a.loadExploreHotspots()
			}
		})
		return
	}

	for _, h := range a.hsFiltered {
		spot := h
		card := elevatedRowCard(c.Key("hspot-"+spot.LocID), t).
			Padding(12, 14).Gap(12).AlignItems(ui.Center).
			Role(ui.RoleButton).Label(spot.LocName)
		card.Children(func() {
			ui.Column(c).Size(54, 54).Radius(14).Background(t.Accent.Alpha(0.12)).Center().Shrink(0).Children(func() {
				ui.Textf(c, "%d", spot.NumSpeciesAllTime).FontSize(17).Bold().TextColor(t.Accent).FontFeatures("tnum")
				ui.Text(c, "种").FontSize(11).TextColor(t.Accent)
			})
			ui.Column(c).Grow(1).Basis(0).Gap(3).Children(func() {
				ui.Text(c, spot.LocName).FontSize(15.5).FontWeight(600).SingleLine()
				ui.Textf(c, "%s · %s", spot.LocID, model.FormatObsTime(spot.LatestObsDt)).
					FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
			ui.Icon(c, iconChevronRight).FontSize(16).TextColor(t.TextMuted.Alpha(0.5))
		})
		if card.Clicked() {
			a.OpenExploreHotspot(spot)
		}
	}
}

func (a *AppState) viewExplorePhotoDetail(c *ui.Context, t *ui.Theme, _ string) {
	item := a.regActivePhoto
	if item == nil {
		ui.Column(c).Fill().Center().Padding(24).Gap(12).Children(func() {
			ui.Text(c, "未找到该现场实拍记录")
			if ui.Button(c.Key("ep-back"), "返回探索").Clicked() {
				a.goBack()
			}
		})
		return
	}

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Padding(10, 16, 8, 16).Gap(10).AlignItems(ui.Center).Children(func() {
			if circleIconBtn(c, "ep-back-btn", "返回", iconBack, t.Surface, t.Text, 38).Clicked() {
				a.goBack()
			}
			ui.Text(c, item.ComName+" · 现场实拍").FontSize(16.5).FontWeight(600).SingleLine().Grow(1)
		})

		ui.Scroll(c.Key("ep-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(0, 0, 28, 0).Gap(14).Children(func() {
				count := len(item.Photos)
				ui.Box(c).FillWidth().Height(272).Background(softPillColor(t)).Children(func() {
					mobile.PageView(c, &a.regPhotoPager, count, func(i int) {
						slide := ui.Box(c.Key(fmt.Sprintf("fpslide-%d", i))).Fill().
							Role(ui.RoleButton).Label("点击放大查看实拍图")
						slide.Children(func() {
							netimage.Image(c, item.Photos[i].URL, netimage.Options{
								MaxSide:     1200,
								Fit:         ui.Cover,
								Placeholder: softPillColor(t),
							}).Fill()
						})
						if slide.Clicked() {
							a.openNetworkPhotosViewer(item.ComName+"（现场实拍）", item.Photos, i)
						}
					}).Fill()
					ui.Textf(c, "%d / %d", a.regPhotoPager.Page+1, count).
						Absolute().Right(12).Bottom(12).Padding(4, 10).Radius(999).
						FontSize(11.5).FontWeight(600).
						Background(ui.RGBA(15, 23, 42, 0.58)).TextColor(ui.RGB(255, 255, 255))
				})

				if count > 1 {
					mobile.PageDots(c, a.regPhotoPager.Page, count)
				}

				ui.Box(c).PaddingX(16).Children(func() {
					elevatedCard(c, t).Padding(16).Gap(8).Children(func() {
						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Text(c, item.ComName).FontSize(22).Bold()
							statusPill(c, model.QualityGradeZh(item.QualityGrade), t.Accent.Alpha(0.12), t.Accent)
						})
						ui.Text(c, item.SciName).Italic().FontSize(14).TextColor(t.TextMuted)
						ui.Row(c).Gap(8).Wrap().Children(func() {
							statusPill(c, model.FormatObsTime(item.ObservedOn), softPillColor(t), secondaryTextColor(t))
							statusPill(c, item.PlaceGuess, softPillColor(t), secondaryTextColor(t))
							statusPill(c, "拍摄："+item.Observer, softPillColor(t), t.TextMuted)
						})
						if item.ObsURL != "" {
							ui.Box(c).Padding(4, 0, 0, 0).Children(func() {
								ui.Link(c, "打开 iNaturalist 现场原记录 ↗", item.ObsURL)
							})
						}
					})
				})

				if len(item.Sounds) > 0 {
					ui.Column(c).PaddingX(16).Gap(8).Children(func() {
						ui.Text(c, "本次目击现场录音").FontSize(16).FontWeight(600)
						st := a.player.Status()
						for i, snd := range item.Sounds {
							snd := snd
							isThis := st.SourceKey == snd.URL
							elevatedRowCard(c.Key(fmt.Sprintf("fps-%d", i)), t).Padding(12).Gap(12).AlignItems(ui.Center).Children(func() {
								ic := iconPlay
								if isThis && st.Playing {
									ic = iconPause
								}
								if circleIconBtn(c, fmt.Sprintf("fpsb-%d", i), "播放现场录音", ic, t.Accent, t.AccentText, 40).Clicked() {
									if isThis && (st.Playing || st.Loading) {
										a.player.TogglePause()
									} else {
										a.player.PlayURL(snd.URL, item.ComName+" · 现场录音", item.PlaceGuess, a.cache, a.client, func() {
											a.updateUI(func() {})
										})
									}
								}
								ui.Column(c).Grow(1).Gap(2).Children(func() {
									ui.Text(c, snd.Type).FontSize(14).FontWeight(600)
									ui.Text(c, "录音者："+snd.Recordist).FontSize(12).TextColor(t.TextMuted)
								})
							})
						}
					})
				}
			})
		})
	})
}

func (a *AppState) viewExploreChecklistDetail(c *ui.Context, t *ui.Theme, subID string) {
	obs := a.regSelectedObs
	if obs == nil || obs.SubID != subID {
		for i := range a.regObsList {
			if a.regObsList[i].SubID == subID {
				copyO := a.regObsList[i]
				obs = &copyO
				break
			}
		}
	}
	if obs == nil {
		ui.Column(c).Fill().Center().Padding(24).Gap(12).Children(func() {
			ui.Text(c, "未找到该条观测清单")
			if ui.Button(c.Key("ec-back"), "返回探索").Clicked() {
				a.goBack()
			}
		})
		return
	}

	checklist := a.regChecklistMap[obs.SubID]
	clBusy := a.regChecklistBusy[obs.SubID]
	var matchedItem *model.ChecklistObs
	if checklist != nil {
		for i := range checklist.Obs {
			item := &checklist.Obs[i]
			if (obs.ObsID != "" && item.ObsID == obs.ObsID) || item.SpeciesCode == obs.SpeciesCode {
				matchedItem = item
				break
			}
		}
	}

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Padding(10, 16, 8, 16).Gap(10).AlignItems(ui.Center).Children(func() {
			if circleIconBtn(c, "ec-back-btn", "返回", iconBack, t.Surface, t.Text, 38).Clicked() {
				a.goBack()
			}
			ui.Text(c, obs.ComName+" · 清单详情").FontSize(16.5).FontWeight(600).SingleLine().Grow(1)
			if filterPill(c, "ec-goto-guide", "看图鉴", false, t).Clicked() {
				a.OpenSpeciesInGuide(obs.SpeciesCode)
			}
		})

		ui.Scroll(c.Key("ec-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(8, 16, 28, 16).Gap(14).Children(func() {
				elevatedCard(c, t).Padding(16).Gap(8).Children(func() {
					ui.Text(c, obs.ComName).FontSize(22).Bold()
					ui.Text(c, obs.SciName).Italic().FontSize(14).TextColor(t.TextMuted)
					ui.Row(c).Gap(8).Wrap().Children(func() {
						statusPill(c, model.FormatObsTime(obs.ObsDt), softPillColor(t), secondaryTextColor(t))
						statusPill(c, fallbackText(obs.LocName, "未命名地点"), softPillColor(t), secondaryTextColor(t))
						if obs.HowMany > 0 {
							statusPill(c, fmt.Sprintf("%d 只", obs.HowMany), softPillColor(t), secondaryTextColor(t))
						}
						if obs.UserDisplayName != "" {
							statusPill(c, obs.UserDisplayName, softPillColor(t), t.TextMuted)
						}
					})
				})

				// 严格隔离卡片：绝不在此处加载图鉴通用物种照
				elevatedCard(c, t).Padding(16).Gap(10).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "本次观测现场媒体（非图鉴标准图）").FontSize(15).FontWeight(600)
						if clBusy {
							ui.Spinner(c).Label("获取清单")
						}
					})

					photoCnt, audioCnt := 0, 0
					if matchedItem != nil && matchedItem.MediaCounts != nil {
						photoCnt = matchedItem.MediaCounts["P"]
						audioCnt = matchedItem.MediaCounts["A"]
					}

					if photoCnt > 0 || audioCnt > 0 || obs.HasRichMedia {
						msg := fmt.Sprintf("观测者已为清单 %s 上传现场媒体：%d 张照片 · %d 段录音。", obs.SubID, photoCnt, audioCnt)
						ui.Text(c, msg).FontSize(13.5)
						ui.Text(c, "eBird API v2 仅返回现场媒体计数而不直接暴露图片地址，请点击下方按钮打开原清单查看本次实拍图（不与图鉴通用图混用）。").
							FontSize(12.5).TextColor(t.TextMuted).LineHeight(1.5)
						if obs.SubID != "" {
							if ui.PrimaryButton(c.Key("ec-open-ebird"), "在浏览器打开 eBird 清单原页 ↗").FillWidth().Height(44).Radius(22).Clicked() {
								_ = mygo.Shell.OpenExternal("https://ebird.org/checklist/" + obs.SubID)
							}
						}
					} else {
						ui.Text(c, "本次目击仅提交了物种观测计数，未附带现场照片或录音。点击右上方「看图鉴」可浏览该鸟种标准生态资料。").
							FontSize(13).TextColor(t.TextMuted).LineHeight(1.5)
					}
				})

				elevatedCard(c, t).Padding(16).Gap(10).Selectable().Children(func() {
					ui.Text(c, "观鸟清单上下文").FontSize(15).FontWeight(600)
					if checklist == nil {
						ui.Text(c, "暂无该清单详细信息。").FontSize(13).TextColor(t.TextMuted)
					} else {
						ui.Row(c).Gap(8).Wrap().Children(func() {
							if checklist.UserDisplayName != "" {
								statusPill(c, "作者："+checklist.UserDisplayName, softPillColor(t), secondaryTextColor(t))
							}
							if checklist.DurationHrs > 0 {
								statusPill(c, fmt.Sprintf("%.1f 小时", checklist.DurationHrs), softPillColor(t), secondaryTextColor(t))
							}
							if checklist.EffortDistanceKm > 0 {
								statusPill(c, fmt.Sprintf("%.2f km", checklist.EffortDistanceKm), softPillColor(t), secondaryTextColor(t))
							}
							statusPill(c, fmt.Sprintf("共记录 %d 种", checklist.NumSpecies), t.Accent.Alpha(0.12), t.Accent)
						})

						if matchedItem != nil && strings.TrimSpace(matchedItem.Comments) != "" {
							ui.Column(c).FillWidth().Padding(12).Radius(12).Background(softPillColor(t)).Gap(4).Children(func() {
								ui.Text(c, "现场记录备注").FontSize(12).FontWeight(600).TextColor(t.TextMuted)
								ui.Text(c, matchedItem.Comments).FontSize(13.5).LineHeight(1.5)
							})
						}

						if len(checklist.Obs) > 0 {
							ui.Text(c, "同清单伴生鸟种").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
							limit := min(20, len(checklist.Obs))
							for i := 0; i < limit; i++ {
								co := checklist.Obs[i]
								name := co.SpeciesCode
								if tx, ok := a.taxonByCode[co.SpeciesCode]; ok {
									name = tx.ComName
								}
								ui.Row(c.Key(fmt.Sprintf("ec-co-%d", i))).FillWidth().Justify(ui.SpaceBetween).Padding(6, 10).Radius(10).Background(softPillColor(t)).Children(func() {
									ui.Text(c, name).FontSize(13)
									ui.Text(c, fallbackText(co.HowManyStr, "在场")).FontSize(12.5).TextColor(t.TextMuted)
								})
							}
						}
					}
				})

				if ui.PrimaryButton(c.Key("ec-add-journal"), "记入观鸟日志").FillWidth().Height(46).Radius(23).Clicked() {
					a.OpenNewJournalModal(obs.SpeciesCode, obs.LocName)
				}
			})
		})
	})
}

func (a *AppState) viewExploreHotspotDetail(c *ui.Context, t *ui.Theme, locID string) {
	spot := a.hsActiveSpot
	if spot == nil || spot.LocID != locID {
		for i := range a.hsFiltered {
			if a.hsFiltered[i].LocID == locID {
				copyS := a.hsFiltered[i]
				spot = &copyS
				break
			}
		}
	}
	if spot == nil {
		ui.Column(c).Fill().Center().Padding(24).Gap(12).Children(func() {
			ui.Text(c, "未找到该观鸟热点")
			if ui.Button(c.Key("es-back"), "返回热点").Clicked() {
				a.goBack()
			}
		})
		return
	}

	obsList := a.hsObsMap[spot.LocID]
	busy := a.hsObsBusy[spot.LocID]

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Padding(10, 16, 8, 16).Gap(10).AlignItems(ui.Center).Children(func() {
			if circleIconBtn(c, "es-back-btn", "返回", iconBack, t.Surface, t.Text, 38).Clicked() {
				a.goBack()
			}
			ui.Text(c, spot.LocName).FontSize(16.5).FontWeight(600).SingleLine().Grow(1)
		})

		ui.Scroll(c.Key("es-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(8, 16, 28, 16).Gap(14).Children(func() {
				elevatedCard(c, t).Padding(16).Gap(10).Children(func() {
					ui.Text(c, spot.LocName).FontSize(20).Bold()
					ui.Textf(c, "热点编号 %s · 坐标 %.4f, %.4f", spot.LocID, spot.Lat, spot.Lng).
						FontSize(12.5).TextColor(t.TextMuted)
					ui.Row(c).Gap(8).Wrap().Children(func() {
						statusPill(c, fmt.Sprintf("累计 %d 种", spot.NumSpeciesAllTime), t.Accent.Alpha(0.12), t.Accent)
						statusPill(c, "最近活跃 "+model.FormatObsTime(spot.LatestObsDt), softPillColor(t), secondaryTextColor(t))
					})

					ui.Row(c).FillWidth().Gap(10).Padding(4, 0, 0, 0).Children(func() {
						if ui.PrimaryButton(c.Key("hs-new-log"), "在此处写日志").Grow(1).Height(42).Radius(21).Clicked() {
							a.OpenNewJournalModal("", spot.LocName)
						}
						if ui.Button(c.Key("hs-open-web"), "网页查看 ↗").Height(42).Radius(21).Clicked() {
							_ = mygo.Shell.OpenExternal("https://ebird.org/hotspot/" + spot.LocID)
						}
					})
				})

				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "近 30 天目击鸟种").FontSize(16).FontWeight(600)
						if busy {
							ui.Spinner(c).Label("加载鸟种")
						} else {
							ui.Textf(c, "%d 条", len(obsList)).FontSize(12.5).TextColor(t.TextMuted)
						}
					})

					if len(obsList) == 0 && !busy {
						elevatedCard(c, t).Padding(16).Children(func() {
							ui.Text(c, "该热点近 30 天暂无公开目击记录。").FontSize(13).TextColor(t.TextMuted)
						})
					} else {
						for i, o := range obsList {
							o := o
							elevatedRowCard(c.Key(fmt.Sprintf("esobs-%d", i)), t).Padding(12, 14).Gap(10).AlignItems(ui.Center).Children(func() {
								ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
									ui.Text(c, o.ComName).FontSize(15).FontWeight(600).SingleLine()
									cnt := "在场"
									if o.HowMany > 0 {
										cnt = fmt.Sprintf("%d 只", o.HowMany)
									}
									ui.Textf(c, "%s · %s", model.FormatObsTime(o.ObsDt), cnt).FontSize(12).TextColor(t.TextMuted)
								})
								if filterPill(c, fmt.Sprintf("eslog-%d", i), "记日志", false, t).Clicked() {
									a.OpenNewJournalModal(o.SpeciesCode, spot.LocName)
								}
								if filterPill(c, fmt.Sprintf("essp-%d", i), "图鉴", true, t).Clicked() {
									a.OpenSpeciesInGuide(o.SpeciesCode)
								}
							})
						}
					}
				})
			})
		})
	})
}

func (a *AppState) viewRegionPickerSheet(c *ui.Context, t *ui.Theme) {
	mobile.Sheet(c, &a.regRegionSheetOpen, mobile.SheetOptions{Detents: []float32{0.72, 0.92}, Label: "选择探索地区"}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 16, 12, 16).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "选择观鸟地区").FontSize(17).FontWeight(600)
				if ui.Button(c.Key("rp-close"), "关闭").Clicked() {
					a.regRegionSheetOpen = false
				}
			})

			ui.Column(c).Padding(0, 16, 10, 16).Gap(10).Children(func() {
				if a.regRegionTarget == "explore" {
					gpsBtn := ui.Row(c.Key("rp-gps")).FillWidth().Height(46).PaddingX(14).Radius(14).
						Background(t.Accent.Alpha(0.12)).Gap(10).AlignItems(ui.Center).
						Role(ui.RoleButton).Label("使用手机 GPS 定位周边")
					gpsBtn.Children(func() {
						if a.regGPSBusy {
							ui.Spinner(c).Label("定位中")
						} else {
							ui.Icon(c, iconLocate).FontSize(18).TextColor(t.Accent)
						}
						ui.Text(c, "使用手机 GPS 定位周边").FontSize(14.5).FontWeight(600).TextColor(t.Accent).Grow(1)
					})
					if gpsBtn.Clicked() {
						a.fillGPSLocationAsync("explore")
					}
					if a.regGPSNote != "" {
						ui.Text(c, a.regGPSNote).FontSize(12).TextColor(t.Warning)
					}

					ui.Row(c).FillWidth().Gap(8).Children(func() {
						in := ui.TextInput(c.Key("rp-custom"), &a.regCustomCode).
							Placeholder("自定义地区码或纬度,经度（如 US-NY / 39.9,116.4）").
							Label("自定义地区").Grow(1).Height(40)
						if in.Submitted() || ui.PrimaryButton(c.Key("rp-go"), "查询").Height(40).Radius(12).Clicked() {
							a.regRegionSheetOpen = false
							a.LoadExploreData()
						}
					})
				}

				ui.SearchField(c.Key("rp-search"), &a.regRegionQuery).
					Placeholder("搜索省市名称或区号…").Label("搜索地区").
					FillWidth().Height(38).Radius(10)
			})

			ui.Scroll(c.Key("rp-list")).Grow(1).Basis(0).FillWidth().Padding(0, 16, 24, 16).Children(func() {
				q := strings.ToLower(strings.TrimSpace(a.regRegionQuery))
				ui.Column(c).FillWidth().Background(t.Surface).Radius(14).Dividers(0.5, subtleHairlineColor(t)).Children(func() {
					for _, reg := range api.ChinaRegions {
						r := reg
						if q != "" && !strings.Contains(strings.ToLower(r.Name), q) && !strings.Contains(strings.ToLower(r.Code), q) {
							continue
						}
						active := false
						if a.regRegionTarget == "settings" {
							active = a.setRegionLabel == r.Name
						} else {
							active = a.regCustomCode == "" && a.regRegionLabel == r.Name
						}
						row := mobile.ListRow(c.Key("regopt-"+r.Code), mobile.RowOptions{
							Title: r.Name,
							Trailing: func() {
								if active {
									ui.Icon(c, iconCheck).FontSize(16).TextColor(t.Accent)
								}
							},
						})
						if row.Clicked() {
							a.regRegionSheetOpen = false
							if a.regRegionTarget == "settings" {
								a.setRegionLabel = r.Name
								a.saveRegionPreferenceOnly(c, r.Code)
							} else {
								a.regCustomCode = ""
								a.regRegionLabel = r.Name
								a.LoadExploreData()
							}
						}
					}
				})
			})
		})
	})
}

func metricMiniCard(c *ui.Context, t *ui.Theme, label, val string) {
	ui.Column(c).Grow(1).Basis(0).Padding(12, 10).Radius(14).Background(t.Surface).AlignItems(ui.Center).Gap(2).Children(func() {
		ui.Text(c, val).FontSize(18).Bold().FontFeatures("tnum")
		ui.Text(c, label).FontSize(11.5).TextColor(t.TextMuted)
	})
}
