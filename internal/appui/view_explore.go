package appui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// resetExploreForRegionChange 切换探索地区时立刻清空旧地区缓存与滚动位置，并使在途旧请求失效，确保界面马上进入灰色骨架卡片加载。
func (a *AppState) resetExploreForRegionChange() {
	a.regPhotoSeq++
	a.regObsSeq++
	a.hsSeq++
	a.regPhotoBusy = false
	a.regObsBusy = false
	a.hsBusy = false
	a.regPhotoErr = ""
	a.regObsErr = ""
	a.hsErr = ""
	a.regFieldPhotos = nil
	a.regObsList = nil
	a.regStat = nil
	a.hsAll = nil
	a.hsFiltered = nil
	a.exploreScroll = [4]ui.ScrollState{}
	a.LoadExploreData()
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
	if a.win == nil && len(a.regFieldPhotos) > 0 {
		return
	}
	regionCode, lat, lng, isGeo := a.currentExploreRegionCode()
	if !isGeo {
		lat, lng = 39.9042, 116.4074
		if opt, ok := api.LookupRegionOption(regionCode); ok {
			lat, lng = opt.Lat, opt.Lng
		}
	}
	a.regPhotoSeq++
	seq := a.regPhotoSeq
	a.regFieldPhotos = nil
	a.regPhotoBusy = true
	a.regPhotoErr = ""
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
		items, err := a.client.FetchRegionalPhotoObservations(ctx, lat, lng, 50, 30)
		cancel()

		a.updateUI(func() {
			if seq != a.regPhotoSeq {
				return
			}
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
		a.regObsSeq++
		a.regObsBusy = false
		a.regObsErr = "查询 eBird 观测动态需要配置 API Key，请前往「设置」填入（或切换至免 Key 的“现场实拍”）"
		a.regObsList = nil
		a.regStat = nil
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

	a.regObsSeq++
	seq := a.regObsSeq
	a.regObsList = nil
	a.regStat = nil
	a.regObsBusy = true
	a.regObsErr = ""
	a.invalidateUI()

	go func() {
		var (
			wg      sync.WaitGroup
			obs     []model.Observation
			err     error
			stat    *model.RegionStat
			statErr error
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
			if isGeo && !notable {
				obs, err = a.client.FetchNearbyObservations(ctx, key, lat, lng, 25, backDays, 100)
			} else {
				obs, err = a.client.FetchRecentObservations(ctx, key, regionCode, backDays, 100, notable)
			}
			cancel()
		}()
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			stat, statErr = a.client.FetchRegionStats(ctx, key, regionCode, time.Now().AddDate(0, 0, -1))
			cancel()
		}()
		wg.Wait()

		a.updateUI(func() {
			if seq != a.regObsSeq {
				return
			}
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
		a.hsSeq++
		a.hsBusy = false
		a.hsErr = "查询观鸟热点需要配置 eBird API Key，请前往「设置」填写"
		a.hsAll = nil
		a.hsFiltered = nil
		a.exploreRefresh.Refreshing = false
		a.invalidateUI()
		return
	}

	a.hsSeq++
	seq := a.hsSeq
	a.hsAll = nil
	a.hsFiltered = nil
	a.hsBusy = true
	a.hsErr = ""
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
		list, err := a.client.FetchHotspots(ctx, key, regionCode, 30)
		cancel()

		a.updateUI(func() {
			if seq != a.hsSeq {
				return
			}
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
		ctx, cancel := context.WithTimeout(context.Background(), 22*time.Second)
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
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
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
		ui.Column(c).FillWidth().Padding(immersiveTopPad(c, 16), 18, 10, 18).Gap(12).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "探索").FontSize(26).FontWeight(600).LetterSpacing(-0.1).Role(ui.RoleHeading).Level(1)

				locBtn := applySculptedSecondaryPillStyle(
					ui.Row(c.Key("exp-region-btn")).Height(34).PaddingX(14).Radius(17).
						Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("切换探索地区"),
					t,
				)
				locBtn.Children(func() {
					ui.Icon(c, iconMapPin).FontSize(13).TextColor(opticalAccentColor(t))
					ui.Text(c, a.exploreRegionButtonText()).FontSize(13).FontWeight(500).SingleLine()
				})
				if locBtn.Clicked() {
					a.regRegionTarget = "explore"
					a.regGPSNote = ""
					a.regRegionSheetOpen = true
				}
			})

			if tactileSegmentedControl(c, "exp-seg", &a.session.ExploreSubTab, t, "现场实拍", "近期目击", "珍稀记录", "观鸟热点") {
				a.exploreScroll[a.session.ExploreSubTab].Y = 0
				a.LoadExploreData()
			}
		})

		ui.Box(c).Grow(1).Basis(0).FillWidth().Children(func() {
			scroll := ui.Scroll(c.Key(fmt.Sprintf("exp-scroll-%d", a.session.ExploreSubTab))).Fill().
				Scrollbars(false).TrackScroll(&a.exploreScroll[a.session.ExploreSubTab])
			if scroll.PullToRefresh(&a.exploreRefresh) {
				a.LoadExploreData()
			}
			scroll.Children(func() {
				ui.Column(c).FillWidth().Padding(12, 18, dockClearancePad(c, 22), 18).Gap(14).Children(func() {
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
			if a.session.ExploreSubTab == 0 && len(a.regFieldPhotos) > 0 {
				scrollTopFadeScrim(c, t, a.exploreScroll[0].Y)
			}
		})
	})
}

func (a *AppState) buildExplorePhotoFeed(c *ui.Context, t *ui.Theme) {
	if a.regPhotoErr != "" {
		tactileCard(c, t).Padding(18).Gap(12).Children(func() {
			ui.Text(c, a.regPhotoErr).FontSize(13).TextColor(t.Danger)
			if filterPill(c, "fp-retry", "重新加载", false, t).Clicked() {
				a.loadExploreFieldPhotos()
			}
		})
		return
	}
	if a.regPhotoBusy {
		ui.Column(c).FillWidth().Gap(16).Children(func() {
			for range 2 {
				tactileCard(c, t).Clip().Children(func() {
					ui.Box(c).FillWidth().Height(228).Children(func() {
						skeletonBirdCard(c, 0, 42, t)
					})
					ui.Column(c).FillWidth().Padding(14, 16).Gap(8).Children(func() {
						skeletonShimmerBox(c, 6, t).WidthPercent(46).Height(16)
						skeletonShimmerBox(c, 5, t).WidthPercent(72).Height(12)
					})
				})
			}
		})
		return
	}
	if len(a.regFieldPhotos) == 0 && !a.regPhotoBusy {
		tactileCard(c, t).Padding(32).Center().Gap(10).Children(func() {
			tactileRecessedStyle(ui.Box(c).Size(52, 52), t, 26).Center().Children(func() {
				ui.Icon(c, iconImage).FontSize(22).TextColor(t.TextMuted)
			})
			ui.Text(c, "周边暂无带图现场实拍").FontSize(15).FontWeight(600)
			ui.Text(c, "可下拉刷新或右上角切换其他省份/GPS定位").FontSize(12).TextColor(t.TextMuted)
			if filterPill(c, "fp-load-btn", "立即获取", true, t).Clicked() {
				a.loadExploreFieldPhotos()
			}
		})
		return
	}

	ui.Column(c).FillWidth().Gap(16).Children(func() {
		for i, item := range a.regFieldPhotos {
			obsItem := item
			card := tactileCard(c, t).Key(fmt.Sprintf("fpcard-%d-%d", obsItem.ID, i)).Clip().
				Role(ui.RoleButton).Label(obsItem.ComName + " 现场实拍")
			card.Children(func() {
				ui.Box(c).FillWidth().Height(228).Background(softPillColor(t)).Children(func() {
					photoSrc, _ := netimage.Load(c, obsItem.Photos[0].URL, netimage.Options{MaxSide: 960})
					if photoSrc != nil {
						ui.Image(c, photoSrc).Fill().Fit(ui.Cover).Role(ui.RoleImage)
						ui.Row(c).Absolute().Right(12).Top(12).Gap(6).Children(func() {
							if len(obsItem.Sounds) > 0 {
								ui.Text(c, "录音").FontSize(11).FontWeight(500).Padding(4, 10).Radius(999).
									Material(photoOverlayGlass(false)).
									TextColor(ui.RGB(255, 255, 255))
							}
							if len(obsItem.Photos) > 1 {
								ui.Textf(c, "1/%d", len(obsItem.Photos)).FontSize(11).FontWeight(500).Padding(4, 10).Radius(999).
									Material(photoOverlayGlass(false)).
									FontFeatures("tnum").
									TextColor(ui.RGB(255, 255, 255))
							}
						})
					} else {
						skeletonBirdCard(c, 0, 42, t)
					}
				})

				ui.Column(c).FillWidth().Padding(14, 16).Gap(4).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Baseline).Gap(8).Children(func() {
						ui.Text(c, obsItem.ComName).FontSize(16).FontWeight(600).SingleLine().Grow(1)
						ui.Text(c, model.FormatObsTime(obsItem.ObservedOn)).FontSize(12).TextColor(t.TextMuted).SingleLine()
					})
					ui.Textf(c, "%s · 拍摄者：%s", fallbackText(obsItem.PlaceGuess, "野外现场"), obsItem.Observer).
						FontSize(12).TextColor(secondaryTextColor(t)).SingleLine()
				})
			})
			if card.Clicked() {
				a.regActivePhoto = &obsItem
				a.regPhotoPager = mobile.PageState{}
				a.router.Push(fmt.Sprintf("/explore/photo/%d", obsItem.ID))
			}
		}
	})
}

func (a *AppState) buildExploreEbirdObsFeed(c *ui.Context, t *ui.Theme) {
	if a.regObsBusy {
		skeletonObsStatsCard(c, t)
	} else if a.regStat != nil {
		tactileCard(c, t).Padding(14, 8).Children(func() {
			ui.Row(c).FillWidth().Dividers(1, subtleHairlineColor(t)).Children(func() {
				metricMiniCard(c, t, "昨日清单", fmt.Sprintf("%d", a.regStat.NumChecklists))
				metricMiniCard(c, t, "目击鸟种", fmt.Sprintf("%d", a.regStat.NumSpecies))
				metricMiniCard(c, t, "贡献人数", fmt.Sprintf("%d", a.regStat.NumContributors))
			})
		})
	}

	ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		backOpts := []string{"7天", "14天", "30天"}
		backIdx := 1
		for i, l := range backOpts {
			if a.regBackLabel == l {
				backIdx = i
				break
			}
		}
		if tactileCompactSegment(c, "obs-back-seg", &backIdx, t, backOpts...) {
			a.regBackLabel = backOpts[backIdx]
			a.LoadExploreData()
		}
		if !a.regObsBusy && len(a.regObsList) > 0 {
			ui.Textf(c, "%d 条记录", len(a.regObsList)).FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
		}
	})

	if a.regObsErr != "" {
		tactileCard(c, t).Padding(18).Gap(12).Children(func() {
			ui.Text(c, a.regObsErr).FontSize(13).TextColor(t.Danger)
			if a.store.LoadEbirdKey() == "" {
				if filterPill(c, "obs-goto-set", "前往设置 Key", true, t).Clicked() {
					a.SelectTab(3)
				}
			}
		})
		return
	}

	if a.regObsBusy {
		skeletonObsListCard(c, 6, t)
		return
	}

	if len(a.regObsList) == 0 {
		tactileCard(c, t).Padding(32).Center().Gap(10).Children(func() {
			tactileRecessedStyle(ui.Box(c).Size(52, 52), t, 26).Center().Children(func() {
				ui.Icon(c, iconGlobe).FontSize(22).TextColor(t.TextMuted)
			})
			ui.Text(c, "近期暂无公开观测记录").FontSize(15).FontWeight(600)
			if filterPill(c, "obs-load-now", "刷新尝试", false, t).Clicked() {
				a.LoadExploreData()
			}
		})
		return
	}

	tactileCard(c, t).Clip().Dividers(1, subtleHairlineColor(t)).Children(func() {
		for i, o := range a.regObsList {
			obs := o
			row := ui.Row(c.Key(fmt.Sprintf("ebobs-%s-%d", obs.SubID, i))).
				FillWidth().Padding(13, 16).Gap(12).AlignItems(ui.Center).
				Role(ui.RoleButton).Label(obs.ComName)
			row.Children(func() {
				ui.Column(c).Grow(1).Basis(0).Gap(3).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, obs.ComName).FontSize(15).FontWeight(500).SingleLine()
						if obs.HasRichMedia {
							statusPill(c, "含媒体", softPillColor(t), opticalAccentColor(t))
						}
					})
					ui.Text(c, fallbackText(obs.LocName, "未命名观鸟点")).FontSize(12).TextColor(t.TextMuted).SingleLine()
				})

				ui.Column(c).AlignItems(ui.End).Gap(3).Shrink(0).Children(func() {
					cntText := "在场"
					if obs.HowMany > 0 {
						cntText = fmt.Sprintf("%d 只", obs.HowMany)
					}
					ui.Text(c, cntText).FontSize(14).FontWeight(500).FontFeatures("tnum")
					ui.Text(c, model.FormatObsTime(obs.ObsDt)).FontSize(12).TextColor(t.TextMuted)
				})
			})
			if row.Clicked() {
				a.OpenExploreChecklist(obs)
			}
		}
	})
}

func (a *AppState) buildExploreHotspotsFeed(c *ui.Context, t *ui.Theme) {
	if minimalSearchBar(c, "hs-search", &a.hsQuery, "搜索热点名称", "搜索热点", t) {
		a.applyHotspotFilterAndSort()
	}

	ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		if tactileCompactSegment(c, "hs-sort-seg", &a.hsSortMode, t, "鸟种最多", "最近活跃") {
			a.applyHotspotFilterAndSort()
		}
		if !a.hsBusy && len(a.hsFiltered) > 0 {
			ui.Textf(c, "%d 个热点", len(a.hsFiltered)).FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
		}
	})

	if a.hsErr != "" {
		tactileCard(c, t).Padding(18).Gap(12).Children(func() {
			ui.Text(c, a.hsErr).FontSize(13).TextColor(t.Danger)
			if a.store.LoadEbirdKey() == "" {
				if filterPill(c, "hs-goto-set", "前往设置 Key", true, t).Clicked() {
					a.SelectTab(3)
				}
			}
		})
		return
	}

	if a.hsBusy {
		skeletonHotspotListCard(c, 6, t)
		return
	}

	if len(a.hsFiltered) == 0 {
		tactileCard(c, t).Padding(32).Center().Gap(10).Children(func() {
			tactileRecessedStyle(ui.Box(c).Size(52, 52), t, 26).Center().Children(func() {
				ui.Icon(c, iconMapPin).FontSize(22).TextColor(t.TextMuted)
			})
			ui.Text(c, "暂无符合条件的观鸟热点").FontSize(15).FontWeight(600)
			if filterPill(c, "hs-load-now", "刷新尝试", false, t).Clicked() {
				a.loadExploreHotspots()
			}
		})
		return
	}

	tactileCard(c, t).Clip().Dividers(1, subtleHairlineColor(t)).Children(func() {
		for _, h := range a.hsFiltered {
			spot := h
			row := ui.Row(c.Key("hspot-"+spot.LocID)).
				FillWidth().Padding(13, 16).Gap(12).AlignItems(ui.Center).
				Role(ui.RoleButton).Label(spot.LocName)
			row.Children(func() {
				ui.Column(c).Grow(1).Basis(0).Gap(3).Children(func() {
					ui.Text(c, spot.LocName).FontSize(15).FontWeight(500).SingleLine()
					ui.Textf(c, "最近活跃 · %s", model.FormatObsTime(spot.LatestObsDt)).
						FontSize(12).TextColor(t.TextMuted).SingleLine()
				})
				badge := ui.Row(c).Height(30).PaddingX(10).Radius(10).Background(softPillColor(t)).
					Border(1, subtleHairlineColor(t)).
					AlignItems(ui.Center).Gap(3).Shrink(0)
				badge.Children(func() {
					ui.Textf(c, "%d", spot.NumSpeciesAllTime).FontSize(15).FontWeight(600).FontFeatures("tnum")
					ui.Text(c, "种").FontSize(11).TextColor(t.TextMuted)
				})
			})
			if row.Clicked() {
				a.OpenExploreHotspot(spot)
			}
		}
	})
}

func (a *AppState) viewExplorePhotoDetail(c *ui.Context, t *ui.Theme, _ string) {
	item := a.regActivePhoto
	if item == nil {
		ui.Column(c).Fill().Center().Padding(24).Gap(12).Children(func() {
			ui.Text(c, "未找到该现场实拍记录")
			if filterPill(c, "ep-back", "返回探索", false, t).Clicked() {
				a.goBack()
			}
		})
		return
	}

	top, _, _, _ := c.SafeArea()
	topBtnY := max(float32(14), top+6)
	sheetRim := ui.RGBA(255, 255, 255, 0.85)
	if t.Dark {
		sheetRim = ui.RGBA(255, 255, 255, 0.14)
	}

	ui.Scroll(c.Key("ep-scroll")).Fill().Scrollbars(false).Children(func() {
		ui.Column(c).FillWidth().Padding(0, 0, 44, 0).Gap(0).Children(func() {
			count := len(item.Photos)
			heroH := float32(330) + top
			ui.Box(c).FillWidth().Height(heroH).Background(t.Surface).Children(func() {
				mobile.PageView(c, &a.regPhotoPager, count, func(i int) {
					slide := ui.Box(c.Key(fmt.Sprintf("fpslide-%d", i))).Fill().
						Role(ui.RoleButton).Label("点击放大查看实拍图")
					slide.Children(func() {
						photo := item.Photos[i]
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
					if slide.Clicked() {
						a.openNetworkPhotosViewer(item.ComName+"（现场实拍）", item.Photos, i)
					}
				}).Fill()

				ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(76 + top).PassThrough().LinearGradient(ui.LinearGradient{
					From:  ui.RGBA(0, 0, 0, 0.34),
					To:    ui.RGBA(0, 0, 0, 0),
					Angle: 180,
				})
				transitioning := a.router.Transitioning()
				if liquidGlassCircleBtnFast(c, "ep-back-btn", "返回", iconBack, 38, transitioning).
					Absolute().Left(16).Top(topBtnY).Clicked() {
					a.goBack()
				}
				if count > 0 {
					applyPhotoOverlayGlass(
						ui.Textf(c, "%d / %d", a.regPhotoPager.Page+1, count).
							Absolute().Left(18).Bottom(38).Padding(4, 10).Radius(999).
							FontSize(12).FontWeight(500).FontFeatures("tnum").
							TextColor(ui.RGB(255, 255, 255)),
						transitioning, false,
					)
					if count > 1 {
						ui.Box(c).Absolute().Right(16).Bottom(40).Children(func() {
							mobile.PageDots(c, a.regPhotoPager.Page, count)
						})
					}
				}
			})

			applyCanvasBackground(
				ui.Column(c).FillWidth().Margin(-26, 0, 0, 0).Radius(26, 26, 0, 0).
					BorderWidth(1, 0, 0, 0).BorderColor(sheetRim).
					Shadow(0, -10, 28, 0, ui.RGBA(0, 0, 0, 0.16)).
					Padding(22, 18, 0, 18).Gap(16),
				t,
			).Children(func() {
				tactileCard(c, t).Padding(16).Gap(8).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
						ui.Text(c, item.ComName).FontSize(22).FontWeight(600).LetterSpacing(-0.1).Grow(1)
						statusPill(c, model.QualityGradeZh(item.QualityGrade), softPillColor(t), opticalAccentColor(t))
					})
					ui.Text(c, item.SciName).FontSize(14).Italic().TextColor(t.TextMuted)
					ui.Textf(c, "%s · %s", model.FormatObsTime(item.ObservedOn), fallbackText(item.PlaceGuess, "野外地点")).
						FontSize(13).TextColor(secondaryTextColor(t))
					ui.Textf(c, "现场拍摄：%s", item.Observer).FontSize(12).TextColor(t.TextMuted)
					if item.ObsURL != "" {
						ui.Box(c).Padding(4, 0, 0, 0).Children(func() {
							externalLinkChip(c, "ep-link-inat", "iNaturalist 现场原记录", iconINaturalist, item.ObsURL, t)
						})
					}
				})

				if len(item.Sounds) > 0 {
					tactileCard(c, t).Padding(16).Gap(10).Children(func() {
						ui.Text(c, "本次目击现场录音").FontSize(16).FontWeight(600)
						st := a.player.Status()
						ui.Column(c).FillWidth().Dividers(1, subtleHairlineColor(t)).Children(func() {
							for i, snd := range item.Sounds {
								snd := snd
								isThis := st.SourceKey == snd.URL
								ui.Row(c.Key(fmt.Sprintf("fps-%d", i))).FillWidth().Padding(10, 0).Gap(12).AlignItems(ui.Center).Children(func() {
									ic := iconPlay
									playBtn := ui.Box(c.Key(fmt.Sprintf("fpsb-%d", i))).Size(38, 38).Radius(19).Center().Shrink(0).
										Role(ui.RoleButton).Label("播放现场录音")
									fg := t.Text
									if isThis && st.Playing {
										ic = iconPause
										fg = ui.RGB(255, 255, 255)
										c1, c2 := opticalAccentColors(t)
										playBtn.LinearGradient(ui.LinearGradient{From: c1, To: c2, Angle: 180}).
											Shadow(0, 4, 12, 0, c1.Alpha(0.35))
									} else {
										applySculptedSecondaryPillStyle(playBtn, t)
									}
									playBtn.Children(func() {
										ui.Icon(c, ic).FontSize(15).TextColor(fg)
									})
									if playBtn.Clicked() {
										if isThis && (st.Playing || st.Loading) {
											a.player.TogglePause()
										} else {
											a.player.PlayURL(snd.URL, item.ComName+" · 现场录音", item.PlaceGuess, a.cache, a.client, func() {
												a.updateUI(func() {})
											})
										}
									}
									ui.Column(c).Grow(1).Gap(2).Children(func() {
										ui.Text(c, snd.Type).FontSize(14).FontWeight(500)
										ui.Text(c, "录音者："+snd.Recordist).FontSize(12).TextColor(t.TextMuted)
									})
								})
							}
						})
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
			if filterPill(c, "ec-back", "返回探索", false, t).Clicked() {
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

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).FillWidth().Padding(immersiveTopPad(c, 14), 18, 10, 18).Gap(10).AlignItems(ui.Center).Children(func() {
			if tactileIconBtn(c, "ec-back-btn", "返回", iconBack, 36, t).Clicked() {
				a.goBack()
			}
			ui.Text(c, "观鸟清单").FontSize(16).FontWeight(600).SingleLine().Grow(1)
			if filterPill(c, "ec-goto-guide", "看图鉴", false, t).Clicked() {
				a.OpenSpeciesInGuide(obs.SpeciesCode)
			}
		})

		ui.Scroll(c.Key("ec-scroll")).Grow(1).Basis(0).FillWidth().Scrollbars(false).Children(func() {
			ui.Column(c).FillWidth().Padding(6, 18, 44, 18).Gap(14).Children(func() {
				tactileCard(c, t).Padding(16).Gap(6).Children(func() {
					ui.Text(c, obs.ComName).FontSize(22).FontWeight(600).LetterSpacing(-0.1)
					ui.Text(c, obs.SciName).FontSize(14).Italic().TextColor(t.TextMuted)
					items := []metaItem{
						{Text: model.FormatObsTime(obs.ObsDt), Tnum: true},
						{Text: fallbackText(obs.LocName, "未命名地点")},
					}
					if obs.HowMany > 0 {
						items = append(items, metaItem{Text: fmt.Sprintf("%d 只", obs.HowMany), Color: t.Text, Weight: 500, Tnum: true})
					}
					if obs.UserDisplayName != "" {
						items = append(items, metaItem{Text: obs.UserDisplayName, Color: t.TextMuted})
					}
					metaAnnotationRow(c, t, items...)
				})

				// 严格隔离卡片：绝不在此处加载图鉴通用物种照
				tactileCard(c, t).Padding(16).Gap(10).Children(func() {
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
						ui.Text(c, msg).FontSize(14)
						ui.Text(c, "eBird API v2 仅返回现场媒体计数而不直接暴露图片地址，请点击下方按钮打开原清单查看本次实拍图（不与图鉴通用图混用）。").
							FontSize(12).TextColor(t.TextMuted).LineHeight(1.5)
						if obs.SubID != "" {
							if tactilePrimaryBtn(c, "ec-open-ebird", "打开 eBird 清单原页 ↗", t).FillWidth().Clicked() {
								_ = mygo.Shell.OpenExternal("https://ebird.org/checklist/" + obs.SubID)
							}
						}
					} else {
						ui.Text(c, "本次目击仅提交了物种观测计数，未附带现场照片或录音。点击右上方「看图鉴」可浏览该鸟种生态资料。").
							FontSize(13).TextColor(t.TextMuted).LineHeight(1.5)
					}
				})

				tactileCard(c, t).Padding(16).Gap(10).Selectable().Children(func() {
					ui.Text(c, "清单上下文").FontSize(16).FontWeight(600)
					if checklist == nil {
						ui.Text(c, "暂无该清单详细信息。").FontSize(13).TextColor(t.TextMuted)
					} else {
						items := []metaItem{}
						if checklist.UserDisplayName != "" {
							items = append(items, metaItem{Text: "作者：" + checklist.UserDisplayName})
						}
						if checklist.DurationHrs > 0 {
							items = append(items, metaItem{Text: fmt.Sprintf("%.1f 小时", checklist.DurationHrs), Tnum: true})
						}
						if checklist.EffortDistanceKm > 0 {
							items = append(items, metaItem{Text: fmt.Sprintf("%.2f km", checklist.EffortDistanceKm), Tnum: true})
						}
						items = append(items, metaItem{Text: fmt.Sprintf("共 %d 种", checklist.NumSpecies), Color: t.Text, Weight: 500, Tnum: true})
						metaAnnotationRow(c, t, items...)

						if matchedItem != nil && strings.TrimSpace(api.CleanEBirdText(matchedItem.Comments)) != "" {
							tactileRecessedStyle(ui.Column(c).FillWidth().Padding(12).Gap(4), t, 12).Children(func() {
								ui.Text(c, "现场记录备注").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
								renderFieldComment(c, matchedItem.Comments, t)
							})
						}

						if len(checklist.Obs) > 0 {
							ui.Text(c, "同清单伴生鸟种").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
							limit := min(20, len(checklist.Obs))
							ui.Column(c).FillWidth().Dividers(1, subtleHairlineColor(t)).Children(func() {
								for i := 0; i < limit; i++ {
									co := checklist.Obs[i]
									name := co.SpeciesCode
									if tx, ok := a.taxonByCode[co.SpeciesCode]; ok {
										name = tx.ComName
									}
									ui.Row(c.Key(fmt.Sprintf("ec-co-%d", i))).FillWidth().Justify(ui.SpaceBetween).Padding(9, 0).Children(func() {
										ui.Text(c, name).FontSize(14)
										ui.Text(c, fallbackText(co.HowManyStr, "在场")).FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
									})
								}
							})
						}
					}
				})

				if tactilePrimaryBtn(c, "ec-add-journal", "记入观鸟日志", t).FillWidth().Height(48).Radius(24).Clicked() {
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
			if filterPill(c, "es-back", "返回热点", false, t).Clicked() {
				a.goBack()
			}
		})
		return
	}

	obsList := a.hsObsMap[spot.LocID]
	busy := a.hsObsBusy[spot.LocID]

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).FillWidth().Padding(immersiveTopPad(c, 14), 18, 10, 18).Gap(10).AlignItems(ui.Center).Children(func() {
			if tactileIconBtn(c, "es-back-btn", "返回", iconBack, 36, t).Clicked() {
				a.goBack()
			}
			ui.Text(c, "观鸟热点").FontSize(16).FontWeight(600).SingleLine().Grow(1)
		})

		ui.Scroll(c.Key("es-scroll")).Grow(1).Basis(0).FillWidth().Scrollbars(false).Children(func() {
			ui.Column(c).FillWidth().Padding(6, 18, 44, 18).Gap(14).Children(func() {
				tactileCard(c, t).Padding(16).Gap(10).Children(func() {
					ui.Text(c, spot.LocName).FontSize(22).FontWeight(600).LetterSpacing(-0.1)
					ui.Textf(c, "累计 %d 种 · 最近活跃 %s · %.4f, %.4f",
						spot.NumSpeciesAllTime, model.FormatObsTime(spot.LatestObsDt), spot.Lat, spot.Lng).
						FontSize(12).TextColor(t.TextMuted)

					ui.Row(c).FillWidth().Gap(10).Padding(4, 0, 0, 0).Children(func() {
						if tactilePrimaryBtn(c, "hs-new-log", "在此处写日志", t).Grow(1).Height(42).Radius(21).Clicked() {
							a.OpenNewJournalModal("", spot.LocName)
						}
						if monochromeButton(c, "hs-open-web", "网页查看 ↗", false, t).Height(42).Radius(21).Clicked() {
							_ = mygo.Shell.OpenExternal("https://ebird.org/hotspot/" + spot.LocID)
						}
					})
				})

				tactileCard(c, t).Padding(16).Gap(10).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "近 30 天目击鸟种").FontSize(16).FontWeight(600)
						if busy {
							ui.Spinner(c).Label("加载鸟种")
						} else {
							ui.Textf(c, "%d 条", len(obsList)).FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
						}
					})

					if len(obsList) == 0 && !busy {
						ui.Text(c, "该热点近 30 天暂无公开目击记录。").FontSize(13).TextColor(t.TextMuted)
					} else {
						ui.Column(c).FillWidth().Dividers(1, subtleHairlineColor(t)).Children(func() {
							for i, o := range obsList {
								o := o
								ui.Row(c.Key(fmt.Sprintf("esobs-%d", i))).FillWidth().Padding(10, 0).Gap(10).AlignItems(ui.Center).Children(func() {
									ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
										ui.Text(c, o.ComName).FontSize(15).FontWeight(500).SingleLine()
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
						})
					}
				})
			})
		})
	})
}

func (a *AppState) viewRegionPickerSheet(c *ui.Context, t *ui.Theme) {
	sheetBg := t.Background
	mobile.Sheet(c, &a.regRegionSheetOpen, mobile.SheetOptions{
		Detents:    []float32{0.72, 0.92},
		Background: &sheetBg,
		Label:      "选择探索地区",
	}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 20, 14, 20).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "选择观鸟地区").FontSize(16).FontWeight(600)
				if filterPill(c, "rp-close", "关闭", false, t).Clicked() {
					a.regRegionSheetOpen = false
				}
			})
			minimalDivider(c, t)

			ui.Column(c).Padding(14, 18, 10, 18).Gap(10).Children(func() {
				if a.regRegionTarget == "explore" {
					gpsBtn := applySculptedSecondaryPillStyle(
						ui.Row(c.Key("rp-gps")).FillWidth().Height(44).PaddingX(14).Radius(14).
							Gap(10).AlignItems(ui.Center).
							Role(ui.RoleButton).Label("使用手机 GPS 定位周边"),
						t,
					)
					gpsBtn.Children(func() {
						if a.regGPSBusy {
							ui.Spinner(c).Label("定位中")
						} else {
							ui.Icon(c, iconLocate).FontSize(17).TextColor(opticalAccentColor(t))
						}
						ui.Text(c, "使用手机 GPS 定位周边").FontSize(14).FontWeight(500).TextColor(t.Text).Grow(1)
					})
					if gpsBtn.Clicked() {
						a.fillGPSLocationAsync("explore")
					}
					if a.regGPSNote != "" {
						ui.Text(c, a.regGPSNote).FontSize(12).TextColor(t.Warning)
					}

					ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
						in := minimalTextInput(c, "rp-custom", &a.regCustomCode,
							"地区码或纬度,经度（如 US-NY / 39.9,116.4）", "自定义地区", t).Grow(1)
						if in.Submitted() || filterPill(c, "rp-go", "查询", true, t).Height(40).Radius(12).Clicked() {
							a.regRegionSheetOpen = false
							a.resetExploreForRegionChange()
						}
					})
				}

				minimalSearchBar(c, "rp-search", &a.regRegionQuery, "搜索省市或地区", "搜索地区", t)
			})

			ui.Scroll(c.Key("rp-list")).Grow(1).Basis(0).FillWidth().Scrollbars(false).Padding(4, 18, 28, 18).Children(func() {
				q := strings.ToLower(strings.TrimSpace(a.regRegionQuery))
				tactileCard(c, t).Clip().Dividers(1, subtleHairlineColor(t)).Children(func() {
					for _, reg := range api.ChinaRegions {
						r := reg
						if q != "" && !strings.Contains(strings.ToLower(r.Name), q) && !strings.Contains(strings.ToLower(r.Code), q) {
							continue
						}
						active := false
						switch a.regRegionTarget {
						case "settings":
							active = a.setRegionLabel == r.Name
						case "seasonal":
							active = strings.EqualFold(a.seasonalRegionCode, r.Code)
						default:
							active = a.regCustomCode == "" && a.regRegionLabel == r.Name
						}
						row := ui.Row(c.Key("regopt-"+r.Code)).FillWidth().Padding(13, 16).
							Justify(ui.SpaceBetween).AlignItems(ui.Center).
							Role(ui.RoleButton).Label(r.Name)
						row.Children(func() {
							weight := 400
							if active {
								weight = 500
							}
							ui.Text(c, r.Name).FontSize(15).FontWeight(weight)
							if active {
								ui.Icon(c, iconCheck).FontSize(16).TextColor(opticalAccentColor(t))
							}
						})
						if row.Clicked() {
							a.regRegionSheetOpen = false
							switch a.regRegionTarget {
							case "settings":
								a.setRegionLabel = r.Name
								a.saveRegionPreferenceOnly(c, r.Code)
							case "seasonal":
								a.seasonalRegionCode = r.Code
								a.seasonalRegionLabel = r.Name
								a.seasonalScroll.Y = 0
							default:
								a.regCustomCode = ""
								a.regRegionLabel = r.Name
								a.resetExploreForRegionChange()
							}
						}
					}
				})
			})
		})
	})
}

func metricMiniCard(c *ui.Context, t *ui.Theme, label, val string) {
	ui.Column(c).Grow(1).Basis(0).AlignItems(ui.Center).Gap(2).Children(func() {
		ui.Text(c, val).FontSize(20).FontWeight(600).FontFeatures("tnum")
		ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
	})
}

type commentSegment struct {
	Text     string
	HasColor bool
	Color    ui.Color
}

func isASCIIAlphaNum(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func parseHexNibble(b byte) (uint8, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	default:
		return 0, false
	}
}

func parseHexColorToken(s string) (ui.Color, bool) {
	switch len(s) {
	case 3:
		r, ok1 := parseHexNibble(s[0])
		g, ok2 := parseHexNibble(s[1])
		b, ok3 := parseHexNibble(s[2])
		if !ok1 || !ok2 || !ok3 {
			return ui.Color{}, false
		}
		return ui.RGB(r*17, g*17, b*17), true
	case 6:
		var vals [6]uint8
		for i := range 6 {
			v, ok := parseHexNibble(s[i])
			if !ok {
				return ui.Color{}, false
			}
			vals[i] = v
		}
		return ui.RGB(vals[0]<<4|vals[1], vals[2]<<4|vals[3], vals[4]<<4|vals[5]), true
	default:
		return ui.Color{}, false
	}
}

// parseCommentColorSegments 识别独立标准十六进制色号（#RGB / #RRGGBB），避免误伤 HTML 实体或普通编号。
func parseCommentColorSegments(text string) []commentSegment {
	var out []commentSegment
	last := 0
	for i := 0; i < len(text); i++ {
		if text[i] != '#' {
			continue
		}
		if i > 0 && (isASCIIAlphaNum(text[i-1]) || text[i-1] == '&') {
			continue
		}
		j := i + 1
		for j < len(text) && isASCIIAlphaNum(text[j]) {
			j++
		}
		hexLen := j - (i + 1)
		if hexLen != 3 && hexLen != 6 {
			i = j - 1
			continue
		}
		col, ok := parseHexColorToken(text[i+1 : j])
		if !ok {
			i = j - 1
			continue
		}
		if i > last {
			out = append(out, commentSegment{Text: text[last:i]})
		}
		out = append(out, commentSegment{Text: text[i:j], HasColor: true, Color: col})
		last = j
		i = j - 1
	}
	if last < len(text) {
		out = append(out, commentSegment{Text: text[last:]})
	}
	return out
}

// renderFieldComment 自动解码 HTML 实体并在存在真实色号时渲染与正文字号等高（14px）、带边框的圆形色块。
func renderFieldComment(c *ui.Context, raw string, t *ui.Theme) {
	clean := api.CleanEBirdText(raw)
	segs := parseCommentColorSegments(clean)
	hasColor := false
	for _, s := range segs {
		if s.HasColor {
			hasColor = true
			break
		}
	}
	if !hasColor {
		ui.Text(c, clean).FontSize(14).LineHeight(1.5)
		return
	}

	swatchBorder := ui.RGBA(15, 23, 42, 0.24)
	if t.Dark {
		swatchBorder = ui.RGBA(255, 255, 255, 0.36)
	}

	lines := strings.Split(strings.ReplaceAll(clean, "\r\n", "\n"), "\n")
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		for _, line := range lines {
			lineSegs := parseCommentColorSegments(line)
			if len(lineSegs) == 0 {
				continue
			}
			ui.Row(c).FillWidth().Gap(4).Wrap().AlignItems(ui.Center).Children(func() {
				for _, seg := range lineSegs {
					if seg.HasColor {
						ui.Row(c).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
							ui.Box(c).Size(14, 14).Radius(7).Background(seg.Color).Border(1, swatchBorder).Shrink(0)
							ui.Text(c, seg.Text).FontSize(14).TextColor(secondaryTextColor(t)).FontFeatures("tnum")
						})
					} else if seg.Text != "" {
						ui.Text(c, seg.Text).FontSize(14).LineHeight(1.5)
					}
				}
			})
		}
	})
}
