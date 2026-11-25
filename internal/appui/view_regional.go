package appui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func parseLatLng(raw string) (float64, float64, bool) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), "，", ",")
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, false
	}
	return lat, lng, true
}

// LoadRegionalData 按当前区域设置拉取 eBird 近期/珍稀记录、区域统计或现场实拍记录。
func (a *AppState) LoadRegionalData() {
	custom := strings.TrimSpace(a.regCustomCode)
	lat, lng, isGeo := parseLatLng(custom)

	regionCode := custom
	if regionCode == "" || isGeo {
		regionCode = regionCodeByLabel(a.regRegionLabel)
	}
	if regionCode == "" {
		regionCode = "CN-11"
	}
	a.regionalRegion = regionCode

	if a.regSourceTab == 2 {
		if isGeo {
			a.loadRegionalFieldPhotosByCoordAsync(lat, lng)
		} else {
			a.loadRegionalFieldPhotosAsync(regionCode)
		}
		return
	}

	key := a.store.LoadEbirdKey()
	if key == "" {
		a.regObsErr = "查询 eBird 区域观测记录需要配置 API Key，请前往「设置」页面填写（或切换至“区域现场实拍”查看免 Key 实拍记录）"
		a.regObsList = nil
		a.invalidateUI()
		return
	}

	backDays := 14
	switch a.regBackLabel {
	case "近 7 天":
		backDays = 7
	case "近 30 天":
		backDays = 30
	}
	notable := a.regSourceTab == 1

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
			if err != nil {
				a.regObsErr = fmt.Sprintf("加载区域观测失败: %v", err)
				a.regObsList = nil
				return
			}
			a.regObsList = obs
			if statErr == nil {
				a.regStat = stat
			} else {
				a.regStat = nil
			}
			if len(obs) > 0 {
				a.regSelected = 0
				a.selectRegionalObs(0)
			} else {
				a.regSelected = -1
				a.regActiveSubID = ""
			}
		})
	}()
}

func (a *AppState) loadRegionalFieldPhotosAsync(regionCode string) {
	lat, lng := 39.9042, 116.4074
	if opt, ok := api.LookupRegionOption(regionCode); ok {
		lat, lng = opt.Lat, opt.Lng
	}
	a.loadRegionalFieldPhotosByCoordAsync(lat, lng)
}

func (a *AppState) loadRegionalFieldPhotosByCoordAsync(lat, lng float64) {
	a.regPhotoBusy = true
	a.regPhotoErr = ""
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
		items, err := a.client.FetchRegionalPhotoObservations(ctx, lat, lng, 50, 30)
		cancel()

		a.updateUI(func() {
			a.regPhotoBusy = false
			if err != nil {
				a.regPhotoErr = fmt.Sprintf("加载现场实拍记录失败: %v", err)
				a.regFieldPhotos = nil
				return
			}
			a.regFieldPhotos = items
			if len(items) > 0 {
				a.regPhotoSelected = 0
				a.regPhotoGalleryIdx = 0
				for _, p := range items[0].Photos {
					a.loadImageAsync("field:"+p.URL, p.URL, false)
				}
			} else {
				a.regPhotoSelected = -1
			}
		})
	}()
}

func (a *AppState) selectRegionalObs(idx int) {
	if idx < 0 || idx >= len(a.regObsList) {
		return
	}
	obs := a.regObsList[idx]
	a.regSelected = idx
	a.regActiveSubID = obs.SubID
	a.regActiveObsID = obs.ObsID
	a.regActiveSpCode = obs.SpeciesCode
	if obs.SubID == "" {
		return
	}
	if _, ok := a.regChecklistMap[obs.SubID]; ok || a.regChecklistBusy[obs.SubID] {
		return
	}

	key := a.store.LoadEbirdKey()
	if key == "" {
		return
	}
	a.regChecklistBusy[obs.SubID] = true

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		detail, err := a.client.FetchChecklistDetail(ctx, key, obs.SubID)
		cancel()

		a.updateUI(func() {
			delete(a.regChecklistBusy, obs.SubID)
			if err == nil && detail != nil {
				a.regChecklistMap[obs.SubID] = detail
			}
		})
	}()
}

func (a *AppState) viewRegional(c *ui.Context, t *ui.Theme) {
	ui.Split(c, &a.regionSplit,
		func() { a.viewRegionalListPane(c, t) },
		func() { a.viewRegionalDetailPane(c, t) },
	).SplitReset(400).Fill()
}

func (a *AppState) viewRegionalListPane(c *ui.Context, t *ui.Theme) {
	regionLabels := make([]string, 0, len(api.ChinaRegions))
	for _, r := range api.ChinaRegions {
		regionLabels = append(regionLabels, r.Name)
	}

	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(14, 14, 10, 14).Gap(10).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "区域观鸟动态").FontSize(17).Bold()
				if a.regObsBusy || a.regPhotoBusy {
					ui.Spinner(c).Label("加载区域观测中")
				} else if ui.Button(c.Key("reg-refresh"), "").Label("刷新").Tooltip("重新拉取区域观测数据").Children(func() {
					ui.Icon(c, iconRefresh).FontSize(14)
				}).Clicked() {
					a.LoadRegionalData()
				}
			})

			if ui.Segmented(c.Key("reg-source-seg"), &a.regSourceTab, "eBird 近期观测", "eBird 珍稀记录", "区域现场实拍").
				FillEqually().Label("观测数据源").Changed() {
				a.LoadRegionalData()
			}

			ui.Row(c).Gap(8).FillWidth().Children(func() {
				if ui.Select(c.Key("reg-region-sel"), &a.regRegionLabel, regionLabels).Label("选择地区").Grow(1).Changed() {
					a.regCustomCode = ""
					a.LoadRegionalData()
				}
				if a.regSourceTab != 2 {
					if ui.Select(c.Key("reg-back-sel"), &a.regBackLabel, []string{"近 7 天", "近 14 天", "近 30 天"}).
						Label("回溯天数").Width(104).Changed() {
						a.LoadRegionalData()
					}
				}
			})

			ui.Row(c).Gap(6).FillWidth().Children(func() {
				in := ui.TextInput(c.Key("reg-custom-code"), &a.regCustomCode).
					Placeholder("自定义区域码、热点 ID 或经纬度（如 US-NY、L12170807、39.99,116.39）").Label("自定义区域码").Grow(1)
				if in.Submitted() || ui.Button(c.Key("reg-go-btn"), "查询").Clicked() {
					a.LoadRegionalData()
				}
			})

			if a.regSourceTab != 2 && a.regStat != nil {
				ui.Row(c).Gap(8).Wrap().Children(func() {
					infoChip(c, t, "昨日清单", fmt.Sprintf("%d 份", a.regStat.NumChecklists))
					infoChip(c, t, "贡献观鸟者", fmt.Sprintf("%d 人", a.regStat.NumContributors))
					infoChip(c, t, "记录鸟种", fmt.Sprintf("%d 种", a.regStat.NumSpecies))
				})
			}
		})

		if a.regSourceTab == 2 {
			a.viewRegionalPhotoList(c, t)
			return
		}

		if a.regObsErr != "" {
			ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
				ui.Text(c, a.regObsErr).FontSize(13).TextColor(t.Danger)
				if a.store.LoadEbirdKey() == "" {
					if ui.PrimaryButton(c.Key("reg-goto-settings"), "前往设置 API Key").Clicked() {
						a.SetPage("settings")
					}
				}
			})
			return
		}

		a.regList.Key = func(i int) any {
			o := a.regObsList[i]
			return o.SubID + "_" + o.SpeciesCode + "_" + o.ObsDt
		}
		a.regList.Label = func(i int) string { return a.regObsList[i].ComName }
		a.regList.Selected = &a.regSelected

		list := ui.List(c.Key("reg-obs-list"), &a.regList, len(a.regObsList)).Grow(1).FillWidth()
		list.Rows(func(row ui.ListRow) {
			o := a.regObsList[row.Index]
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

			ui.Column(rc).Padding(10, 14).Gap(3).Background(bg).Children(func() {
				ui.Row(rc).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(rc, o.ComName).Bold().TextColor(textColor).SingleLine().Grow(1)
					cntText := "在场"
					if o.HowMany > 0 {
						cntText = fmt.Sprintf("%d 只", o.HowMany)
					}
					ui.Text(rc, cntText).FontSize(11.5).TextColor(mutedColor)
				})
				ui.Textf(rc, "%s · %s", fallbackText(o.LocName, "未命名地点"), o.ObsDt).
					FontSize(11.5).TextColor(mutedColor).SingleLine()

				if o.HasRichMedia || o.ObsValid {
					ui.Row(rc).Gap(6).Children(func() {
						if o.HasRichMedia {
							b := ui.Badge(rc, "含现场媒体")
							if row.Selected() {
								b.Background(ui.RGBA(255, 255, 255, 0.2)).TextColor(t.AccentText)
							}
						}
						if o.ObsValid {
							b := ui.Badge(rc, "已核验")
							if row.Selected() {
								b.Background(ui.RGBA(255, 255, 255, 0.2)).TextColor(t.AccentText)
							} else {
								b.Background(t.Success.Alpha(0.14)).TextColor(t.Success)
							}
						}
					})
				}
			})
		}).Children(func() {
			if len(a.regObsList) == 0 && !a.regObsBusy {
				ui.Column(c).Fill().Center().Padding(24).Gap(6).Children(func() {
					ui.Text(c, "该地区近期暂无公开观测记录").Bold()
					ui.Text(c, "可尝试切换回溯天数或更换省份").FontSize(12.5).TextColor(t.TextMuted)
				})
			}
		})

		if list.Changed() && a.regSelected >= 0 {
			a.selectRegionalObs(a.regSelected)
		}
	})
}

func (a *AppState) viewRegionalPhotoList(c *ui.Context, t *ui.Theme) {
	if a.regPhotoErr != "" {
		ui.Column(c).Fill().Padding(20).Children(func() {
			ui.Text(c, a.regPhotoErr).FontSize(13).TextColor(t.Danger)
		})
		return
	}

	a.regPhotoList.Key = func(i int) any { return a.regFieldPhotos[i].ID }
	a.regPhotoList.Label = func(i int) string { return a.regFieldPhotos[i].ComName }
	a.regPhotoList.Selected = &a.regPhotoSelected

	list := ui.List(c.Key("reg-photo-list"), &a.regPhotoList, len(a.regFieldPhotos)).Grow(1).FillWidth()
	list.Rows(func(row ui.ListRow) {
		item := a.regFieldPhotos[row.Index]
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
		ui.Column(rc).Padding(10, 14).Gap(3).Background(bg).Children(func() {
			ui.Row(rc).FillWidth().Justify(ui.SpaceBetween).Children(func() {
				ui.Text(rc, item.ComName).Bold().TextColor(textColor).SingleLine().Grow(1)
				ui.Textf(rc, "%d 张实拍", len(item.Photos)).FontSize(11.5).TextColor(mutedColor)
			})
			ui.Textf(rc, "%s · %s", item.PlaceGuess, item.ObservedOn).
				FontSize(11.5).TextColor(mutedColor).SingleLine()
		})
	}).Children(func() {
		if len(a.regFieldPhotos) == 0 && !a.regPhotoBusy {
			ui.Column(c).Fill().Center().Padding(24).Children(func() {
				ui.Text(c, "该区域暂无现场实拍记录").TextColor(t.TextMuted)
			})
		}
	})

	if list.Changed() && a.regPhotoSelected >= 0 && a.regPhotoSelected < len(a.regFieldPhotos) {
		a.regPhotoGalleryIdx = 0
		for _, p := range a.regFieldPhotos[a.regPhotoSelected].Photos {
			a.loadImageAsync("field:"+p.URL, p.URL, false)
		}
	}
}

func (a *AppState) viewRegionalDetailPane(c *ui.Context, t *ui.Theme) {
	if a.regSourceTab == 2 {
		a.viewRegionalFieldPhotoDetail(c, t)
		return
	}

	if a.regSelected < 0 || a.regSelected >= len(a.regObsList) {
		ui.Column(c).Fill().Center().Gap(8).Padding(32).Children(func() {
			ui.Icon(c, iconGlobe).FontSize(32).TextColor(t.TextMuted)
			ui.Text(c, "选择一条区域观测记录").FontSize(15).Bold()
			ui.Text(c, "将展示本次观鸟清单详情与现场媒体信息（不混用图鉴资料图）").FontSize(12.5).TextColor(t.TextMuted)
		})
		return
	}

	obs := a.regObsList[a.regSelected]
	checklist := a.regChecklistMap[obs.SubID]
	clBusy := a.regChecklistBusy[obs.SubID]

	// 查找本次观测对应的清单项明细
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

	ui.Scroll(c.Key("reg-detail-scroll")).Fill().Padding(20).Gap(16).Children(func() {
		ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Start).Gap(12).Children(func() {
			ui.Column(c).Grow(1).Gap(4).Children(func() {
				ui.Text(c, obs.ComName).FontSize(22).Bold()
				ui.Text(c, obs.SciName).Italic().FontSize(14).TextColor(t.TextMuted)
			})
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c.Key("reg-goto-guide"), "在图鉴中查看该物种").Clicked() {
					a.OpenSpeciesInGuide(obs.SpeciesCode)
				}
				if ui.PrimaryButton(c.Key("reg-add-log"), "记入观鸟日志").Clicked() {
					a.OpenNewJournalModal(obs.SpeciesCode, obs.LocName)
				}
			})
		})

		ui.Row(c).Gap(8).Wrap().Children(func() {
			infoChip(c, t, "观测时间", obs.ObsDt)
			infoChip(c, t, "观测地点", fallbackText(obs.LocName, "未命名"))
			if obs.HowMany > 0 {
				infoChip(c, t, "观测数量", fmt.Sprintf("%d 只", obs.HowMany))
			}
			infoChip(c, t, "坐标", fmt.Sprintf("%.4f, %.4f", obs.Lat, obs.Lng))
			if obs.SubID != "" {
				infoChip(c, t, "清单 ID", obs.SubID)
			}
			if obs.UserDisplayName != "" {
				infoChip(c, t, "观测者", obs.UserDisplayName)
			}
		})

		// 核心原则：严格区分本次观测现场媒体，绝不加载图鉴物种标准图
		ui.Column(c).FillWidth().Padding(14).Gap(10).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "本次观测现场媒体（非图鉴标准图）").Bold()
				if clBusy {
					ui.Spinner(c).Label("获取清单信息")
				}
			})

			photoCount := 0
			audioCount := 0
			if matchedItem != nil && matchedItem.MediaCounts != nil {
				photoCount = matchedItem.MediaCounts["P"]
				audioCount = matchedItem.MediaCounts["A"]
			}

			if photoCount > 0 || audioCount > 0 || obs.HasRichMedia {
				msg := fmt.Sprintf("观测者为本次记录 (%s) 上传了现场媒体：", obs.SubID)
				if photoCount > 0 || audioCount > 0 {
					msg += fmt.Sprintf(" %d 张现场照片 · %d 段现场录音。", photoCount, audioCount)
				} else {
					msg += "包含富媒体附件。"
				}
				ui.Text(c, msg).FontSize(13)
				ui.Text(c, "说明：eBird API v2 清单接口仅返回现场媒体计数，不直接返回原始照片下载链接；点击下方按钮可在浏览器打开本次观测清单查看现场原图。").
					FontSize(12).TextColor(t.TextMuted)
				if obs.SubID != "" {
					if ui.PrimaryButton(c.Key("reg-open-cl"), "在浏览器查看本次 eBird 清单原图").Clicked() {
						_ = mygo.Shell.OpenExternal("https://ebird.org/checklist/" + obs.SubID)
					}
				}
			} else {
				ui.Text(c, "本次观测仅上报了物种计数，观测者未上传现场照片或录音。如需查看该鸟种标准外形与鸣声，请点击右上方「在图鉴中查看该物种」。").
					FontSize(12.5).TextColor(t.TextMuted)
			}
		})

		// 观测清单详细上下文
		ui.Column(c).FillWidth().Padding(14).Gap(10).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Selectable().Children(func() {
			ui.Text(c, "观鸟清单上下文").Bold()
			if clBusy && checklist == nil {
				ui.Text(c, "正在加载该清单的完整观测记录…").FontSize(12.5).TextColor(t.TextMuted)
			} else if checklist == nil {
				ui.Text(c, "暂无清单明细（可能未提供 subId 或需要有效的 eBird API Key）。").FontSize(12.5).TextColor(t.TextMuted)
			} else {
				ui.Row(c).Gap(8).Wrap().Children(func() {
					if checklist.UserDisplayName != "" {
						infoChip(c, t, "清单作者", checklist.UserDisplayName)
					}
					if checklist.DurationHrs > 0 {
						infoChip(c, t, "观鸟时长", fmt.Sprintf("%.1f 小时", checklist.DurationHrs))
					}
					if checklist.EffortDistanceKm > 0 {
						infoChip(c, t, "行进距离", fmt.Sprintf("%.2f km", checklist.EffortDistanceKm))
					}
					infoChip(c, t, "同清单物种数", fmt.Sprintf("%d 种", checklist.NumSpecies))
				})

				if matchedItem != nil && strings.TrimSpace(matchedItem.Comments) != "" {
					ui.Column(c).Padding(10).Gap(4).Radius(8).Background(t.Background).Children(func() {
						ui.Text(c, "观测者现场备注：").FontSize(12).Bold()
						ui.Text(c, matchedItem.Comments).FontSize(13).LineHeight(1.5)
					})
				}

				if len(checklist.Obs) > 0 {
					ui.Text(c, "同清单伴生鸟种：").FontSize(12.5).Bold()
					maxRows := min(18, len(checklist.Obs))
					for i := 0; i < maxRows; i++ {
						co := checklist.Obs[i]
						name := co.SpeciesCode
						if tx, ok := a.taxonByCode[co.SpeciesCode]; ok {
							name = tx.ComName
						}
						cnt := fallbackText(co.HowManyStr, "在场")
						ui.Row(c.Key(fmt.Sprintf("cl-sp-%d", i))).FillWidth().Justify(ui.SpaceBetween).Padding(4, 8).Radius(6).Background(t.Background).Children(func() {
							ui.Text(c, name).FontSize(12.5)
							ui.Text(c, cnt).FontSize(12).TextColor(t.TextMuted)
						})
					}
				}
			}
		})
	})
}

func (a *AppState) viewRegionalFieldPhotoDetail(c *ui.Context, t *ui.Theme) {
	if a.regPhotoSelected < 0 || a.regPhotoSelected >= len(a.regFieldPhotos) {
		ui.Column(c).Fill().Center().Gap(8).Padding(32).Children(func() {
			ui.Icon(c, iconImage).FontSize(32).TextColor(t.TextMuted)
			ui.Text(c, "选择一条区域现场实拍记录").FontSize(15).Bold()
			ui.Text(c, "展示的均为观测者在当次目击现场拍摄上传的真实照片与录音").FontSize(12.5).TextColor(t.TextMuted)
		})
		return
	}

	item := a.regFieldPhotos[a.regPhotoSelected]
	ui.Scroll(c.Key("reg-field-scroll")).Fill().Padding(20).Gap(16).Children(func() {
		ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Start).Gap(12).Children(func() {
			ui.Column(c).Grow(1).Gap(4).Children(func() {
				ui.Text(c, item.ComName).FontSize(22).Bold()
				ui.Text(c, item.SciName).Italic().FontSize(14).TextColor(t.TextMuted)
			})
			ui.Row(c).Gap(8).Children(func() {
				if item.ObsURL != "" {
					ui.Link(c, "查看 iNaturalist 原记录", item.ObsURL)
				}
			})
		})

		ui.Row(c).Gap(8).Wrap().Children(func() {
			infoChip(c, t, "目击日期", item.ObservedOn)
			infoChip(c, t, "现场地点", item.PlaceGuess)
			infoChip(c, t, "拍摄者", item.Observer)
			if item.QualityGrade != "" {
				infoChip(c, t, "鉴定等级", item.QualityGrade)
			}
		})

		ui.Column(c).FillWidth().Padding(14).Gap(10).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Text(c, "本次目击现场实拍图").Bold()
			if len(item.Photos) > 0 {
				if a.regPhotoGalleryIdx >= len(item.Photos) {
					a.regPhotoGalleryIdx = 0
				}
				p := item.Photos[a.regPhotoGalleryIdx]
				imgKey := "field:" + p.URL
				a.loadImageAsync(imgKey, p.URL, false)
				if src, ok := a.imgCache[imgKey]; ok {
					ui.Image(c, src).FillWidth().Height(300).Fit(ui.Contain).Radius(8).Background(t.Background).Label(item.ComName)
				} else {
					ui.Box(c).FillWidth().Height(260).Center().Radius(8).Background(t.Background).Children(func() {
						ui.Spinner(c).Label("加载现场照片中")
					})
				}
				ui.Textf(c, "来源：%s · 署名：%s", p.Source, p.Attribution).FontSize(11.5).TextColor(t.TextMuted)
				if len(item.Photos) > 1 {
					ui.Row(c).Gap(6).Children(func() {
						for i := range item.Photos {
							btn := ui.Button(c.Key(fmt.Sprintf("fp-tab-%d", i)), fmt.Sprintf("现场图 %d", i+1))
							if i == a.regPhotoGalleryIdx {
								btn.Background(t.Accent).TextColor(t.AccentText)
							}
							if btn.Clicked() {
								a.regPhotoGalleryIdx = i
							}
						}
					})
				}
			}
		})

		if len(item.Sounds) > 0 {
			ui.Column(c).FillWidth().Padding(14).Gap(8).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Text(c, "本次目击现场录音").Bold()
				st := a.player.Status()
				for i, snd := range item.Sounds {
					isThis := st.SourceKey == snd.URL
					ui.Row(c.Key(fmt.Sprintf("fp-snd-%d", i))).FillWidth().Padding(8, 10).Gap(10).AlignItems(ui.Center).Radius(8).Background(t.Background).Children(func() {
						btnIcon := iconPlay
						if isThis && st.Playing {
							btnIcon = iconPause
						}
						if ui.Button(c.Key(fmt.Sprintf("fp-snd-b-%d", i)), "").Label("播放现场录音").Children(func() {
							ui.Icon(c, btnIcon).FontSize(13)
						}).Clicked() {
							if isThis && st.Active {
								a.player.TogglePause()
							} else {
								a.player.PlayURL(snd.URL, item.ComName+" · 现场录音", item.PlaceGuess, a.cache, a.client, func() {
									a.updateUI(func() {})
								})
							}
						}
						ui.Textf(c, "%s · %s", snd.Type, snd.Recordist).FontSize(12.5)
					})
				}
			})
		}
	})
}
