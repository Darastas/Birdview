package appui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/model"
	"github.com/Darastas/Birdview/internal/store"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"
	"github.com/egoist/mygo/ui/netimage"
)

// seasonalPhaseColors 返回四大观鸟时令状态的柔和徽章配色（兼顾浅色与深色模式可读性）。
func seasonalPhaseColors(phase model.SeasonalPhase, dark bool) (bg, fg ui.Color) {
	switch phase {
	case model.PhaseLeaving: // 马上离开：暖琥珀/珊瑚
		if dark {
			return ui.RGBA(249, 115, 22, 0.22), ui.Hex("#FDBA74")
		}
		return ui.Hex("#FFF1E6"), ui.Hex("#C2410C")
	case model.PhaseInSeason: // 当季：草木翠绿
		if dark {
			return ui.RGBA(16, 185, 129, 0.22), ui.Hex("#6EE7B7")
		}
		return ui.Hex("#E7F6EF"), ui.Hex("#157A5C")
	case model.PhaseJustArrived: // 刚到：晴空蓝
		if dark {
			return ui.RGBA(56, 189, 248, 0.22), ui.Hex("#7DD3FC")
		}
		return ui.Hex("#E8F4FD"), ui.Hex("#0369A1")
	case model.PhaseArrivingSoon: // 马上到：柔和鸢尾紫
		if dark {
			return ui.RGBA(167, 139, 250, 0.22), ui.Hex("#C4B5FD")
		}
		return ui.Hex("#F2EEFE"), ui.Hex("#6D28D9")
	default:
		if dark {
			return ui.RGBA(148, 163, 184, 0.16), ui.Hex("#94A3B8")
		}
		return ui.Hex("#EDF1F5"), ui.Hex("#64748B")
	}
}

// seasonalPhenologyBarStyle 返回无边框柔和胶囊柱状月历的渐变色彩与韵律高度（灵感源自 Merlin Bird ID 与 Apple Weather）。
func seasonalPhenologyBarStyle(ph model.SeasonalPhase, dark bool) (topCol, botCol ui.Color, barH float32) {
	switch ph {
	case model.PhaseInSeason:
		if dark {
			return ui.Hex("#5EEAD4"), ui.Hex("#10B981"), 28
		}
		return ui.Hex("#45B08C"), ui.Hex("#2B8A6E"), 28
	case model.PhaseJustArrived:
		if dark {
			return ui.Hex("#7DD3FC"), ui.Hex("#38BDF8"), 20
		}
		return ui.Hex("#5CB3F0"), ui.Hex("#2C89D1"), 20
	case model.PhaseLeaving:
		if dark {
			return ui.Hex("#FDBA74"), ui.Hex("#F97316"), 20
		}
		return ui.Hex("#F4A261"), ui.Hex("#DF7742"), 20
	case model.PhaseArrivingSoon:
		if dark {
			return ui.Hex("#C4B5FD"), ui.Hex("#8B5CF6"), 13
		}
		return ui.Hex("#A594F9"), ui.Hex("#7E6AE8"), 13
	default:
		if dark {
			return ui.RGBA(255, 255, 255, 0.20), ui.RGBA(255, 255, 255, 0.20), 5
		}
		return ui.RGBA(100, 116, 139, 0.24), ui.RGBA(100, 116, 139, 0.24), 5
	}
}

type phenMetricsKey struct{}

// renderSeasonalPhenologyChart 渲染柔和无边框的 1–12 月物候胶囊韵律月历（详请页与分享图共用）。
func renderSeasonalPhenologyChart(c *ui.Context, cal [12]model.SeasonalPhase, activeMonth int, dark, interactive bool, onSelect func(int)) {
	trackBg := ui.Hex("#F3F6F8")
	pillBg := ui.Hex("#FFFFFF")
	textCol := ui.Hex("#16232B")
	mutedCol := ui.Hex("#64748B")
	if dark {
		trackBg = ui.RGBA(255, 255, 255, 0.04)
		pillBg = ui.RGBA(255, 255, 255, 0.12)
		textCol = ui.RGB(255, 255, 255)
		mutedCol = ui.RGBA(255, 255, 255, 0.56)
	}

	var st *segMetrics
	if interactive {
		st = ui.Local(c.Root(), phenMetricsKey{}, func() segMetrics { return segMetrics{} })
	}

	ui.Box(c).FillWidth().Radius(14).Background(trackBg).Padding(10, 6, 8, 6).Children(func() {
		trackRow := ui.Row(c.Key("phen-track")).FillWidth().Gap(3).AlignItems(ui.End)
		rowRect := trackRow.LayoutBounds()
		idx := max(0, min(11, activeMonth-1))
		spotReady := interactive && st != nil && len(st.ws) == 12 && st.ws[idx] > 0 && st.hs[idx] > 0
		trackRow.Children(func() {
			if spotReady {
				animX := trackRow.AnimateTo("phen-spot-x", st.xs[idx], segmentSlideAnim)
				animW := trackRow.AnimateTo("phen-spot-w", st.ws[idx], segmentSlideAnim)
				spot := ui.Box(c.Key("phen-spot")).Absolute().Left(animX).Top(st.ys[idx]).Width(animW).Height(st.hs[idx]).
					Radius(10).Background(pillBg).PassThrough().ZIndex(-1)
				if !dark {
					spot.Shadow(0, 2, 8, 0, ui.RGBA(15, 23, 42, 0.10))
				}
			} else if interactive {
				c.AnimationFrame()
			}

			for m := 1; m <= 12; m++ {
				mm := m
				ph := cal[mm-1]
				selected := mm == activeMonth
				topCol, botCol, barH := seasonalPhenologyBarStyle(ph, dark)

				col := ui.Column(c.Key(fmt.Sprintf("phen-m-%d", mm))).Grow(1).Basis(0).
					Radius(10).Padding(5, 2, 5, 2).Gap(5).AlignItems(ui.Center)
				if interactive {
					col.Role(ui.RoleButton).Label(fmt.Sprintf("%d月 %s", mm, ph.Label()))
				}
				if selected && !spotReady {
					col.Background(pillBg)
					if !dark {
						col.Shadow(0, 2, 8, 0, ui.RGBA(15, 23, 42, 0.10))
					}
				}
				col.Children(func() {
					ui.Box(c).FillWidth().Height(30).AlignItems(ui.Center).Justify(ui.End).Children(func() {
						if ph == model.PhaseAbsent {
							ui.Box(c).Size(5, 5).Radius(2.5).Background(botCol).Margin(0, 0, 3, 0)
						} else {
							bar := ui.Box(c).Width(10).Height(barH).Radius(5).LinearGradient(ui.LinearGradient{
								From: topCol, To: botCol, Angle: 180,
							})
							if selected {
								bar.Shadow(0, 2, 6, 0, botCol.Alpha(0.32))
							}
						}
					})

					numWeight := 500
					numCol := mutedCol
					if selected {
						numWeight = 700
						numCol = textCol
					}
					ui.Textf(c, "%d", mm).FontSize(10.5).FontWeight(numWeight).TextColor(numCol).FontFeatures("tnum")
				})
				if interactive && st != nil {
					if len(st.xs) != 12 {
						st.xs = make([]float32, 12)
						st.ys = make([]float32, 12)
						st.ws = make([]float32, 12)
						st.hs = make([]float32, 12)
					}
					if r := col.LayoutBounds(); rowRect.W > 0 && r.W > 0 && r.H > 0 {
						st.xs[mm-1] = r.X - rowRect.X
						st.ys[mm-1] = r.Y - rowRect.Y
						st.ws[mm-1] = r.W
						st.hs[mm-1] = r.H
					}
				}
				if interactive && onSelect != nil && hapticClick(col) {
					onSelect(mm)
					c.AnimationFrame()
				}
			}
		})
	})

	// 柔和无边框圆点图例
	ui.Row(c).FillWidth().Gap(12).Wrap().AlignItems(ui.Center).Children(func() {
		for _, ph := range []model.SeasonalPhase{
			model.PhaseInSeason, model.PhaseJustArrived, model.PhaseLeaving, model.PhaseArrivingSoon,
		} {
			_, botCol, _ := seasonalPhenologyBarStyle(ph, dark)
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(8, 8).Radius(4).Background(botCol)
				ui.Text(c, ph.Label()).FontSize(11).TextColor(mutedCol)
			})
		}
	})
}

func seasonalTabToPhase(tab int) model.SeasonalPhase {
	switch tab {
	case 0:
		return model.PhaseLeaving
	case 1:
		return model.PhaseInSeason
	case 2:
		return model.PhaseJustArrived
	case 3:
		return model.PhaseArrivingSoon
	default:
		return model.PhaseInSeason
	}
}

func (a *AppState) seasonalRegionDisplayName() string {
	if strings.TrimSpace(a.seasonalRegionLabel) != "" {
		return a.seasonalRegionLabel
	}
	return regionShortNameByCode(fallbackText(a.seasonalRegionCode, a.store.Settings().DefaultRegion))
}

func (a *AppState) seasonalTaxonFor(b model.SeasonalBird) model.Taxon {
	if tx, ok := a.taxonByCode[b.SpeciesCode]; ok {
		return tx
	}
	return model.Taxon{
		SpeciesCode:   b.SpeciesCode,
		ComName:       b.ComName,
		SciName:       b.SciName,
		Order:         b.Order,
		FamilySciName: b.FamilySciName,
	}
}

func (a *AppState) activeMemoCount() int {
	n := 0
	for _, m := range a.store.Memos() {
		if !m.Completed {
			n++
		}
	}
	return n
}

func (a *AppState) toggleSeasonalBirdMemo(b model.SeasonalBird, zone model.EcoZone, month int, regionName string) {
	// 如果当月已有未完成备忘则移除，否则添加新追鸟备忘
	for _, m := range a.store.Memos() {
		if !m.Completed && m.SpeciesCode == b.SpeciesCode && m.TargetMonth == month {
			_ = a.store.DeleteMemo(m.ID)
			a.hapticSelection()
			a.showToast("已从追鸟备忘移除：" + b.ComName)
			return
		}
	}
	ph := b.PhaseFor(zone, month)
	_, err := a.store.AddMemo(model.BirdingMemo{
		SpeciesCode: b.SpeciesCode,
		ComName:     b.ComName,
		SciName:     b.SciName,
		RegionName:  regionName,
		TargetMonth: month,
		PhaseLabel:  ph.Label(),
		Location:    b.RecommendedSpots(zone),
		Notes:       b.CardSeasonBlurb(ph, month, regionName),
	})
	if err != nil {
		a.showToast(fmt.Sprintf("添加备忘失败: %v", err))
		return
	}
	a.hapticSuccess()
	a.showToast(fmt.Sprintf("已加入 %d 月追鸟备忘：%s", month, b.ComName))
}

func (a *AppState) openSeasonalBirdInGuide(b model.SeasonalBird) {
	if _, ok := a.taxonByCode[b.SpeciesCode]; !ok {
		tx := a.seasonalTaxonFor(b)
		a.taxonByCode[b.SpeciesCode] = tx
	}
	a.OpenSpeciesInGuide(b.SpeciesCode)
}

func (a *AppState) viewSeasonalHome(c *ui.Context, t *ui.Theme) {
	if a.seasonalMonth < 1 || a.seasonalMonth > 12 {
		a.seasonalMonth = int(time.Now().Month())
	}
	regionCode := fallbackText(a.seasonalRegionCode, a.store.Settings().DefaultRegion)
	regionName := a.seasonalRegionDisplayName()
	zone := model.RegionCodeToEcoZone(regionCode)
	grouped := api.SeasonalBirdsByRegionAndMonth(regionCode, a.seasonalMonth)
	activePhase := seasonalTabToPhase(a.seasonalTab)
	birds := grouped[activePhase]
	activeMemos := a.activeMemoCount()

	ui.Column(c).Fill().Children(func() {
		// 顶部标题、地区与追鸟备忘入口
		ui.Column(c).FillWidth().Padding(immersiveTopPad(c, 16), 18, 10, 18).Gap(10).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, "观鸟指南").FontSize(26).FontWeight(600).LetterSpacing(-0.1).Role(ui.RoleHeading).Level(1)

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					regBtn := applySculptedSecondaryPillStyle(
						ui.Row(c.Key("seasonal-region-btn")).Height(34).PaddingX(12).Radius(17).
							Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("切换观鸟指南地区"),
						t,
					)
					regBtn.Children(func() {
						ui.Icon(c, iconMapPin).FontSize(13).TextColor(opticalAccentColor(t))
						ui.Text(c, regionName).FontSize(13).FontWeight(500).SingleLine()
					})
					if hapticClick(regBtn) {
						a.regRegionTarget = "seasonal"
						a.regRegionQuery = ""
						a.regRegionSheetOpen = true
					}

					memoLabel := "备忘"
					if activeMemos > 0 {
						memoLabel = fmt.Sprintf("备忘 %d", activeMemos)
					}
					memoBtn := applySculptedSecondaryPillStyle(
						ui.Row(c.Key("seasonal-memo-btn")).Height(34).PaddingX(12).Radius(17).
							Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("打开追鸟备忘"),
						t,
					)
					memoBtn.Children(func() {
						ic := iconBookmark
						col := t.Text
						if activeMemos > 0 {
							ic = iconBookmarkFill
							col = opticalAccentColor(t)
						}
						ui.Icon(c, ic).FontSize(13).TextColor(col)
						ui.Text(c, memoLabel).FontSize(13).FontWeight(500).SingleLine()
					})
					if hapticClick(memoBtn) {
						a.seasonalMemoSheet = true
					}
				})
			})

			// 当月中文名与“观鸟指南”保持同号大字（26sp），右侧完整呈现区域与时令说明（不截断）
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Text(c, model.MonthZh(a.seasonalMonth)).
					FontSize(26).FontWeight(600).LetterSpacing(-0.2).TextColor(t.Text).Shrink(0)
				ui.Textf(c, "%s · %s", model.EcoZoneLabel(zone), model.MonthSeasonTheme(a.seasonalMonth)).
					FontSize(12.5).LineHeight(1.42).TextColor(secondaryTextColor(t)).Grow(1).Basis(0)
			})

			// 四大时令分类分段器：马上离开 / 当季 / 刚到 / 马上到
			if tactileSegmentedControl(c, "seasonal-phase-seg", &a.seasonalTab, t, "马上离开", "当季", "刚到", "马上到") {
				a.seasonalScroll.Y = 0
				a.seasonalList.ScrollIntoView(0)
			}
		})

		// 主体时令鸟种卡片列表（虚拟化按需渲染，与图鉴共享缩略图缓存并避免突发并发 429）
		rowCount := 0
		if len(birds) > 0 {
			rowCount = 1 + len(birds)
		}
		a.seasonalList.Key = func(r int) any {
			if r == 0 {
				return fmt.Sprintf("sph-%s-%d-%d", regionCode, a.seasonalMonth, a.seasonalTab)
			}
			idx := r - 1
			if idx >= 0 && idx < len(birds) {
				return "sb-" + birds[idx].SpeciesCode
			}
			return r
		}
		a.seasonalList.Label = nil

		ui.Box(c).Grow(1).Basis(0).FillWidth().Children(func() {
			phaseBg, phaseFg := seasonalPhaseColors(activePhase, t.Dark)
			renderPhaseBanner := func(ctx *ui.Context) {
				ui.Row(ctx).FillWidth().Padding(8, 12).Radius(12).Background(phaseBg).
					Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Row(ctx).Gap(8).AlignItems(ui.Center).Grow(1).Basis(0).Children(func() {
						ui.Text(ctx, activePhase.Label()).FontSize(11.5).FontWeight(600).TextColor(phaseFg).Shrink(0)
						ui.Text(ctx, activePhase.ShortActionHint()).FontSize(12).TextColor(phaseFg).SingleLine().Grow(1)
					})
					ui.Textf(ctx, "%d 种", len(birds)).FontSize(11.5).FontWeight(600).TextColor(phaseFg).FontFeatures("tnum").Shrink(0)
				})
			}

			list := ui.List(c.Key("seasonal-list"), &a.seasonalList, rowCount).Fill().Scrollbars(false).TrackScroll(&a.seasonalScroll)
			list.Rows(func(row ui.ListRow) {
				rc := row.Context
				if row.Index == 0 {
					ui.Box(rc).FillWidth().Padding(8, 18, 10, 18).Children(func() {
						renderPhaseBanner(rc)
					})
					return
				}
				idx := row.Index - 1
				if idx < 0 || idx >= len(birds) {
					return
				}
				padBottom := float32(12)
				if row.Index == rowCount-1 {
					padBottom = dockClearancePad(c, 22)
				}
				ui.Box(rc).FillWidth().Padding(0, 18, padBottom, 18).Children(func() {
					a.buildSeasonalBirdCard(rc, t, idx, birds[idx], zone, activePhase, regionName)
				})
			}).Children(func() {
				if len(birds) == 0 {
					ui.Column(c).FillWidth().Padding(8, 18, dockClearancePad(c, 22), 18).Gap(12).Children(func() {
						renderPhaseBanner(c)
						tactileCard(c, t).Padding(28).Center().Gap(10).Children(func() {
							tactileRecessedStyle(ui.Box(c).Size(50, 50), t, 25).Center().Children(func() {
								ui.Icon(c, iconNavSeasonal).FontSize(22).TextColor(t.TextMuted)
							})
							ui.Textf(c, "%s%s暂无「%s」重点鸟种", model.MonthZh(a.seasonalMonth), regionName, activePhase.Label()).
								FontSize(15).FontWeight(600)
							ui.Text(c, "可以切换其他时令分组或顶部地区，查看周边重点迁徙鸟种。").
								FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center)
							if a.seasonalTab != 1 && len(grouped[model.PhaseInSeason]) > 0 {
								if hapticClick(filterPill(c, "seasonal-jump-inseason", fmt.Sprintf("查看当季 %d 种", len(grouped[model.PhaseInSeason])), true, t)) {
									a.seasonalTab = 1
									a.seasonalScroll.Y = 0
									a.seasonalList.ScrollIntoView(0)
								}
							}
						})
					})
				}
			})
			if len(birds) > 0 {
				scrollTopFadeScrim(c, t, a.seasonalScroll.Y)
			}
		})
	})
}

func (a *AppState) buildSeasonalBirdCard(c *ui.Context, t *ui.Theme, idx int, bird model.SeasonalBird, zone model.EcoZone, phase model.SeasonalPhase, regionName string) {
	tx := a.seasonalTaxonFor(bird)
	thumbURL := a.speciesThumbURL(bird.SpeciesCode)
	var thumbSrc ui.ImageSource
	if thumbURL != "" {
		var thumbErr error
		thumbSrc, thumbErr = netimage.Load(c, thumbURL, netimage.Options{MaxSide: speciesThumbMaxSide})
		if thumbSrc == nil && thumbErr != nil && strings.Contains(thumbURL, "/small.") {
			fbURL := strings.Replace(thumbURL, "/small.", "/medium.", 1)
			thumbSrc, _ = netimage.Load(c, fbURL, netimage.Options{MaxSide: speciesThumbMaxSide})
		}
	} else {
		a.queueSpeciesThumb(tx)
	}
	hasMemo := a.store.HasActiveMemo(bird.SpeciesCode, a.seasonalMonth)
	blurb := bird.CardSeasonBlurb(phase, a.seasonalMonth, regionName)
	spotHint := bird.RecommendedSpots(zone)

	card := tactileCard(c, t).Key(fmt.Sprintf("scard-%d-%s", idx, bird.SpeciesCode)).Padding(12).
		Role(ui.RoleButton).Label(bird.ComName + " 时令详情")
	card.Children(func() {
		ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Start).Children(func() {
			// 左侧高清鸟类缩略图（保持干净无遮挡，不重复叠加当前已选中的分类徽标）
			ui.Box(c).Size(98, 108).Radius(14).Clip().Shrink(0).Background(softPillColor(t)).Children(func() {
				if thumbSrc != nil {
					ui.Image(c.Key("simg-"+bird.SpeciesCode), thumbSrc).Fill().Fit(ui.Cover).Label(bird.ComName)
				} else {
					skeletonBirdCard(c, 0, 28, t)
				}
			})

			// 右侧时令介绍与快捷备忘
			ui.Column(c).Grow(1).Basis(0).Gap(5).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Column(c).Grow(1).Basis(0).Gap(1).Children(func() {
						ui.Text(c, bird.ComName).FontSize(16).FontWeight(600).SingleLine()
						ui.Text(c, bird.Habitats).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
					})

					bmIcon := iconBookmark
					bmBg := softPillColor(t)
					bmFg := t.TextMuted
					bmLabel := "加入追鸟备忘"
					if hasMemo {
						bmIcon = iconBookmarkFill
						bmBg = opticalAccentColor(t).Alpha(0.14)
						bmFg = opticalAccentColor(t)
						bmLabel = "取消追鸟备忘"
					}
					bmBtn := circleIconBtn(c, fmt.Sprintf("sbm-%s", bird.SpeciesCode), bmLabel, bmIcon, bmBg, bmFg, 30)
					if hapticClick(bmBtn) {
						a.toggleSeasonalBirdMemo(bird, zone, a.seasonalMonth, regionName)
					}
				})

				ui.Text(c, blurb).FontSize(13).LineHeight(1.48).TextColor(secondaryTextColor(t)).MaxLines(2)

				ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconMapPin).FontSize(11).TextColor(opticalAccentColor(t)).Shrink(0)
					ui.Text(c, spotHint).FontSize(11.5).TextColor(t.TextMuted).SingleLine().Grow(1)
				})
			})
		})
	})
	if hapticClick(card) {
		a.ensureGuideMediaLoaded(tx)
		a.seasonalDetailMonth = a.seasonalMonth
		a.router.Push("/seasonal/" + bird.SpeciesCode)
	}
}

func (a *AppState) viewSeasonalDetailPage(c *ui.Context, t *ui.Theme, code string) {
	bird, ok := api.FindSeasonalBirdByCode(code)
	if !ok {
		ui.Column(c).Fill().Center().Padding(24).Gap(12).Children(func() {
			ui.Text(c, "未找到该鸟种的时令指南档案")
			if hapticClick(filterPill(c, "sd-nf-back", "返回观鸟指南", false, t)) {
				a.goBack()
			}
		})
		return
	}

	if a.seasonalMonth < 1 || a.seasonalMonth > 12 {
		a.seasonalMonth = int(time.Now().Month())
	}
	detailMonth := a.seasonalDetailMonth
	if detailMonth < 1 || detailMonth > 12 {
		detailMonth = a.seasonalMonth
	}
	tx := a.seasonalTaxonFor(bird)
	a.ensureGuideMediaLoaded(tx)

	regionCode := fallbackText(a.seasonalRegionCode, a.store.Settings().DefaultRegion)
	regionName := a.seasonalRegionDisplayName()
	zone := model.RegionCodeToEcoZone(regionCode)
	phase := bird.PhaseFor(zone, detailMonth)
	cal := bird.MonthCalendar(zone)
	hasMemo := a.store.HasActiveMemo(bird.SpeciesCode, detailMonth)
	transitioning := a.router.Transitioning()
	top, _, bottom, _ := c.SafeArea()

	// 解析头图（优先高清媒体图，其次缩略图）
	var heroSrc ui.ImageSource
	if m := a.guideMediaMap[code]; m != nil && len(m.Photos) > 0 {
		heroSrc, _ = netimage.Load(c, m.Photos[0].URL, netimage.Options{MaxSide: 1080})
		if heroSrc == nil && m.Photos[0].ThumbURL != "" {
			heroSrc, _ = netimage.Load(c, m.Photos[0].ThumbURL, netimage.Options{MaxSide: 480})
		}
	}
	if heroSrc == nil {
		if u := a.speciesThumbURL(code); u != "" {
			heroSrc, _ = netimage.Load(c, u, netimage.Options{MaxSide: 480})
		} else {
			a.queueSpeciesThumb(tx)
		}
	}

	ui.Scroll(c.Key("sd-scroll-"+code)).Fill().Scrollbars(false).Children(func() {
		ui.Column(c).FillWidth().Padding(0, 0, max(float32(36), bottom+28), 0).Children(func() {
			// 顶部全出血 Hero 头图
			ui.Box(c).FillWidth().Height(296 + top*0.35).Background(softPillColor(t)).Clip().Children(func() {
				if heroSrc != nil {
					ui.Image(c, heroSrc).Fill().Fit(ui.Cover).Label(bird.ComName)
				} else {
					skeletonBirdCard(c, 0, 44, t)
				}

				// 顶部柔暗护栏，保证返回按钮与状态栏清晰
				ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(84 + top).PassThrough().
					LinearGradient(ui.LinearGradient{
						From: ui.RGBA(0, 0, 0, 0.50), To: ui.Transparent, Angle: 180,
					})
				// 底部柔和过渡暗部
				ui.Box(c).Absolute().Left(0).Bottom(0).Right(0).Height(72).PassThrough().
					LinearGradient(ui.LinearGradient{
						From: ui.Transparent, To: ui.RGBA(0, 0, 0, 0.42), Angle: 180,
					})

				// 顶部导航悬浮按钮（转场期间自动避开离屏模糊以保持 120fps）
				ui.Row(c).Absolute().Left(16).Right(16).Top(immersiveTopPad(c, 14)).
					Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					if hapticClick(liquidGlassCircleBtnFast(c, "sd-back", "返回观鸟指南", iconBack, 40, transitioning)) {
						a.goBack()
					}
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if hapticClick(liquidGlassCircleBtnFast(c, "sd-share", "分享时令卡片", iconShare, 40, transitioning)) {
							a.ShareSeasonalBirdImageAsync(bird, zone, detailMonth, regionName)
						}
						bmIc := iconBookmark
						if hasMemo {
							bmIc = iconBookmarkFill
						}
						if hapticClick(liquidGlassCircleBtnFast(c, "sd-bm-top", "加入追鸟备忘", bmIc, 40, transitioning)) {
							a.toggleSeasonalBirdMemo(bird, zone, detailMonth, regionName)
						}
					})
				})

				// 左下角生态区 + 月份小标
				applyPhotoOverlayGlass(
					ui.Row(c).Absolute().Left(18).Bottom(32).Padding(5, 11).Radius(999).Gap(5).AlignItems(ui.Center),
					transitioning, false,
				).Children(func() {
					ui.Icon(c, iconNavSeasonal).FontSize(12).TextColor(ui.RGB(255, 255, 255))
					ui.Textf(c, "%s · %s（%s）", model.MonthZh(detailMonth), regionName, model.EcoZoneLabel(zone)).
						FontSize(11.5).FontWeight(500).TextColor(ui.RGB(255, 255, 255))
				})
			})

			// 圆角内容卡叠层
			ui.Column(c).FillWidth().Margin(-22, 0, 0, 0).Radius(24, 24, 0, 0).
				Background(t.Background).Padding(20, 18, 12, 18).Gap(14).Children(func() {
				// 标题 + 状态徽章 + 快捷操作按钮行
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(10).Children(func() {
						ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
							ui.Text(c, bird.ComName).FontSize(26).FontWeight(700).LetterSpacing(-0.2)
							ui.Text(c, bird.SciName).FontSize(13.5).Italic().TextColor(t.TextMuted)
						})
						bg, fg := seasonalPhaseColors(phase, t.Dark)
						statusPill(c, phase.Label(), bg, fg)
					})

					ui.Text(c, bird.CardSeasonBlurb(phase, detailMonth, regionName)).
						FontSize(14).LineHeight(1.62).TextColor(secondaryTextColor(t))

					// 追鸟备忘 + 定日历 双主操作
					ui.Row(c).FillWidth().Gap(10).Padding(4, 0, 0, 0).Children(func() {
						memoText := "加入追鸟备忘"
						if hasMemo {
							memoText = "已加追鸟备忘 ✓"
						}
						memoBtn := applySculptedSecondaryPillStyle(
							ui.Row(c.Key("sd-memo-btn")).Grow(1).Basis(0).Height(42).Radius(21).
								Gap(6).AlignItems(ui.Center).Justify(ui.Center).
								Role(ui.RoleButton).Label(memoText),
							t,
						)
						memoBtn.Children(func() {
							ic := iconBookmark
							col := t.Text
							if hasMemo {
								ic = iconBookmarkFill
								col = opticalAccentColor(t)
							}
							ui.Icon(c, ic).FontSize(15).TextColor(col)
							ui.Text(c, memoText).FontSize(13.5).FontWeight(500).TextColor(col)
						})
						if hapticClick(memoBtn) {
							a.toggleSeasonalBirdMemo(bird, zone, detailMonth, regionName)
						}

						calBtn := applySculptedSecondaryPillStyle(
							ui.Row(c.Key("sd-cal-btn")).Grow(1).Basis(0).Height(42).Radius(21).
								Gap(6).AlignItems(ui.Center).Justify(ui.Center).
								Role(ui.RoleButton).Label("添加到手机日历"),
							t,
						)
						calBtn.Children(func() {
							ui.Icon(c, iconCalendar).FontSize(15).TextColor(opticalAccentColor(t))
							ui.Text(c, "添加到日历").FontSize(13.5).FontWeight(500).TextColor(t.Text)
						})
						if hapticClick(calBtn) {
							a.AddSeasonalToCalendar(bird, zone, detailMonth, regionName)
						}
					})
				})

				// 卡片 1：全年 1-12 月柔和无边框物候月历（灵感源自 Merlin Bird ID 与 Apple Weather）
				tactileCard(c, t).Padding(16).Gap(10).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "全年物候月历").FontSize(15).FontWeight(600)
						ui.Textf(c, "%s地区 · 点击切换月份", model.EcoZoneLabel(zone)).FontSize(11.5).TextColor(t.TextMuted)
					})

					renderSeasonalPhenologyChart(c, cal, detailMonth, t.Dark, true, func(mm int) {
						a.seasonalDetailMonth = mm
					})

					if bird.MigrationTimeline != "" {
						minimalDivider(c, t)
						ui.Text(c, bird.MigrationTimeline).FontSize(13.5).LineHeight(1.65).TextColor(secondaryTextColor(t))
					}
				})

				// 卡片 2：不同时间在哪里看 & 推荐生境
				tactileCard(c, t).Padding(16).Gap(10).Children(func() {
					ui.Text(c, "在哪里看 · 推荐点位").FontSize(15).FontWeight(600)
					ui.Column(c).FillWidth().Gap(6).Children(func() {
						ui.Textf(c, "%s重点推荐", regionName).FontSize(12).FontWeight(600).TextColor(opticalAccentColor(t))
						ui.Text(c, bird.RecommendedSpots(zone)).FontSize(14).LineHeight(1.6).TextColor(t.Text)
					})
					if bird.NationalSpots != "" && bird.NationalSpots != bird.RecommendedSpots(zone) {
						ui.Column(c).FillWidth().Gap(4).Children(func() {
							ui.Text(c, "全国经典观测地").FontSize(12).FontWeight(500).TextColor(t.TextMuted)
							ui.Text(c, bird.NationalSpots).FontSize(13).LineHeight(1.55).TextColor(secondaryTextColor(t))
						})
					}
					minimalDivider(c, t)
					metaAnnotationRow(c, t,
						metaItem{Text: bird.Habitats},
					)
					if bird.BestTimeOfDay != "" {
						ui.Row(c).FillWidth().Padding(8, 11).Radius(10).Background(softPillColor(t)).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconCalendar).FontSize(13).TextColor(opticalAccentColor(t)).Shrink(0)
							ui.Text(c, "最佳时段："+bird.BestTimeOfDay).FontSize(12.5).TextColor(secondaryTextColor(t)).Grow(1)
						})
					}
				})

				// 卡片 3：羽色与行为的季节变化
				tactileCard(c, t).Padding(16).Gap(10).Children(func() {
					ui.Text(c, "羽色与季节变化").FontSize(15).FontWeight(600)
					if bird.PlumageChanges != "" {
						ui.Column(c).FillWidth().Gap(4).Children(func() {
							ui.Text(c, "羽色 / 形态演变").FontSize(12).FontWeight(600).TextColor(opticalAccentColor(t))
							ui.Text(c, bird.PlumageChanges).FontSize(14).LineHeight(1.65).TextColor(secondaryTextColor(t))
						})
					}
					if bird.BehaviorChanges != "" {
						minimalDivider(c, t)
						ui.Column(c).FillWidth().Gap(4).Children(func() {
							ui.Text(c, "季节习性与行为").FontSize(12).FontWeight(600).TextColor(opticalAccentColor(t))
							ui.Text(c, bird.BehaviorChanges).FontSize(14).LineHeight(1.65).TextColor(secondaryTextColor(t))
						})
					}
				})

				// 卡片 4：野外实操贴士
				if bird.FieldTips != "" {
					tactileCard(c, t).Padding(16).Gap(6).Children(func() {
						ui.Text(c, "观鸟实操贴士").FontSize(15).FontWeight(600)
						ui.Text(c, bird.FieldTips).FontSize(14).LineHeight(1.65).TextColor(secondaryTextColor(t))
					})
				}

				// 底部主按钮：直达鸟类图鉴（查看完整多图、鸣声参考、维基百科与记录日志）
				ui.Column(c).FillWidth().Padding(4, 0, 6, 0).Gap(8).Children(func() {
					if hapticClick(tactilePrimaryBtn(c, "sd-goto-guide", "进入鸟类图鉴查看鸣声与百科 ↗", t).FillWidth()) {
						a.openSeasonalBirdInGuide(bird)
					}
				})
			})
		})
	})
}

func (a *AppState) viewSeasonalMemoSheet(c *ui.Context, t *ui.Theme) {
	sheetBg := t.Background
	mobile.Sheet(c, &a.seasonalMemoSheet, mobile.SheetOptions{
		Detents:    []float32{0.72, 0.92},
		Background: &sheetBg,
		Label:      "追鸟备忘",
	}, func() {
		memos := a.store.Memos()
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 20, 14, 20).Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "追鸟备忘").FontSize(16).FontWeight(600)
					ui.Textf(c, "共 %d 条观鸟计划", len(memos)).FontSize(12).TextColor(t.TextMuted)
				})
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if len(memos) > 0 {
						if hapticClick(filterPill(c, "memo-share-img", "分享图片", true, t)) {
							a.seasonalMemoSheet = false
							a.ShareMemosImageAsync()
						}
					}
					if hapticClick(filterPill(c, "memo-close", "完成", false, t)) {
						a.seasonalMemoSheet = false
					}
				})
			})
			minimalDivider(c, t)

			ui.Scroll(c.Key("memo-sheet-scroll")).Grow(1).Basis(0).FillWidth().Scrollbars(false).Children(func() {
				ui.Column(c).FillWidth().Padding(12, 18, 36, 18).Gap(10).Children(func() {
					if len(memos) == 0 {
						tactileCard(c, t).Padding(28).Center().Gap(10).Children(func() {
							tactileRecessedStyle(ui.Box(c).Size(48, 48), t, 24).Center().Children(func() {
								ui.Icon(c, iconBookmark).FontSize(20).TextColor(t.TextMuted)
							})
							ui.Text(c, "还没有追鸟备忘").FontSize(15).FontWeight(600)
							ui.Text(c, "在观鸟指南卡片点一下书签，即可收藏心仪目标并同步到手机日历或生成清单图片分享。").
								FontSize(12.5).TextColor(t.TextMuted).TextAlign(ui.Center).LineHeight(1.5)
						})
						return
					}

					for i, item := range memos {
						m := item
						tactileCard(c, t).Key("memo-row-"+m.ID).Padding(14).Gap(10).Children(func() {
							// 不显示头像占位圈，正文整体齐左排版
							body := ui.Column(c.Key(fmt.Sprintf("memo-body-%d", i))).FillWidth().Gap(4).
								Role(ui.RoleButton).Label("查看" + m.ComName + "时令详情")
							body.Children(func() {
								ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Baseline).Gap(8).Children(func() {
									nameCol := t.Text
									if m.Completed {
										nameCol = t.TextMuted
									}
									ui.Text(c, m.ComName).FontSize(15.5).FontWeight(600).TextColor(nameCol).SingleLine().Grow(1)
									ui.Text(c, fallbackText(m.TargetDate, model.MonthZh(m.TargetMonth))).
										FontSize(12).FontWeight(500).TextColor(opticalAccentColor(t)).FontFeatures("tnum").Shrink(0)
								})
								ui.Text(c, fallbackText(m.Location, m.RegionName)).
									FontSize(12).TextColor(t.TextMuted).MaxLines(2)
								if strings.TrimSpace(m.Notes) != "" {
									ui.Text(c, m.Notes).FontSize(12.5).LineHeight(1.45).TextColor(secondaryTextColor(t)).MaxLines(2)
								}
							})
							if hapticClick(body) {
								a.seasonalMemoSheet = false
								if m.TargetMonth >= 1 && m.TargetMonth <= 12 {
									a.seasonalDetailMonth = m.TargetMonth
								}
								a.router.Push("/seasonal/" + m.SpeciesCode)
							}

							ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
								if m.PhaseLabel != "" {
									ui.Textf(c, "%s · %s计划", m.PhaseLabel, model.MonthZh(m.TargetMonth)).FontSize(11.5).TextColor(t.TextMuted)
								} else {
									ui.Box(c)
								}
								ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
									doneText := "标为已见"
									if m.Completed {
										doneText = "已见 ✓"
									}
									chkPill := applySculptedSecondaryPillStyle(
										ui.Row(c.Key(fmt.Sprintf("memo-chk-%d", i))).Height(28).PaddingX(10).Radius(14).
											AlignItems(ui.Center).Role(ui.RoleButton).Label(doneText),
										t,
									)
									chkPill.Children(func() {
										fg := t.TextMuted
										if m.Completed {
											fg = t.Success
										}
										ui.Text(c, doneText).FontSize(11.5).FontWeight(500).TextColor(fg)
									})
									if hapticClick(chkPill) {
										_ = a.store.ToggleMemoDone(m.ID)
									}

									calBtn := applySculptedSecondaryPillStyle(
										ui.Row(c.Key(fmt.Sprintf("memo-cal-%d", i))).Height(28).PaddingX(10).Radius(14).
											Gap(4).AlignItems(ui.Center).Role(ui.RoleButton).Label("定日历"),
										t,
									)
									calBtn.Children(func() {
										ui.Icon(c, iconCalendar).FontSize(11).TextColor(opticalAccentColor(t))
										ui.Text(c, "定日历").FontSize(11.5).FontWeight(500)
									})
									if hapticClick(calBtn) {
										a.AddMemoToCalendar(m)
									}

									if hapticClick(circleIconBtn(c, fmt.Sprintf("memo-del-%d", i), "删除备忘", iconTrash, t.Danger.Alpha(0.12), t.Danger, 28)) {
										_ = a.store.DeleteMemo(m.ID)
									}
								})
							})
						})
					}
				})
			})
		})
	})
}

// AddSeasonalToCalendar 将单鸟时令观赏计划添加到系统日历（Android 调起原生 CalendarContract 编辑页，桌面导出 .ics）。
func (a *AppState) AddSeasonalToCalendar(bird model.SeasonalBird, zone model.EcoZone, month int, regionName string) {
	phase := bird.PhaseFor(zone, month)
	title := fmt.Sprintf("观鸟计划：%s（%s）", bird.ComName, phase.Label())
	loc := bird.RecommendedSpots(zone)
	desc := fmt.Sprintf("Birdview %d月%s观鸟指南\n状态：%s\n推荐点位：%s\n最佳时段：%s\n羽色变化：%s",
		month, regionName, phase.Label(), loc, bird.BestTimeOfDay, bird.PlumageChanges)
	start := pickTargetMorningTime("", month)
	a.openCalendarEvent(title, loc, desc, "birdview_"+bird.SpeciesCode+".ics", start)
}

// AddMemoToCalendar 将一条追鸟备忘添加到系统日历。
func (a *AppState) AddMemoToCalendar(m model.BirdingMemo) {
	title := fmt.Sprintf("追鸟计划：%s", m.ComName)
	if m.PhaseLabel != "" {
		title += "（" + m.PhaseLabel + "）"
	}
	loc := fallbackText(m.Location, m.RegionName)
	desc := fmt.Sprintf("Birdview 追鸟备忘\n目标地区：%s\n备注：%s", m.RegionName, m.Notes)
	start := pickTargetMorningTime(m.TargetDate, m.TargetMonth)
	a.openCalendarEvent(title, loc, desc, "birdview_memo_"+m.SpeciesCode+".ics", start)
}

func pickTargetMorningTime(targetDate string, month int) time.Time {
	if strings.TrimSpace(targetDate) != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(targetDate), time.Local); err == nil {
			return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 7, 30, 0, 0, time.Local)
		}
	}
	now := time.Now()
	if month < 1 || month > 12 {
		t := now.AddDate(0, 0, 1)
		return time.Date(t.Year(), t.Month(), t.Day(), 7, 30, 0, 0, time.Local)
	}
	year := now.Year()
	if month < int(now.Month()) {
		year++
		return time.Date(year, time.Month(month), 10, 7, 30, 0, 0, time.Local)
	}
	if month > int(now.Month()) {
		return time.Date(year, time.Month(month), 8, 7, 30, 0, 0, time.Local)
	}
	for i := 1; i <= 7; i++ {
		cand := now.AddDate(0, 0, i)
		if cand.Weekday() == time.Saturday && int(cand.Month()) == month {
			return time.Date(cand.Year(), cand.Month(), cand.Day(), 7, 30, 0, 0, time.Local)
		}
	}
	t := now.AddDate(0, 0, 1)
	return time.Date(t.Year(), t.Month(), t.Day(), 7, 30, 0, 0, time.Local)
}

func (a *AppState) openCalendarEvent(title, location, desc, defaultICSName string, start time.Time) {
	end := start.Add(150 * time.Minute)
	if runtime.GOOS == "android" {
		q := url.Values{}
		q.Set("title", title)
		q.Set("location", location)
		q.Set("desc", desc)
		q.Set("begin", fmt.Sprintf("%d", start.UnixMilli()))
		q.Set("end", fmt.Sprintf("%d", end.UnixMilli()))
		calURI := "birdview-calendar://add?" + q.Encode()
		if err := mygo.Shell.OpenExternal(calURI); err == nil {
			a.hapticSuccess()
			a.showToast("已打开系统日历创建观鸟日程")
			return
		}
	}

	ics := store.BuildCalendarICS(title, location, desc, start, 150*time.Minute)
	go func() {
		dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      a.win,
			Title:       "保存日历日程 (.ics)",
			DefaultPath: defaultICSName,
		})
		if err != nil || dest == "" {
			return
		}
		if werr := os.WriteFile(dest, []byte(ics), 0o644); werr != nil {
			a.updateUI(func() { a.showToast(fmt.Sprintf("保存日历文件失败: %v", werr)) })
			return
		}
		a.updateUI(func() {
			a.hapticSuccess()
			a.showToast("已导出日历日程 " + filepath.Base(dest))
		})
	}()
}
