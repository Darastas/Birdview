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
	progress := float64(len(unlocked)) / float64(chinaTotal)
	if progress > 1 {
		progress = 1
	}
	familyCount := make(map[string]int)
	for _, u := range unlocked {
		familyCount[fallbackText(u.FamilyName, "未分类")]++
	}

	ui.Box(c).Fill().Children(func() {
		ui.Column(c).Fill().Children(func() {
			// 顶部标题与个人解锁成就概览卡片
			ui.Column(c).FillWidth().Padding(12, 16, 10, 16).Gap(10).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "观鸟日志").FontSize(26).Bold().Role(ui.RoleHeading).Level(1)
					shareBtn := ui.Row(c.Key("j-export-btn")).Height(34).PaddingX(12).Radius(17).
						Background(t.Surface).Gap(5).AlignItems(ui.Center).Role(ui.RoleButton).Label("分享或导出")
					if !t.Dark {
						shareBtn.Shadow(0, 1, 4, 0, ui.RGBA(15, 23, 42, 0.06))
					}
					shareBtn.Children(func() {
						ui.Icon(c, iconShare).FontSize(13).TextColor(t.Text)
						ui.Text(c, "导出/分享").FontSize(12.5).FontWeight(600)
					})
					if shareBtn.Clicked() {
						a.exportSheetOpen = true
					}
				})

				// 简洁成就进度条卡片
				elevatedCard(c, t).Padding(14, 16).Gap(8).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.End).Children(func() {
						ui.Row(c).Gap(6).AlignItems(ui.Baseline).Children(func() {
							ui.Textf(c, "%d", len(unlocked)).FontSize(26).Bold().TextColor(t.Accent).FontFeatures("tnum")
							ui.Textf(c, "/ %d 种已解锁", chinaTotal).FontSize(13).TextColor(t.TextMuted).FontFeatures("tnum")
						})
						ui.Textf(c, "%d 条日志 · %d 科", len(entries), len(familyCount)).FontSize(12).TextColor(t.TextMuted)
					})
					ui.Progress(c, progress).Label("个人鸟种解锁进度").FillWidth()
				})

				segmentedPillStrip(c, "j-subtab", &a.session.JournalSubTab, t, "时间线日志", "已解锁图谱")
			})

			if a.session.JournalSubTab == 0 {
				a.buildJournalTimeline(c, t)
			} else {
				a.buildLifelistGrid(c, t, unlocked)
			}
		})

		// 右下角新建观鸟日志悬浮主按钮（录音/弹窗开启时不重叠）
		if !a.journalSheetOpen {
			fab := mobile.FloatingButton(c.Key("j-fab"), "新建观鸟日志", paintIconPlus).
				Absolute().Right(16).Bottom(16)
			if fab.Clicked() {
				a.OpenNewJournalModal("", "")
			}
		}
	})
}

func (a *AppState) buildJournalTimeline(c *ui.Context, t *ui.Theme) {
	ui.Box(c).Padding(0, 16, 8, 16).Children(func() {
		if ui.SearchField(c.Key("j-search"), &a.session.JournalQuery).
			Placeholder("搜索鸟种、地点或观察笔记…").Label("搜索日志").
			FillWidth().Height(38).Radius(10).Changed() {
			a.refreshJournalList()
		}
	})

	scroll := ui.Scroll(c.Key("j-timeline-scroll")).Grow(1).Basis(0).FillWidth()
	if scroll.PullToRefresh(&a.journalRefresh) {
		a.refreshJournalList()
		a.journalRefresh.Refreshing = false
	}
	scroll.Children(func() {
		ui.Column(c).FillWidth().Padding(2, 16, 88, 16).Gap(10).Children(func() {
			if len(a.filteredEntries) == 0 {
				elevatedCard(c, t).Padding(28).Center().Gap(8).Children(func() {
					ui.Icon(c, iconJournal).FontSize(32).TextColor(t.TextMuted.Alpha(0.6))
					ui.Text(c, "暂无观鸟日志").FontSize(16).FontWeight(600)
					ui.Text(c, "点击右下角 + 记录今天的目击并点亮图鉴鸟种").FontSize(13).TextColor(t.TextMuted)
				})
				return
			}

			for i, item := range a.filteredEntries {
				entry := item
				ui.Box(c.Key("jsw-"+entry.ID)).Radius(16).Clip().Children(func() {
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
						card := elevatedRowCard(c, t).Padding(12, 14).Gap(12).AlignItems(ui.Center).
							Role(ui.RoleButton).Label(entry.ComName + " 日志详情")
						card.Children(func() {
							// 左侧用户自拍照片缩略图或无图图标块
							if len(entry.PhotoPaths) > 0 {
								p := entry.PhotoPaths[0]
								a.loadLocalImageAsync(p)
								if src, ok := a.localImgCache[p]; ok {
									ui.Image(c, src).Size(64, 64).Fit(ui.Cover).Radius(12).Shrink(0).Label(entry.ComName)
								} else {
									ui.Box(c).Size(64, 64).Radius(12).Background(softPillColor(t)).Center().Shrink(0).Children(func() {
										ui.Icon(c, iconImage).FontSize(22).TextColor(t.TextMuted.Alpha(0.6))
									})
								}
							}

							ui.Column(c).Grow(1).Basis(0).Gap(4).Children(func() {
								ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
									ui.Text(c, entry.ComName).FontSize(16).FontWeight(600).SingleLine().Grow(1)
									statusPill(c, fmt.Sprintf("%d 只", max(1, entry.Count)), softPillColor(t), secondaryTextColor(t))
								})
								ui.Textf(c, "%s · %s", model.FormatObsTime(entry.ObservedAt), fallbackText(entry.Location, "野外地点")).
									FontSize(12.5).TextColor(t.TextMuted).SingleLine()

								if strings.TrimSpace(entry.Notes) != "" {
									ui.Text(c, entry.Notes).FontSize(13).TextColor(secondaryTextColor(t)).MaxLines(2)
								}

								if entry.AudioPath != "" || len(entry.PhotoPaths) > 1 {
									ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
										if entry.AudioPath != "" {
											pst := a.player.Status()
											isThis := pst.SourceKey == entry.AudioPath
											label := "播放现场录音"
											if isThis && pst.Playing {
												label = "暂停录音"
											}
											playPill := ui.Row(c.Key(fmt.Sprintf("jt-aud-%d", i))).Padding(3, 10).Radius(999).
												Background(t.Accent.Alpha(0.12)).Gap(4).AlignItems(ui.Center).Role(ui.RoleButton).Label(label)
											playPill.Children(func() {
												ic := iconPlay
												if isThis && pst.Playing {
													ic = iconPause
												}
												ui.Icon(c, ic).FontSize(10).TextColor(t.Accent)
												ui.Text(c, label).FontSize(11.5).FontWeight(600).TextColor(t.Accent)
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

							ui.Icon(c, iconChevronRight).FontSize(16).TextColor(t.TextMuted.Alpha(0.5))
						})
						if card.Clicked() {
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
	ui.Box(c).Padding(0, 16, 8, 16).Children(func() {
		ui.SearchField(c.Key("ll-search"), &a.session.LifelistFilter).
			Placeholder("搜索已解锁鸟种或发现地…").Label("搜索已解锁").
			FillWidth().Height(38).Radius(10)
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
		ui.Column(c).FillWidth().Padding(2, 16, 88, 16).Gap(12).Children(func() {
			if len(visible) == 0 {
				elevatedCard(c, t).Padding(28).Center().Gap(8).Children(func() {
					ui.Icon(c, iconAward).FontSize(32).TextColor(t.TextMuted.Alpha(0.6))
					ui.Text(c, "尚未解锁鸟种").FontSize(16).FontWeight(600)
					ui.Text(c, "新建一条观鸟日志即可在此永久点亮对应物种").FontSize(13).TextColor(t.TextMuted)
				})
				return
			}

			for i := 0; i < len(visible); i += 2 {
				rowIdx := i
				ui.Row(c.Key(fmt.Sprintf("llrow-%d", rowIdx))).FillWidth().Gap(10).Children(func() {
					for col := 0; col < 2; col++ {
						idx := rowIdx + col
						if idx >= len(visible) {
							ui.Box(c).Grow(1).Basis(0)
							continue
						}
						bird := visible[idx]
						card := elevatedCard(c.Key("llcard-"+bird.SpeciesCode), t).Grow(1).Basis(0).Clip().
							Role(ui.RoleButton).Label(bird.ComName)
						card.Children(func() {
							ui.Box(c).FillWidth().Height(112).Background(softPillColor(t)).Children(func() {
								if bird.CoverPhotoPath != "" {
									a.loadLocalImageAsync(bird.CoverPhotoPath)
									if src, ok := a.localImgCache[bird.CoverPhotoPath]; ok {
										ui.Image(c, src).Fill().Fit(ui.Cover).Label(bird.ComName)
									} else {
										ui.Box(c).Fill().Center().Children(func() {
											ui.Icon(c, iconImage).FontSize(22).TextColor(t.TextMuted.Alpha(0.6))
										})
									}
								} else if thumbURL := a.speciesThumbURL(bird.SpeciesCode); thumbURL != "" {
									netimage.Image(c, thumbURL, netimage.Options{
										MaxSide:     320,
										Fit:         ui.Cover,
										Placeholder: softPillColor(t),
									}).Fill()
								} else {
									ui.Box(c).Fill().Center().Children(func() {
										ui.Icon(c, iconAward).FontSize(28).TextColor(t.Accent.Alpha(0.7))
									})
								}
								ui.Textf(c, "%d 次", max(1, bird.TotalSightings)).
									Absolute().Right(8).Bottom(8).Padding(2, 8).Radius(999).
									FontSize(11).FontWeight(600).
									Background(ui.RGBA(15, 23, 42, 0.58)).TextColor(ui.RGB(255, 255, 255))
							})

							ui.Column(c).FillWidth().Padding(10, 12).Gap(2).Children(func() {
								ui.Text(c, bird.ComName).FontSize(15).FontWeight(600).SingleLine()
								fam := bird.FamilyName
								if tx, ok := a.taxonByCode[bird.SpeciesCode]; ok {
									fam = model.FamilyZh(tx.FamilySciName, tx.FamilyComName)
								}
								ui.Text(c, fallbackText(fam, bird.FirstLocation)).FontSize(12).TextColor(t.TextMuted).SingleLine()
							})
						})
						if card.Clicked() {
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
			if ui.Button(c.Key("jd-nf-back"), "返回日志列表").Clicked() {
				a.goBack()
			}
		})
		return
	}

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Padding(10, 16, 8, 16).Gap(8).AlignItems(ui.Center).Children(func() {
			if circleIconBtn(c, "jd-back-btn", "返回", iconBack, t.Surface, t.Text, 38).Clicked() {
				a.goBack()
			}
			ui.Text(c, entry.ComName).FontSize(17).FontWeight(600).SingleLine().Grow(1)
			if filterPill(c, "jd-edit", "编辑", false, t).Clicked() {
				a.openEditJournalModal(*entry)
			}
			if circleIconBtn(c, "jd-del", "删除日志", iconTrash, t.Danger.Alpha(0.12), t.Danger, 36).Clicked() {
				a.jDeleteTargetID = entry.ID
				a.jDeleteConfirm = true
			}
		})

		ui.Scroll(c.Key("jd-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(0, 0, 28, 0).Gap(14).Children(func() {
				if len(entry.PhotoPaths) > 0 {
					count := len(entry.PhotoPaths)
					ui.Box(c).FillWidth().Height(260).Background(softPillColor(t)).Children(func() {
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
							Absolute().Right(12).Bottom(12).Padding(4, 10).Radius(999).
							FontSize(11.5).FontWeight(600).
							Background(ui.RGBA(15, 23, 42, 0.58)).TextColor(ui.RGB(255, 255, 255))
					})
					if count > 1 {
						mobile.PageDots(c, a.journalDetailPager.Page, count)
					}
				}

				ui.Box(c).PaddingX(16).Children(func() {
					elevatedCard(c, t).Padding(16).Gap(8).Children(func() {
						ui.Text(c, entry.ComName).FontSize(22).Bold()
						ui.Text(c, entry.SciName).Italic().FontSize(14).TextColor(t.TextMuted)
						ui.Row(c).Gap(8).Wrap().Children(func() {
							statusPill(c, model.FormatObsTime(entry.ObservedAt), softPillColor(t), secondaryTextColor(t))
							statusPill(c, fallbackText(entry.Location, "未填地点"), softPillColor(t), secondaryTextColor(t))
							statusPill(c, fmt.Sprintf("%d 只", max(1, entry.Count)), t.Accent.Alpha(0.12), t.Accent)
						})
					})
				})

				if entry.AudioPath != "" {
					ui.Box(c).PaddingX(16).Children(func() {
						pst := a.player.Status()
						isThis := pst.SourceKey == entry.AudioPath
						elevatedRowCard(c, t).Padding(14).Gap(12).AlignItems(ui.Center).Children(func() {
							ic := iconPlay
							lbl := "播放个人录音"
							if isThis && pst.Playing {
								ic = iconPause
								lbl = "暂停录音"
							}
							if circleIconBtn(c, "jd-aud-play", lbl, ic, t.Accent, t.AccentText, 42).Clicked() {
								if isThis && pst.Active {
									a.player.TogglePause()
								} else {
									a.player.PlayFile(entry.AudioPath, entry.ComName+" · 个人录音", model.FormatObsTime(entry.ObservedAt), func() {
										a.updateUI(func() {})
									})
								}
							}
							ui.Column(c).Grow(1).Gap(2).Children(func() {
								ui.Text(c, "现场鸟鸣录音").FontSize(15).FontWeight(600)
								ui.Text(c, filepath.Base(entry.AudioPath)).FontSize(12).TextColor(t.TextMuted).SingleLine()
							})
						})
					})
				}

				ui.Box(c).PaddingX(16).Children(func() {
					elevatedCard(c, t).Padding(16).Gap(8).Selectable().Children(func() {
						ui.Text(c, "观察笔记").FontSize(15).FontWeight(600)
						if strings.TrimSpace(entry.Notes) != "" {
							ui.Text(c, entry.Notes).FontSize(14.5).LineHeight(1.6)
						} else {
							ui.Text(c, "本次观测未填写文字笔记。").FontSize(13).TextColor(t.TextMuted)
						}
					})
				})

				if _, ok := a.taxonByCode[entry.SpeciesCode]; ok {
					ui.Box(c).PaddingX(16).Children(func() {
						if ui.Button(c.Key("jd-goto-guide"), "查看图鉴详情 ↗").FillWidth().Height(44).Radius(22).Clicked() {
							a.OpenSpeciesInGuide(entry.SpeciesCode)
						}
					})
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

	mobile.Sheet(c, &a.journalSheetOpen, mobile.SheetOptions{
		Detents:   []float32{0.92},
		NoDismiss: recStatus.Recording || a.jRecBusy,
		Label:     title,
	}, func() {
		ui.Column(c).Fill().Children(func() {
			// 表单顶栏：取消 / 标题 / 保存
			ui.Row(c).FillWidth().Padding(8, 16, 12, 16).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				if ui.Button(c.Key("j-modal-cancel"), "取消").Clicked() {
					a.recorder.Cancel()
					a.jRecBusy = false
					a.journalSheetOpen = false
				}
				ui.Text(c, title).FontSize(17).FontWeight(600)
				if ui.PrimaryButton(c.Key("j-modal-save"), "保存记录").Disabled(recStatus.Recording || a.jRecBusy).Radius(16).Clicked() {
					a.saveJournalDraft(c)
				}
			})

			ui.Scroll(c.Key("j-sheet-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Padding(4, 16, 28, 16).Gap(14).Children(func() {
					// 1. 鸟种输入 + 实时联想选择
					elevatedCard(c, t).Padding(14).Gap(8).Children(func() {
						ui.Text(c, "观测鸟种").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
						ui.TextInput(c.Key("j-species-input"), &a.jDraftSpecies).
							Placeholder("输入鸟种中文名或拉丁学名（如：普通翠鸟）").
							Label("鸟种").FillWidth().Height(42)

						suggestions := a.speciesSuggestions(a.jDraftSpecies)
						// 若已选定为完整 "[code]" 格式则收起联想列表
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

					// 2. 地点、时间与数量步进器
					elevatedCard(c, t).Padding(14).Gap(12).Children(func() {
						ui.Column(c).Gap(6).Children(func() {
							ui.Text(c, "地点与坐标").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
							ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
								ui.TextInput(c.Key("j-obs-loc"), &a.jDraftLocation).
									Placeholder("公园、湿地或坐标").Label("观测地点").Grow(1).Height(42)
								gpsBtn := ui.Row(c.Key("j-loc-gps")).Size(42, 42).Radius(12).
									Background(softPillColor(t)).Center().Shrink(0).
									Role(ui.RoleButton).Label("定位当前位置")
								gpsBtn.Children(func() {
									if a.jDraftGPSBusy {
										ui.Spinner(c).Label("定位")
									} else {
										ui.Icon(c, iconLocate).FontSize(17).TextColor(t.Accent)
									}
								})
								if gpsBtn.Clicked() {
									a.fillGPSLocationAsync("journal")
								}
							})
						})

						ui.Row(c).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
							ui.Column(c).Grow(1).Gap(6).Children(func() {
								ui.Text(c, "观测时间").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
								ui.TextInput(c.Key("j-obs-dt"), &a.jDraftObservedAt).Label("观测时间").FillWidth().Height(40)
							})

							ui.Column(c).Gap(6).AlignItems(ui.End).Children(func() {
								ui.Text(c, "数量").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
								ui.Row(c).Height(40).Radius(12).Background(softPillColor(t)).AlignItems(ui.Center).Gap(6).PaddingX(6).Children(func() {
									if circleIconBtn(c, "cnt-dec", "减少数量", iconMinus, t.Surface, t.Text, 28).Clicked() && a.jDraftCount > 1 {
										a.jDraftCount--
									}
									ui.Textf(c, " %d ", max(1, a.jDraftCount)).FontSize(15).FontWeight(600).FontFeatures("tnum")
									if circleIconBtn(c, "cnt-inc", "增加数量", iconPlus, t.Accent, t.AccentText, 28).Clicked() {
										a.jDraftCount++
									}
								})
							})
						})
					})

					// 3. 现场照片（横向滚动条 + 相册/相机 ActionSheet）
					elevatedCard(c, t).Padding(14).Gap(10).Children(func() {
						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "现场照片").FontSize(14).FontWeight(600)
							ui.Textf(c, "%d 张", len(a.jDraftPhotos)).FontSize(12).TextColor(t.TextMuted)
						})

						ui.ScrollHorizontal(c.Key("j-draft-photos-strip")).FillWidth().Gap(10).Children(func() {
							addTile := ui.Column(c.Key("j-add-photo-tile")).Size(76, 76).Radius(14).
								Background(softPillColor(t)).Center().Gap(4).Shrink(0).
								Role(ui.RoleButton).Label("添加照片")
							addTile.Children(func() {
								ui.Icon(c, iconCamera).FontSize(22).TextColor(t.Accent)
								ui.Text(c, "添加照片").FontSize(11).FontWeight(600).TextColor(t.Accent)
							})
							if addTile.Clicked() {
								a.journalPhotoSheet = true
							}

							for i, p := range a.jDraftPhotos {
								idx := i
								path := p
								a.loadLocalImageAsync(path)
								ui.Box(c.Key(fmt.Sprintf("jdp-%d", i))).Size(76, 76).Radius(14).Clip().Shrink(0).Background(softPillColor(t)).Children(func() {
									if src, ok := a.localImgCache[path]; ok {
										ui.Image(c, src).Fill().Fit(ui.Cover)
									}
									rm := circleIconBtn(c, fmt.Sprintf("jdprm-%d", i), "移除照片", iconClose, ui.RGBA(15, 23, 42, 0.65), ui.RGB(255, 255, 255), 22).
										Absolute().Top(4).Right(4)
									if rm.Clicked() {
										a.jDraftPhotos = append(a.jDraftPhotos[:idx], a.jDraftPhotos[idx+1:]...)
									}
								})
							}
						})
					})

					// 4. 鸟鸣现场录音区（大圆形脉冲录音按钮 + 导入音频）
					elevatedCard(c, t).Padding(14).Gap(12).Children(func() {
						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "现场鸟鸣录音").FontSize(14).FontWeight(600)
							if ui.Button(c.Key("j-pick-audio"), "导入音频").Disabled(recStatus.Recording || a.jRecBusy).Clicked() {
								a.pickJournalAudioAsync()
							}
						})

						ui.Row(c).FillWidth().Gap(14).AlignItems(ui.Center).Children(func() {
							if !recStatus.Recording {
								recCircle := ui.Box(c.Key("j-rec-start")).Size(60, 60).Radius(30).
									Background(t.Accent).Center().Shrink(0).
									Role(ui.RoleButton).Label("开始录音")
								recCircle.Children(func() {
									ui.Icon(c, iconMic).FontSize(26).TextColor(t.AccentText)
								})
								if recCircle.Clicked() && !a.jRecBusy {
									a.startJournalRecording()
								}

								ui.Column(c).Grow(1).Gap(2).Children(func() {
									if a.jDraftAudio != "" {
										ui.Text(c, "已完成录音："+filepath.Base(a.jDraftAudio)).FontSize(13.5).FontWeight(600).SingleLine()
										ui.Row(c).Gap(8).Children(func() {
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
										ui.Text(c, "点击蓝色麦克风立即录制").FontSize(14).FontWeight(600)
										ui.Text(c, "或点击右上角导入 WAV / MP3 / OGG / FLAC").FontSize(12).TextColor(t.TextMuted)
									}
								})
							} else {
								stopCircle := ui.Box(c.Key("j-rec-stop")).Size(60, 60).Radius(30).
									Background(t.Danger).Center().Shrink(0).
									Role(ui.RoleButton).Label("停止录音")
								stopCircle.Children(func() {
									ui.Icon(c, iconStop).FontSize(24).TextColor(ui.RGB(255, 255, 255))
								})
								if stopCircle.Clicked() {
									a.stopJournalRecording()
								}

								ui.Column(c).Grow(1).Gap(6).Children(func() {
									ui.Textf(c, "正在录音 %s", formatDuration(recStatus.Elapsed)).FontSize(15).Bold().TextColor(t.Danger)
									ui.Meter(c, recStatus.Level, 0, 1, &ui.MeterLevels{Warning: 0.75, Critical: 0.92}).
										Label("录音电平").FillWidth()
								})
							}
						})
					})

					// 5. 文字观察笔记
					elevatedCard(c, t).Padding(14).Gap(6).Children(func() {
						ui.Text(c, "观察笔记").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
						ui.TextArea(c.Key("j-obs-notes"), &a.jDraftNotes).
							Placeholder("记录生境、羽色、鸣叫或行为特征…").
							Label("文字记录").Height(92).FillWidth()
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
