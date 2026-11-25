package appui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/imagepicker"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"
	"github.com/egoist/mygo/ui/netimage"
)

func formatSpeciesOption(tx model.Taxon) string {
	return fmt.Sprintf("%s (%s) [%s]", tx.ComName, tx.SciName, tx.SpeciesCode)
}

func (a *AppState) rebuildJournalSpeciesOptions() {
	opts := make([]string, 0, 2000)
	seen := make(map[string]bool)
	if len(a.chinaCodes) > 0 {
		for _, tx := range a.allTaxa {
			if a.chinaCodes[tx.SpeciesCode] {
				opts = append(opts, formatSpeciesOption(tx))
				seen[tx.SpeciesCode] = true
			}
		}
	}
	for _, tx := range a.allTaxa {
		if seen[tx.SpeciesCode] {
			continue
		}
		if len(opts) >= 2000 {
			break
		}
		opts = append(opts, formatSpeciesOption(tx))
		seen[tx.SpeciesCode] = true
	}
	a.journalSpeciesOpts = opts
}

func (a *AppState) speciesSuggestions(query string) []model.Taxon {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	out := make([]model.Taxon, 0, 6)
	for _, tx := range a.allTaxa {
		if strings.Contains(strings.ToLower(tx.ComName), q) ||
			strings.Contains(strings.ToLower(tx.SciName), q) ||
			strings.Contains(strings.ToLower(tx.SpeciesCode), q) {
			out = append(out, tx)
			if len(out) >= 6 {
				break
			}
		}
	}
	return out
}

func (a *AppState) resolveDraftSpecies() (code, comName, sciName, order, family string) {
	raw := strings.TrimSpace(a.jDraftSpecies)
	if raw == "" {
		return "", "", "", "", ""
	}
	if lb := strings.LastIndex(raw, "["); lb >= 0 && strings.HasSuffix(raw, "]") {
		c := strings.TrimSpace(raw[lb+1 : len(raw)-1])
		if tx, ok := a.taxonByCode[c]; ok {
			return tx.SpeciesCode, tx.ComName, tx.SciName, model.OrderZh(tx.Order), model.FamilyZh(tx.FamilySciName, tx.FamilyComName)
		}
	}
	lower := strings.ToLower(raw)
	for _, tx := range a.allTaxa {
		if strings.EqualFold(tx.SpeciesCode, raw) || strings.EqualFold(tx.ComName, raw) || strings.EqualFold(tx.SciName, raw) {
			return tx.SpeciesCode, tx.ComName, tx.SciName, model.OrderZh(tx.Order), model.FamilyZh(tx.FamilySciName, tx.FamilyComName)
		}
	}
	for _, tx := range a.allTaxa {
		if strings.Contains(strings.ToLower(tx.ComName), lower) || strings.Contains(strings.ToLower(tx.SciName), lower) {
			return tx.SpeciesCode, tx.ComName, tx.SciName, model.OrderZh(tx.Order), model.FamilyZh(tx.FamilySciName, tx.FamilyComName)
		}
	}
	slug := strings.ReplaceAll(lower, " ", "_")
	return "custom_" + slug, raw, raw, "", ""
}

// OpenNewJournalModal 打开新建观鸟日志底部抽屉并预填鸟种与地点。
func (a *AppState) OpenNewJournalModal(speciesCode, location string) {
	a.jEditID = ""
	a.jDraftSpecies = ""
	if speciesCode != "" {
		if tx, ok := a.taxonByCode[speciesCode]; ok {
			a.jDraftSpecies = formatSpeciesOption(tx)
		} else {
			a.jDraftSpecies = speciesCode
		}
	}
	a.jDraftObservedAt = time.Now().Format("2006-01-02 15:04")
	a.jDraftLocation = location
	a.jDraftLat = 0
	a.jDraftLng = 0
	a.jDraftCount = 1
	a.jDraftNotes = ""
	a.jDraftPhotos = nil
	a.jDraftAudio = ""
	a.jDraftError = ""
	a.journalSheetOpen = true
	a.invalidateUI()
}

func (a *AppState) openEditJournalModal(e model.JournalEntry) {
	a.jEditID = e.ID
	if tx, ok := a.taxonByCode[e.SpeciesCode]; ok {
		a.jDraftSpecies = formatSpeciesOption(tx)
	} else {
		a.jDraftSpecies = e.ComName
	}
	a.jDraftObservedAt = e.ObservedAt
	a.jDraftLocation = e.Location
	a.jDraftLat = e.Lat
	a.jDraftLng = e.Lng
	a.jDraftCount = max(1, e.Count)
	a.jDraftNotes = e.Notes
	a.jDraftPhotos = append([]string(nil), e.PhotoPaths...)
	a.jDraftAudio = e.AudioPath
	a.jDraftError = ""
	a.journalSheetOpen = true
}

func (a *AppState) refreshJournalList() {
	entries := a.store.Entries()
	q := strings.ToLower(strings.TrimSpace(a.session.JournalQuery))
	out := make([]model.JournalEntry, 0, len(entries))
	for _, e := range entries {
		if q != "" {
			match := strings.Contains(strings.ToLower(e.ComName), q) ||
				strings.Contains(strings.ToLower(e.SciName), q) ||
				strings.Contains(strings.ToLower(e.Location), q) ||
				strings.Contains(strings.ToLower(e.Notes), q)
			if !match {
				continue
			}
		}
		out = append(out, e)
	}
	a.filteredEntries = out
}

func (a *AppState) viewJournalHome(c *ui.Context, t *ui.Theme) {
	unlocked := a.store.UnlockedBirds()
	entries := a.store.Entries()
	chinaTotal := len(a.chinaCodes)
	if chinaTotal == 0 {
		chinaTotal = 1469
	}
	progress := float32(len(unlocked)) / float32(chinaTotal)
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}
	familyCount := make(map[string]int)
	for _, u := range unlocked {
		familyCount[fallbackText(u.FamilyName, "未分类")]++
	}
	optA, optB := opticalAccentColors(t)

	ui.Box(c).Fill().Children(func() {
		ui.Column(c).Fill().Children(func() {
			ui.Column(c).FillWidth().Padding(16, 18, 10, 18).Gap(12).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "日志").FontSize(26).FontWeight(600).LetterSpacing(-0.1).Role(ui.RoleHeading).Level(1)

					shareBtn := applySculptedSecondaryPillStyle(
						ui.Row(c.Key("j-export-btn")).Height(34).PaddingX(14).Radius(17).
							Gap(5).AlignItems(ui.Center).
							Role(ui.RoleButton).Label("分享或导出"),
						t,
					)
					shareBtn.Children(func() {
						ui.Icon(c, iconShare).FontSize(13).TextColor(t.Text)
						ui.Text(c, "导出").FontSize(13).FontWeight(500)
					})
					if shareBtn.Clicked() {
						a.exportSheetOpen = true
					}
				})

				// 立体解锁成就卡片 + 内凹进度槽与光学微渐变光条
				tactileCard(c, t).Padding(16).Gap(10).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Baseline).Children(func() {
						ui.Row(c).Gap(6).AlignItems(ui.Baseline).Children(func() {
							ui.Textf(c, "%d", len(unlocked)).FontSize(28).FontWeight(600).TextColor(t.Text).FontFeatures("tnum")
							ui.Textf(c, "/ %d 种已解锁", chinaTotal).FontSize(13).TextColor(t.TextMuted).FontFeatures("tnum")
						})
						ui.Textf(c, "%d 条日志 · %d 科", len(entries), len(familyCount)).FontSize(12).TextColor(secondaryTextColor(t))
					})
					tactileRecessedStyle(ui.Box(c).FillWidth().Height(7).Clip(), t, 3.5).Children(func() {
						if progress > 0 {
							pct := progress * 100
							if pct < 2.5 {
								pct = 2.5
							}
							ui.Box(c).FillHeight().WidthPercent(pct).Radius(3.5).
								LinearGradient(ui.LinearGradient{From: optA, To: optB, Angle: 90})
						}
					})
				})

				tactileSegmentedControl(c, "j-subtab", &a.session.JournalSubTab, t, "时间线日志", "已解锁图谱")
			})

			if a.session.JournalSubTab == 0 {
				a.buildJournalTimeline(c, t)
			} else {
				a.buildLifelistGrid(c, t, unlocked)
			}
		})

		// 右下角立体浮雕圆形主按钮（避开底部 Liquid Glass 悬空导航栏）
		if !a.journalSheetOpen {
			_, _, bottom, _ := c.SafeArea()
			fabBottom := max(float32(88), bottom+82)
			fab := applySculptedPrimaryStyle(
				ui.Box(c.Key("j-fab")).Size(54, 54).Radius(27).Center().
					Absolute().Right(18).Bottom(fabBottom).
					Role(ui.RoleButton).Label("新建观鸟日志"),
				t,
			)
			fab.Children(func() {
				ui.Icon(c, iconPlus).FontSize(22).TextColor(t.AccentText)
			})
			if fab.Clicked() {
				a.OpenNewJournalModal("", "")
			}
		}
	})
}

func (a *AppState) buildJournalTimeline(c *ui.Context, t *ui.Theme) {
	ui.Box(c).Padding(2, 18, 8, 18).Children(func() {
		if minimalSearchBar(c, "j-search", &a.session.JournalQuery, "搜索鸟种、地点或笔记", "搜索日志", t) {
			a.refreshJournalList()
		}
	})

	scroll := ui.Scroll(c.Key("j-timeline-scroll")).Grow(1).Basis(0).FillWidth()
	if scroll.PullToRefresh(&a.journalRefresh) {
		a.refreshJournalList()
		a.journalRefresh.Refreshing = false
	}
	scroll.Children(func() {
		ui.Column(c).FillWidth().Padding(4, 18, 112, 18).Gap(10).Children(func() {
			if len(a.filteredEntries) == 0 {
				tactileCard(c, t).Padding(34).Center().Gap(10).Children(func() {
					tactileRecessedStyle(ui.Box(c).Size(52, 52), t, 26).Center().Children(func() {
						ui.Icon(c, iconJournal).FontSize(22).TextColor(t.TextMuted)
					})
					ui.Text(c, "暂无观鸟日志").FontSize(16).FontWeight(600)
					ui.Text(c, "点击右下角 + 记录今天的目击并点亮图鉴").FontSize(13).TextColor(t.TextMuted)
				})
				return
			}

			for i, item := range a.filteredEntries {
				entry := item
				tactileCard(c, t).Key("jsw-"+entry.ID).Clip().Children(func() {
					mobile.SwipeRow(c, nil, mobile.SwipeOptions{
						Trailing: []mobile.SwipeAction{{
							Label:       "删除",
							Destructive: true,
							OnTap: func() {
								a.jDeleteTargetID = entry.ID
								a.jDeleteConfirm = true
							},
						}},
					}, func() {
						row := ui.Row(c).FillWidth().Padding(14, 16).Gap(12).AlignItems(ui.Center).
							Role(ui.RoleButton).Label(entry.ComName + " 日志详情")
						row.Children(func() {
							if len(entry.PhotoPaths) > 0 {
								p := entry.PhotoPaths[0]
								a.loadLocalImageAsync(p)
								if src, ok := a.localImgCache[p]; ok {
									ui.Image(c, src).Size(62, 62).Fit(ui.Cover).Radius(12).Shrink(0).Label(entry.ComName)
								} else {
									tactileRecessedStyle(ui.Box(c).Size(62, 62), t, 12).Center().Shrink(0).Children(func() {
										ui.Icon(c, iconImage).FontSize(20).TextColor(t.TextMuted)
									})
								}
							}

							ui.Column(c).Grow(1).Basis(0).Gap(3).Children(func() {
								ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Baseline).Gap(8).Children(func() {
									ui.Text(c, entry.ComName).FontSize(15).FontWeight(600).SingleLine().Grow(1)
									statusPill(c, fmt.Sprintf("%d 只", max(1, entry.Count)), softPillColor(t), secondaryTextColor(t))
								})
								ui.Textf(c, "%s · %s", model.FormatObsTime(entry.ObservedAt), fallbackText(entry.Location, "野外地点")).
									FontSize(12).TextColor(t.TextMuted).SingleLine()

								if strings.TrimSpace(entry.Notes) != "" {
									ui.Text(c, entry.Notes).FontSize(13).LineHeight(1.5).TextColor(secondaryTextColor(t)).MaxLines(2)
								}

								if entry.AudioPath != "" || len(entry.PhotoPaths) > 1 {
									ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(2, 0, 0, 0).Children(func() {
										if entry.AudioPath != "" {
											pst := a.player.Status()
											isThis := pst.SourceKey == entry.AudioPath
											label := "现场录音"
											if isThis && pst.Playing {
												label = "暂停录音"
											}
											playPill := ui.Row(c.Key(fmt.Sprintf("jt-aud-%d", i))).Padding(4, 10).Radius(999).
												Background(opticalAccentColor(t).Alpha(0.12)).Gap(4).AlignItems(ui.Center).Role(ui.RoleButton).Label(label)
											playPill.Children(func() {
												ic := iconPlay
												if isThis && pst.Playing {
													ic = iconPause
												}
												ui.Icon(c, ic).FontSize(10).TextColor(opticalAccentColor(t))
												ui.Text(c, label).FontSize(11).FontWeight(500).TextColor(opticalAccentColor(t))
											})
											if playPill.Clicked() {
												if isThis && pst.Active {
													a.player.TogglePause()
												} else {
													a.player.PlayFile(entry.AudioPath, entry.ComName+" · 现场录音", model.FormatObsTime(entry.ObservedAt), func() {
														a.updateUI(func() {})
													})
												}
											}
										}
										if len(entry.PhotoPaths) > 1 {
											statusPill(c, fmt.Sprintf("%d 张照片", len(entry.PhotoPaths)), softPillColor(t), t.TextMuted)
										}
									})
								}
							})
						})
						if row.Clicked() {
							copyE := entry
							a.journalActiveEntry = &copyE
							a.journalDetailPager.Page = 0
							a.router.Push("/journal/" + entry.ID)
						}
					})
				})
			}
		})
	})
}

func (a *AppState) buildLifelistGrid(c *ui.Context, t *ui.Theme, unlocked []model.UnlockedBird) {
	ui.Box(c).Padding(2, 18, 8, 18).Children(func() {
		minimalSearchBar(c, "ll-search", &a.session.LifelistFilter, "搜索已解锁鸟种或发现地", "搜索已解锁", t)
	})

	q := strings.ToLower(strings.TrimSpace(a.session.LifelistFilter))
	var visible []model.UnlockedBird
	for _, u := range unlocked {
		if q == "" || strings.Contains(strings.ToLower(u.ComName), q) ||
			strings.Contains(strings.ToLower(u.SciName), q) ||
			strings.Contains(strings.ToLower(u.FirstLocation), q) {
			visible = append(visible, u)
		}
	}

	ui.Scroll(c.Key("ll-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Padding(4, 18, 112, 18).Gap(12).Children(func() {
			if len(visible) == 0 {
				tactileCard(c, t).Padding(34).Center().Gap(10).Children(func() {
					tactileRecessedStyle(ui.Box(c).Size(52, 52), t, 26).Center().Children(func() {
						ui.Icon(c, iconAward).FontSize(22).TextColor(t.TextMuted)
					})
					ui.Text(c, "尚未解锁鸟种").FontSize(16).FontWeight(600)
					ui.Text(c, "记录一条观鸟日志即可在此点亮物种").FontSize(13).TextColor(t.TextMuted)
				})
				return
			}

			for i := 0; i < len(visible); i += 2 {
				rowIdx := i
				ui.Row(c.Key(fmt.Sprintf("llrow-%d", rowIdx))).FillWidth().Gap(12).Children(func() {
					for col := 0; col < 2; col++ {
						idx := rowIdx + col
						if idx >= len(visible) {
							ui.Box(c).Grow(1).Basis(0)
							continue
						}
						bird := visible[idx]
						hasImg := bird.CoverPhotoPath != "" || a.speciesThumbURL(bird.SpeciesCode) != ""
						tile := applyCardRimAndShadow(
							ui.Box(c.Key("llcard-"+bird.SpeciesCode)).Grow(1).Basis(0).Height(196),
							18, t,
						).Clip().Role(ui.RoleButton).Label(bird.ComName)

						tile.Children(func() {
							if bird.CoverPhotoPath != "" {
								a.loadLocalImageAsync(bird.CoverPhotoPath)
								if src, ok := a.localImgCache[bird.CoverPhotoPath]; ok {
									ui.Image(c, src).Fill().Fit(ui.Cover).Label(bird.ComName)
								}
							} else if thumbURL := a.speciesThumbURL(bird.SpeciesCode); thumbURL != "" {
								netimage.Image(c, thumbURL, netimage.Options{
									MaxSide:     380,
									Fit:         ui.Cover,
									Placeholder: t.Surface,
								}).Fill()
							} else {
								ui.Box(c).Fill().Center().Children(func() {
									ui.Icon(c, iconBird).FontSize(34).TextColor(t.TextMuted.Alpha(0.32))
								})
							}

							if hasImg {
								ui.Box(c).Absolute().Left(0).Bottom(0).Right(0).Height(88).PassThrough().
									LinearGradient(ui.LinearGradient{
										Angle: 180,
										Stops: []ui.GradientStop{
											{Color: ui.Transparent, At: 0},
											{Color: ui.RGBA(8, 11, 17, 0.36), At: 0.45},
											{Color: ui.RGBA(8, 11, 17, 0.86), At: 1.0},
										},
									})
							}

							ui.Textf(c, "%d 次", max(1, bird.TotalSightings)).
								Absolute().Right(10).Top(10).Padding(3, 9).Radius(999).
								FontSize(11).FontWeight(500).FontFeatures("tnum").
								Background(ui.RGBA(0, 0, 0, 0.45)).TextColor(ui.RGB(255, 255, 255))

							titleCol := t.Text
							subCol := t.TextMuted
							if hasImg {
								titleCol = ui.RGB(255, 255, 255)
								subCol = ui.RGBA(255, 255, 255, 0.80)
							}
							fam := bird.FamilyName
							if tx, ok := a.taxonByCode[bird.SpeciesCode]; ok {
								fam = model.FamilyZh(tx.FamilySciName, tx.FamilyComName)
							}
							ui.Column(c).Absolute().Left(0).Right(0).Bottom(0).Padding(0, 12, 12, 12).Gap(2).PassThrough().Children(func() {
								ui.Text(c, bird.ComName).FontSize(15).FontWeight(600).TextColor(titleCol).SingleLine()
								ui.Text(c, fallbackText(fam, bird.FirstLocation)).FontSize(12).TextColor(subCol).SingleLine()
							})
						})
						if tile.Clicked() {
							if _, ok := a.taxonByCode[bird.SpeciesCode]; ok {
								a.OpenSpeciesInGuide(bird.SpeciesCode)
							} else {
								a.session.JournalQuery = bird.ComName
								a.session.JournalSubTab = 0
								a.refreshJournalList()
							}
						}
					}
				})
			}
		})
	})
}

func (a *AppState) viewJournalDetailPage(c *ui.Context, t *ui.Theme, entryID string) {
	var entry *model.JournalEntry
	for i := range a.filteredEntries {
		if a.filteredEntries[i].ID == entryID {
			copyE := a.filteredEntries[i]
			entry = &copyE
			break
		}
	}
	if entry == nil {
		for _, e := range a.store.Entries() {
			if e.ID == entryID {
				copyE := e
				entry = &copyE
				break
			}
		}
	}
	if entry == nil {
		ui.Column(c).Fill().Center().Padding(24).Gap(12).Children(func() {
			ui.Text(c, "该观鸟日志已被删除")
			if filterPill(c, "jd-nf-back", "返回日志列表", false, t).Clicked() {
				a.goBack()
			}
		})
		return
	}

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).FillWidth().Padding(14, 18, 10, 18).Gap(8).AlignItems(ui.Center).Children(func() {
			if tactileIconBtn(c, "jd-back-btn", "返回", iconBack, 36, t).Clicked() {
				a.goBack()
			}
			ui.Text(c, entry.ComName).FontSize(16).FontWeight(600).SingleLine().Grow(1)
			if filterPill(c, "jd-edit", "编辑", false, t).Clicked() {
				a.openEditJournalModal(*entry)
			}
			if circleIconBtn(c, "jd-del", "删除日志", iconTrash, t.Danger.Alpha(0.12), t.Danger, 34).Clicked() {
				a.jDeleteTargetID = entry.ID
				a.jDeleteConfirm = true
			}
		})

		ui.Scroll(c.Key("jd-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(6, 18, 44, 18).Gap(14).Children(func() {
				if len(entry.PhotoPaths) > 0 {
					count := len(entry.PhotoPaths)
					tactileCard(c, t).Height(264).Clip().Children(func() {
						mobile.PageView(c, &a.journalDetailPager, count, func(i int) {
							p := entry.PhotoPaths[i]
							a.loadLocalImageAsync(p)
							slide := ui.Box(c.Key(fmt.Sprintf("jpslide-%d", i))).Fill().
								Role(ui.RoleButton).Label("放大查看个人照片")
							slide.Children(func() {
								if src, ok := a.localImgCache[p]; ok {
									ui.Image(c, src).Fill().Fit(ui.Cover).Label(entry.ComName)
								} else {
									ui.Box(c).Fill().Center().Children(func() {
										ui.Spinner(c).Label("读取照片")
									})
								}
							})
							if slide.Clicked() {
								a.openLocalPhotosViewer(entry.ComName+" · 我的记录", entry.PhotoPaths, i)
							}
						}).Fill()
						ui.Textf(c, "%d / %d", a.journalDetailPager.Page+1, count).
							Absolute().Left(14).Bottom(12).Padding(3, 9).Radius(999).
							FontSize(12).FontWeight(500).FontFeatures("tnum").
							Background(ui.RGBA(0, 0, 0, 0.48)).TextColor(ui.RGB(255, 255, 255))
						if count > 1 {
							ui.Box(c).Absolute().Right(14).Bottom(12).Children(func() {
								mobile.PageDots(c, a.journalDetailPager.Page, count)
							})
						}
					})
				}

				tactileCard(c, t).Padding(16).Gap(8).Children(func() {
					ui.Text(c, entry.ComName).FontSize(24).FontWeight(600).LetterSpacing(-0.1)
					ui.Text(c, entry.SciName).FontSize(14).Italic().TextColor(t.TextMuted)
					ui.Row(c).Gap(6).Wrap().Padding(4, 0, 0, 0).Children(func() {
						statusPill(c, model.FormatObsTime(entry.ObservedAt), softPillColor(t), secondaryTextColor(t))
						statusPill(c, fallbackText(entry.Location, "未填地点"), softPillColor(t), secondaryTextColor(t))
						statusPill(c, fmt.Sprintf("%d 只", max(1, entry.Count)), softPillColor(t), t.Text)
					})
				})

				if entry.AudioPath != "" {
					pst := a.player.Status()
					isThis := pst.SourceKey == entry.AudioPath
					tactileCard(c, t).Padding(14).Children(func() {
						ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
							ic := iconPlay
							lbl := "播放个人录音"
							playBtn := ui.Box(c.Key("jd-aud-play")).Size(40, 40).Radius(20).Center().Shrink(0).
								Role(ui.RoleButton).Label(lbl)
							fg := t.Text
							if isThis && pst.Playing {
								ic = iconPause
								playBtn.Label("暂停录音")
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
								if isThis && pst.Active {
									a.player.TogglePause()
								} else {
									a.player.PlayFile(entry.AudioPath, entry.ComName+" · 个人录音", model.FormatObsTime(entry.ObservedAt), func() {
										a.updateUI(func() {})
									})
								}
							}
							ui.Column(c).Grow(1).Gap(2).Children(func() {
								ui.Text(c, "现场鸟鸣录音").FontSize(14).FontWeight(500)
								ui.Text(c, filepath.Base(entry.AudioPath)).FontSize(12).TextColor(t.TextMuted).SingleLine()
							})
						})
					})
				}

				tactileCard(c, t).Padding(16).Gap(8).Selectable().Children(func() {
					ui.Text(c, "观察笔记").FontSize(16).FontWeight(600)
					if strings.TrimSpace(entry.Notes) != "" {
						ui.Text(c, entry.Notes).FontSize(15).LineHeight(1.7).TextColor(secondaryTextColor(t))
					} else {
						ui.Text(c, "本次观测未填写文字笔记。").FontSize(13).TextColor(t.TextMuted)
					}
				})

				if _, ok := a.taxonByCode[entry.SpeciesCode]; ok {
					if monochromeButton(c, "jd-goto-guide", "查看图鉴详情 ↗", false, t).FillWidth().Clicked() {
						a.OpenSpeciesInGuide(entry.SpeciesCode)
					}
				}
			})
		})
	})
}

func (a *AppState) viewJournalComposerSheet(c *ui.Context, t *ui.Theme) {
	recStatus := a.recorder.Status()
	if recStatus.Recording {
		c.After(80 * time.Millisecond)
	}
	title := "记录观鸟"
	if a.jEditID != "" {
		title = "编辑日志"
	}

	sheetBg := t.Background
	mobile.Sheet(c, &a.journalSheetOpen, mobile.SheetOptions{
		Detents:    []float32{0.92},
		NoDismiss:  recStatus.Recording || a.jRecBusy,
		Background: &sheetBg,
		Label:      title,
	}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 20, 14, 20).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				cancelBtn := ui.Box(c.Key("j-modal-cancel")).Padding(6, 4).
					Role(ui.RoleButton).Label("取消").Children(func() {
					ui.Text(c, "取消").FontSize(15).TextColor(t.TextMuted)
				})
				if cancelBtn.Clicked() {
					a.recorder.Cancel()
					a.jRecBusy = false
					a.journalSheetOpen = false
				}
				ui.Text(c, title).FontSize(16).FontWeight(600)
				saveBtn := filterPill(c, "j-modal-save", "保存记录", true, t).Disabled(recStatus.Recording || a.jRecBusy)
				if saveBtn.Clicked() {
					a.saveJournalDraft(c)
				}
			})
			minimalDivider(c, t)

			ui.Scroll(c.Key("j-sheet-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Padding(10, 18, 28, 18).Gap(10).Children(func() {
					// 1. 观测鸟种卡片
					tactileCard(c, t).Padding(13).Gap(8).Children(func() {
						ui.Text(c, "观测鸟种").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
						minimalTextInput(c, "j-species-input", &a.jDraftSpecies,
							"输入中文名或拉丁学名（如：普通翠鸟）", "鸟种", t)

						suggestions := a.speciesSuggestions(a.jDraftSpecies)
						if len(suggestions) > 0 && !strings.HasSuffix(strings.TrimSpace(a.jDraftSpecies), "]") {
							ui.Row(c).FillWidth().Gap(6).Wrap().Children(func() {
								for i, tx := range suggestions {
									item := tx
									if filterPill(c, fmt.Sprintf("jsug-%d-%s", i, item.SpeciesCode), item.ComName, false, t).Clicked() {
										a.jDraftSpecies = formatSpeciesOption(item)
									}
								}
							})
						}
					})

					// 2. 地点、时间与数量步进器卡片
					tactileCard(c, t).Padding(13).Gap(10).Children(func() {
						ui.Column(c).Gap(6).Children(func() {
							ui.Text(c, "地点与坐标").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
							ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
								minimalTextInput(c, "j-obs-loc", &a.jDraftLocation,
									"公园、湿地或坐标", "观测地点", t).Grow(1)
								gpsBtn := applySculptedSecondaryPillStyle(
									ui.Row(c.Key("j-loc-gps")).Size(44, 44).Radius(12).Center().Shrink(0).
										Role(ui.RoleButton).Label("定位当前位置"),
									t,
								)
								gpsBtn.Children(func() {
									if a.jDraftGPSBusy {
										ui.Spinner(c).Label("定位")
									} else {
										ui.Icon(c, iconLocate).FontSize(17).TextColor(opticalAccentColor(t))
									}
								})
								if gpsBtn.Clicked() {
									a.fillGPSLocationAsync("journal")
								}
							})
						})

						ui.Row(c).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
							ui.Column(c).Grow(1).Gap(6).Children(func() {
								ui.Text(c, "观测时间").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
								minimalTextInput(c, "j-obs-dt", &a.jDraftObservedAt, "", "观测时间", t)
							})

							ui.Column(c).Gap(6).Children(func() {
								ui.Text(c, "数量").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
								tactileRecessedStyle(ui.Row(c).Height(44).AlignItems(ui.Center).Gap(6).PaddingX(7), t, 12).Children(func() {
									dec := applySculptedSecondaryPillStyle(
										ui.Box(c.Key("cnt-dec")).Size(28, 28).Radius(14).Center().
											Role(ui.RoleButton).Label("减少数量").Children(func() {
											ui.Icon(c, iconMinus).FontSize(13).TextColor(t.Text)
										}),
										t,
									)
									if dec.Clicked() && a.jDraftCount > 1 {
										a.jDraftCount--
									}
									ui.Textf(c, " %d ", max(1, a.jDraftCount)).FontSize(15).FontWeight(600).FontFeatures("tnum")
									inc := applySculptedPrimaryStyle(
										ui.Box(c.Key("cnt-inc")).Size(28, 28).Radius(14).Center().
											Role(ui.RoleButton).Label("增加数量").Children(func() {
											ui.Icon(c, iconPlus).FontSize(13).TextColor(t.AccentText)
										}),
										t,
									)
									if inc.Clicked() {
										a.jDraftCount++
									}
								})
							})
						})
					})

					// 3. 现场照片卡片
					tactileCard(c, t).Padding(13).Gap(8).Children(func() {
						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "现场照片").FontSize(14).FontWeight(500)
							ui.Textf(c, "%d 张", len(a.jDraftPhotos)).FontSize(12).TextColor(t.TextMuted)
						})

						ui.ScrollHorizontal(c.Key("j-draft-photos-strip")).FillWidth().Gap(10).Children(func() {
							addTile := tactileRecessedStyle(
								ui.Column(c.Key("j-add-photo-tile")).Size(68, 68).Center().Gap(3).Shrink(0).
									Role(ui.RoleButton).Label("添加照片"),
								t, 14,
							)
							addTile.Children(func() {
								ui.Icon(c, iconCamera).FontSize(19).TextColor(t.Text)
								ui.Text(c, "添加").FontSize(11).FontWeight(500).TextColor(secondaryTextColor(t))
							})
							if addTile.Clicked() {
								a.journalPhotoSheet = true
							}

							for i, p := range a.jDraftPhotos {
								idx := i
								path := p
								a.loadLocalImageAsync(path)
								ui.Box(c.Key(fmt.Sprintf("jdp-%d", i))).Size(68, 68).Radius(14).Clip().Shrink(0).Background(softPillColor(t)).Children(func() {
									if src, ok := a.localImgCache[path]; ok {
										ui.Image(c, src).Fill().Fit(ui.Cover)
									}
									rm := circleIconBtn(c, fmt.Sprintf("jdprm-%d", i), "移除照片", iconClose, ui.RGBA(0, 0, 0, 0.58), ui.RGB(255, 255, 255), 20).
										Absolute().Top(4).Right(4)
									if rm.Clicked() {
										a.jDraftPhotos = append(a.jDraftPhotos[:idx], a.jDraftPhotos[idx+1:]...)
									}
								})
							}
						})
					})

					// 4. 鸟鸣现场录音（同心圆环录音棚质感麦克风控件）
					tactileCard(c, t).Padding(13).Gap(8).Children(func() {
						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "现场鸟鸣录音").FontSize(14).FontWeight(500)
							if filterPill(c, "j-pick-audio", "导入音频", false, t).Disabled(recStatus.Recording || a.jRecBusy).Clicked() {
								a.pickJournalAudioAsync()
							}
						})

						ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
							if !recStatus.Recording {
								halo := ui.Box(c).Size(54, 54).Radius(27).Center().Shrink(0).
									Background(t.Text.Alpha(0.06)).Border(1, subtleHairlineColor(t))
								halo.Children(func() {
									recCircle := applySculptedPrimaryStyle(
										ui.Box(c.Key("j-rec-start")).Size(42, 42).Radius(21).Center().
											Role(ui.RoleButton).Label("开始录音"),
										t,
									)
									recCircle.Children(func() {
										ui.Icon(c, iconMic).FontSize(18).TextColor(t.AccentText)
									})
									if recCircle.Clicked() && !a.jRecBusy {
										a.startJournalRecording()
									}
								})

								ui.Column(c).Grow(1).Gap(3).Children(func() {
									if a.jDraftAudio != "" {
										ui.Text(c, "已录制："+filepath.Base(a.jDraftAudio)).FontSize(14).FontWeight(500).SingleLine()
										ui.Row(c).Gap(6).Children(func() {
											if filterPill(c, "j-prev-aud", "试听", true, t).Clicked() {
												a.player.PlayFile(a.jDraftAudio, "录音试听", filepath.Base(a.jDraftAudio), func() {
													a.updateUI(func() {})
												})
											}
											if filterPill(c, "j-clr-aud", "移除", false, t).Clicked() {
												a.jDraftAudio = ""
											}
										})
									} else {
										ui.Text(c, "点击麦克风录制现场鸟鸣").FontSize(14).FontWeight(500)
										ui.Text(c, "支持 WAV / MP3 / OGG / FLAC").FontSize(12).TextColor(t.TextMuted)
									}
								})
							} else {
								halo := ui.Box(c).Size(54, 54).Radius(27).Center().Shrink(0).
									Background(t.Danger.Alpha(0.14)).Border(1, t.Danger.Alpha(0.3))
								halo.Children(func() {
									stopCircle := ui.Box(c.Key("j-rec-stop")).Size(42, 42).Radius(21).Center().
										LinearGradient(ui.LinearGradient{From: ui.Hex("#EF4444"), To: ui.Hex("#DC2626"), Angle: 180}).
										InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.35)).
										Shadow(0, 4, 12, 0, t.Danger.Alpha(0.4)).
										Role(ui.RoleButton).Label("停止录音")
									stopCircle.Children(func() {
										ui.Icon(c, iconStop).FontSize(17).TextColor(ui.RGB(255, 255, 255))
									})
									if stopCircle.Clicked() {
										a.stopJournalRecording()
									}
								})

								ui.Column(c).Grow(1).Gap(6).Children(func() {
									ui.Textf(c, "正在录音 %s", formatDuration(recStatus.Elapsed)).FontSize(14).FontWeight(600).TextColor(t.Danger)
									ui.Meter(c, recStatus.Level, 0, 1, &ui.MeterLevels{Warning: 0.75, Critical: 0.92}).
										Label("录音电平").FillWidth()
								})
							}
						})
					})

					// 5. 观察笔记卡片
					tactileCard(c, t).Padding(13).Gap(6).Children(func() {
						ui.Text(c, "观察笔记").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
						minimalTextArea(c, "j-obs-notes", &a.jDraftNotes,
							"记录生境、羽色、鸣叫或行为特征…", "文字记录", 80, t)
					})

					if a.jDraftError != "" {
						ui.Text(c, a.jDraftError).FontSize(13).FontWeight(600).TextColor(t.Danger)
					}
				})
			})
		})
	})
}

func (a *AppState) startJournalRecording() {
	a.jDraftError = ""
	if a.win == nil {
		if err := a.recorder.Start(); err != nil {
			a.jDraftError = err.Error()
		}
		return
	}
	a.jRecBusy = true
	go func() {
		err := a.recorder.Start()
		a.updateUI(func() {
			a.jRecBusy = false
			if err != nil {
				a.jDraftError = err.Error()
			}
		})
	}()
}

func (a *AppState) stopJournalRecording() {
	dst := filepath.Join(a.store.DataDir(), "journal", "media", fmt.Sprintf("rec_%d.wav", time.Now().UnixMilli()))
	if a.win == nil {
		if err := a.recorder.Stop(dst); err != nil {
			a.jDraftError = err.Error()
		} else {
			a.jDraftAudio = dst
			a.jDraftError = ""
		}
		return
	}
	a.jRecBusy = true
	go func() {
		err := a.recorder.Stop(dst)
		a.updateUI(func() {
			a.jRecBusy = false
			if err != nil {
				a.jDraftError = err.Error()
			} else {
				a.jDraftAudio = dst
				a.jDraftError = ""
				a.hapticSuccess()
			}
		})
	}()
}

func (a *AppState) viewJournalPhotoActionSheet(c *ui.Context) {
	mobile.ActionSheet(c, &a.journalPhotoSheet, mobile.ActionSheetOptions{
		Title:  "添加观鸟照片",
		Cancel: "取消",
	},
		mobile.Action{
			Label: "拍照记录",
			OnTap: func() { a.captureJournalPhotoAsync() },
		},
		mobile.Action{
			Label: "从相册选择",
			OnTap: func() { a.pickJournalPhotosAsync() },
		},
	)
}

func (a *AppState) viewJournalExportActionSheet(c *ui.Context) {
	mobile.ActionSheet(c, &a.exportSheetOpen, mobile.ActionSheetOptions{
		Title:  "分享与备份",
		Cancel: "取消",
	},
		mobile.Action{
			Label: "系统分享最近观察",
			OnTap: func() { a.ShareOrExportJournal("share") },
		},
		mobile.Action{
			Label: "导出为 CSV 表格",
			OnTap: func() { a.ShareOrExportJournal("csv") },
		},
		mobile.Action{
			Label: "导出为 JSON 文件",
			OnTap: func() { a.ShareOrExportJournal("json") },
		},
	)
}

func (a *AppState) captureJournalPhotoAsync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		p, err := imagepicker.Capture(ctx)
		cancel()
		if err != nil {
			if errors.Is(err, errors.ErrUnsupported) {
				a.pickJournalPhotosDesktopFallback()
				return
			}
			a.updateUI(func() { a.jDraftError = fmt.Sprintf("调用相机失败: %v", err) })
			return
		}
		if p != "" {
			a.updateUI(func() { a.jDraftPhotos = append(a.jDraftPhotos, p) })
		}
	}()
}

func (a *AppState) pickJournalPhotosAsync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		paths, err := imagepicker.Pick(ctx, imagepicker.Options{Kind: imagepicker.Images, Limit: 9})
		cancel()
		if err != nil {
			if errors.Is(err, errors.ErrUnsupported) {
				a.pickJournalPhotosDesktopFallback()
				return
			}
			a.updateUI(func() { a.jDraftError = fmt.Sprintf("打开相册失败: %v", err) })
			return
		}
		if len(paths) > 0 {
			a.updateUI(func() { a.jDraftPhotos = append(a.jDraftPhotos, paths...) })
		}
	}()
}

func (a *AppState) pickJournalPhotosDesktopFallback() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:   a.win,
		Title:    "选择观测照片",
		Filters:  []mygo.FileFilter{{Name: "图片文件", Extensions: []string{"png", "jpg", "jpeg", "webp", "bmp"}}},
		Multiple: true,
	})
	if err != nil || len(paths) == 0 {
		return
	}
	a.updateUI(func() {
		a.jDraftPhotos = append(a.jDraftPhotos, paths...)
	})
}

func (a *AppState) pickJournalAudioAsync() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:  a.win,
			Title:   "选择现场录音文件",
			Filters: []mygo.FileFilter{{Name: "音频文件", Extensions: []string{"wav", "mp3", "ogg", "flac", "m4a"}}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.updateUI(func() {
			a.jDraftAudio = paths[0]
		})
	}()
}

func (a *AppState) saveJournalDraft(c *ui.Context) {
	code, comName, sciName, order, family := a.resolveDraftSpecies()
	if code == "" || comName == "" {
		a.jDraftError = "请先选择或输入一个鸟种"
		return
	}

	entry := model.JournalEntry{
		ID:          a.jEditID,
		SpeciesCode: code,
		ComName:     comName,
		SciName:     sciName,
		Order:       order,
		FamilyName:  family,
		ObservedAt:  strings.TrimSpace(a.jDraftObservedAt),
		Location:    strings.TrimSpace(a.jDraftLocation),
		Lat:         a.jDraftLat,
		Lng:         a.jDraftLng,
		Count:       max(1, a.jDraftCount),
		Notes:       strings.TrimSpace(a.jDraftNotes),
		PhotoPaths:  a.jDraftPhotos,
		AudioPath:   a.jDraftAudio,
	}

	if a.jEditID == "" {
		saved, newlyUnlocked, err := a.store.AddJournalEntry(entry)
		if err != nil {
			a.jDraftError = err.Error()
			return
		}
		a.journalSheetOpen = false
		a.refreshJournalList()
		a.refreshStorageStats()
		a.hapticSuccess()
		if newlyUnlocked {
			c.Toast(fmt.Sprintf("已解锁新鸟种：%s（%d 种）", saved.ComName, len(a.store.UnlockedBirds())))
		} else {
			c.Toast("已保存观鸟日志")
		}
	} else {
		_, err := a.store.UpdateJournalEntry(entry)
		if err != nil {
			a.jDraftError = err.Error()
			return
		}
		a.journalSheetOpen = false
		a.refreshJournalList()
		a.refreshStorageStats()
		a.hapticSuccess()
		c.Toast("已更新观鸟日志")
	}
}

func (a *AppState) viewJournalDeleteDialog(c *ui.Context) {
	switch ui.AlertDialog(c, &a.jDeleteConfirm, "删除观鸟日志？", "删除后不可恢复；若该鸟种已无其他记录，解锁状态也会同步取消。", "取消", "确认删除") {
	case 1:
		target := a.jDeleteTargetID
		_ = a.store.DeleteJournalEntry(target)
		a.jDeleteTargetID = ""
		a.refreshJournalList()
		a.refreshStorageStats()
		if strings.HasPrefix(a.router.Path(), "/journal/") {
			a.goBack()
		}
		c.Toast("已删除观鸟日志")
	}
}
