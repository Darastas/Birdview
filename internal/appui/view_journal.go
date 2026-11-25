package appui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
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
	limit := 2000
	for _, tx := range a.allTaxa {
		if seen[tx.SpeciesCode] {
			continue
		}
		if len(opts) >= limit {
			break
		}
		opts = append(opts, formatSpeciesOption(tx))
		seen[tx.SpeciesCode] = true
	}
	a.journalSpeciesOpts = opts
}

func (a *AppState) ensureOptionForSpecies(speciesCode string) string {
	tx, ok := a.taxonByCode[speciesCode]
	if !ok {
		return ""
	}
	label := formatSpeciesOption(tx)
	for _, opt := range a.journalSpeciesOpts {
		if opt == label {
			return label
		}
	}
	a.journalSpeciesOpts = append([]string{label}, a.journalSpeciesOpts...)
	return label
}

func (a *AppState) resolveDraftSpecies() (code, comName, sciName, order, family string) {
	raw := strings.TrimSpace(a.jDraftSpecies)
	if raw == "" {
		return "", "", "", "", ""
	}
	// 解析格式 "普通翠鸟 (Alcedo atthis) [comkin1]"
	if lb := strings.LastIndex(raw, "["); lb >= 0 && strings.HasSuffix(raw, "]") {
		c := strings.TrimSpace(raw[lb+1 : len(raw)-1])
		if tx, ok := a.taxonByCode[c]; ok {
			return tx.SpeciesCode, tx.ComName, tx.SciName, tx.Order, tx.FamilyComName
		}
	}
	// 按中文名、学名或代码精确/包含匹配
	lower := strings.ToLower(raw)
	for _, tx := range a.allTaxa {
		if strings.EqualFold(tx.SpeciesCode, raw) || strings.EqualFold(tx.ComName, raw) || strings.EqualFold(tx.SciName, raw) {
			return tx.SpeciesCode, tx.ComName, tx.SciName, tx.Order, tx.FamilyComName
		}
	}
	for _, tx := range a.allTaxa {
		if strings.Contains(strings.ToLower(tx.ComName), lower) || strings.Contains(strings.ToLower(tx.SciName), lower) {
			return tx.SpeciesCode, tx.ComName, tx.SciName, tx.Order, tx.FamilyComName
		}
	}
	// 离线手输兜底：用输入文本作为鸟种名
	slug := strings.ReplaceAll(lower, " ", "_")
	return "custom_" + slug, raw, raw, "", ""
}

// OpenNewJournalModal 打开新建观鸟日志弹窗（可预填鸟种代码与观测地点）。
func (a *AppState) OpenNewJournalModal(speciesCode, location string) {
	a.page = "journal"
	a.jEditID = ""
	a.jDraftSpecies = ""
	if speciesCode != "" {
		if label := a.ensureOptionForSpecies(speciesCode); label != "" {
			a.jDraftSpecies = label
		} else {
			a.jDraftSpecies = speciesCode
		}
	}
	a.jDraftObservedAt = time.Now().Format("2006-01-02 15:04")
	a.jDraftLocation = location
	a.jDraftCount = "1"
	a.jDraftNotes = ""
	a.jDraftPhotos = nil
	a.jDraftAudio = ""
	a.jDraftError = ""
	a.journalModalOpen = true
	a.invalidateUI()
}

func (a *AppState) openEditJournalModal(e model.JournalEntry) {
	a.jEditID = e.ID
	if label := a.ensureOptionForSpecies(e.SpeciesCode); label != "" {
		a.jDraftSpecies = label
	} else {
		a.jDraftSpecies = e.ComName
	}
	a.jDraftObservedAt = e.ObservedAt
	a.jDraftLocation = e.Location
	a.jDraftCount = strconv.Itoa(max(1, e.Count))
	a.jDraftNotes = e.Notes
	a.jDraftPhotos = append([]string(nil), e.PhotoPaths...)
	a.jDraftAudio = e.AudioPath
	a.jDraftError = ""
	a.journalModalOpen = true
}

func (a *AppState) refreshJournalList() {
	entries := a.store.Entries()
	q := strings.ToLower(strings.TrimSpace(a.journalQuery))
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
	a.journalSelected = -1
	for i, e := range out {
		if e.ID == a.journalActiveID {
			a.journalSelected = i
			break
		}
	}
	if a.journalSelected < 0 && len(out) > 0 {
		a.journalSelected = 0
		a.journalActiveID = out[0].ID
		a.journalPhotoIndex = 0
	}
	if len(out) == 0 {
		a.journalActiveID = ""
	}
}

func (a *AppState) viewJournal(c *ui.Context, t *ui.Theme) {
	ui.Split(c, &a.journalSplit,
		func() { a.viewJournalListPane(c, t) },
		func() { a.viewJournalDetailPane(c, t) },
	).SplitReset(380).Fill()
}

func (a *AppState) viewJournalListPane(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(14, 14, 10, 14).Gap(10).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "观鸟日志").FontSize(17).Bold()
				if ui.PrimaryButton(c.Key("j-new-btn"), "").Children(func() {
					ui.Icon(c, iconPlus).FontSize(14)
					ui.Text(c, "新建记录").SingleLine()
				}).Clicked() {
					a.OpenNewJournalModal("", "")
				}
			})

			if ui.SearchField(c.Key("j-search"), &a.journalQuery).
				Placeholder("筛选鸟种、地点或文字记录…").Label("筛选日志").Changed() {
				a.refreshJournalList()
			}

			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "共 %d 条记录 · 已解锁 %d 种", len(a.filteredEntries), len(a.store.UnlockedBirds())).
					FontSize(12).TextColor(t.TextMuted)
				ui.Row(c).Gap(6).Children(func() {
					if ui.Button(c.Key("j-export-csv"), "导出 CSV").Clicked() {
						a.ExportJournalDialog("csv")
					}
					if ui.Button(c.Key("j-export-json"), "导出 JSON").Clicked() {
						a.ExportJournalDialog("json")
					}
				})
			})
		})

		a.journalList.Key = func(i int) any { return a.filteredEntries[i].ID }
		a.journalList.Label = func(i int) string { return a.filteredEntries[i].ComName }
		a.journalList.Selected = &a.journalSelected

		list := ui.List(c.Key("j-list"), &a.journalList, len(a.filteredEntries)).Grow(1).FillWidth()
		list.Rows(func(row ui.ListRow) {
			e := a.filteredEntries[row.Index]
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

			ui.Column(rc).Padding(10, 14).Gap(4).Background(bg).Children(func() {
				ui.Row(rc).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(rc, e.ComName).Bold().TextColor(textColor).SingleLine().Grow(1)
					ui.Textf(rc, "%d 只", max(1, e.Count)).FontSize(11.5).TextColor(mutedColor)
				})
				loc := fallbackText(e.Location, "未填地点")
				ui.Textf(rc, "%s · %s", e.ObservedAt, loc).FontSize(11.5).TextColor(mutedColor).SingleLine()

				if len(e.PhotoPaths) > 0 || e.AudioPath != "" {
					ui.Row(rc).Gap(6).Children(func() {
						if len(e.PhotoPaths) > 0 {
							b := ui.Badge(rc, fmt.Sprintf("%d 张照片", len(e.PhotoPaths)))
							if row.Selected() {
								b.Background(ui.RGBA(255, 255, 255, 0.2)).TextColor(t.AccentText)
							}
						}
						if e.AudioPath != "" {
							b := ui.Badge(rc, "含现场录音")
							if row.Selected() {
								b.Background(ui.RGBA(255, 255, 255, 0.2)).TextColor(t.AccentText)
							}
						}
					})
				}
			})
		}).Children(func() {
			if len(a.filteredEntries) == 0 {
				ui.Column(c).Fill().Center().Padding(24).Gap(10).Children(func() {
					ui.Icon(c, iconJournal).FontSize(28).TextColor(t.TextMuted)
					ui.Text(c, "暂无观鸟记录").Bold()
					ui.Text(c, "记录今天看到的鸟种，首次记录会自动解锁该鸟种").FontSize(12.5).TextColor(t.TextMuted)
					if ui.PrimaryButton(c.Key("j-empty-add"), "写第一条日志").Clicked() {
						a.OpenNewJournalModal("", "")
					}
				})
			}
		})

		if list.Changed() && a.journalSelected >= 0 && a.journalSelected < len(a.filteredEntries) {
			a.journalActiveID = a.filteredEntries[a.journalSelected].ID
			a.journalPhotoIndex = 0
		}
	})
}

func (a *AppState) viewJournalDetailPane(c *ui.Context, t *ui.Theme) {
	if a.journalActiveID == "" {
		ui.Column(c).Fill().Center().Gap(8).Padding(32).Children(func() {
			ui.Icon(c, iconJournal).FontSize(32).TextColor(t.TextMuted)
			ui.Text(c, "选择一条观鸟日志查看详情").FontSize(15).Bold()
			ui.Text(c, "可上传个人摄影图片、用麦克风录制鸟鸣并自动累计解锁鸟种").FontSize(12.5).TextColor(t.TextMuted)
		})
		return
	}

	var cur *model.JournalEntry
	for i := range a.filteredEntries {
		if a.filteredEntries[i].ID == a.journalActiveID {
			cur = &a.filteredEntries[i]
			break
		}
	}
	if cur == nil {
		return
	}
	entry := *cur

	ui.Scroll(c.Key("j-detail-scroll")).Fill().Padding(20).Gap(16).Children(func() {
		ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Start).Gap(12).Children(func() {
			ui.Column(c).Grow(1).Gap(4).Children(func() {
				ui.Text(c, entry.ComName).FontSize(22).Bold()
				ui.Text(c, entry.SciName).Italic().FontSize(14).TextColor(t.TextMuted)
			})
			ui.Row(c).Gap(8).Children(func() {
				if _, ok := a.taxonByCode[entry.SpeciesCode]; ok {
					if ui.Button(c.Key("j-goto-guide"), "前往图鉴").Clicked() {
						a.OpenSpeciesInGuide(entry.SpeciesCode)
					}
				}
				if ui.Button(c.Key("j-edit-btn"), "编辑").Clicked() {
					a.openEditJournalModal(entry)
				}
				if ui.Button(c.Key("j-del-btn"), "删除").TextColor(t.Danger).Clicked() {
					a.jDeleteTargetID = entry.ID
					a.jDeleteConfirm = true
				}
			})
		})

		ui.Row(c).Gap(8).Wrap().Children(func() {
			infoChip(c, t, "观测时间", entry.ObservedAt)
			infoChip(c, t, "地点", fallbackText(entry.Location, "未填写"))
			infoChip(c, t, "数量", fmt.Sprintf("%d 只", max(1, entry.Count)))
			if entry.FamilyName != "" {
				infoChip(c, t, "科", entry.FamilyName)
			}
		})

		// 用户自己上传的照片画廊
		ui.Column(c).FillWidth().Padding(14).Gap(10).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "个人观测照片").Bold()
				ui.Textf(c, "%d 张", len(entry.PhotoPaths)).FontSize(12).TextColor(t.TextMuted)
			})
			if len(entry.PhotoPaths) == 0 {
				ui.Text(c, "本条记录未附加照片，点击右上角「编辑」可补充图片。").FontSize(12.5).TextColor(t.TextMuted)
			} else {
				if a.journalPhotoIndex >= len(entry.PhotoPaths) {
					a.journalPhotoIndex = 0
				}
				p := entry.PhotoPaths[a.journalPhotoIndex]
				imgKey := "local:" + p
				a.loadImageAsync(imgKey, p, true)
				if src, ok := a.imgCache[imgKey]; ok {
					ui.Image(c, src).FillWidth().Height(280).Fit(ui.Contain).Radius(8).Background(t.Background).Label(entry.ComName)
				} else if errMsg := a.imgErr[imgKey]; errMsg != "" {
					ui.Textf(c, "照片读取失败: %s", errMsg).FontSize(12).TextColor(t.Danger)
				} else {
					ui.Box(c).FillWidth().Height(200).Center().Radius(8).Background(t.Background).Children(func() {
						ui.Spinner(c).Label("读取照片中")
					})
				}

				if len(entry.PhotoPaths) > 1 {
					ui.Row(c).Gap(6).Wrap().Children(func() {
						for i := range entry.PhotoPaths {
							btn := ui.Button(c.Key(fmt.Sprintf("j-p-%d", i)), fmt.Sprintf("照片 %d", i+1))
							if i == a.journalPhotoIndex {
								btn.Background(t.Accent).TextColor(t.AccentText)
							}
							if btn.Clicked() {
								a.journalPhotoIndex = i
							}
						}
					})
				}
			}
		})

		// 用户自己录制/上传的录音
		ui.Column(c).FillWidth().Padding(14).Gap(10).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Text(c, "个人现场录音").Bold()
			if entry.AudioPath == "" {
				ui.Text(c, "本条记录未附带现场录音。").FontSize(12.5).TextColor(t.TextMuted)
			} else {
				st := a.player.Status()
				isThis := st.SourceKey == entry.AudioPath
				ui.Row(c).FillWidth().Padding(10, 12).Gap(10).AlignItems(ui.Center).Radius(8).Background(t.Background).Children(func() {
					btnIcon := iconPlay
					btnLabel := "播放录音"
					if isThis && st.Playing {
						btnIcon = iconPause
						btnLabel = "暂停录音"
					}
					if ui.Button(c.Key("j-play-audio"), "").Label(btnLabel).Children(func() {
						ui.Icon(c, btnIcon).FontSize(13)
					}).Clicked() {
						if isThis && st.Active {
							a.player.TogglePause()
						} else {
							a.player.PlayFile(entry.AudioPath, entry.ComName+" · 个人录音", entry.ObservedAt, func() {
								a.updateUI(func() {})
							})
						}
					}
					ui.Column(c).Grow(1).Gap(2).Children(func() {
						ui.Text(c, filepath.Base(entry.AudioPath)).Bold().FontSize(13).SingleLine()
						ui.Text(c, "本地保存的观鸟现场录音").FontSize(11.5).TextColor(t.TextMuted)
					})
				})
			}
		})

		// 文字观察记录
		ui.Column(c).FillWidth().Padding(14).Gap(8).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Selectable().Children(func() {
			ui.Text(c, "观察笔记").Bold()
			if strings.TrimSpace(entry.Notes) != "" {
				ui.Text(c, entry.Notes).LineHeight(1.6)
			} else {
				ui.Text(c, "暂无文字观察笔记。").FontSize(12.5).TextColor(t.TextMuted)
			}
		})
	})
}

func (a *AppState) viewJournalModal(c *ui.Context, t *ui.Theme) {
	if !a.journalModalOpen {
		return
	}
	recStatus := a.recorder.Status()
	if recStatus.Recording {
		c.After(80 * time.Millisecond)
	}

	dlg := ui.Modal(c, &a.journalModalOpen, func() {
		ui.Column(c).Width(620).MaxHeight(680).Gap(12).Children(func() {
			title := "新建观鸟日志"
			if a.jEditID != "" {
				title = "编辑观鸟日志"
			}
			ui.Text(c, title).FontSize(18).Bold()

			ui.Scroll(c.Key("j-modal-scroll")).Grow(1).FillWidth().Padding(2, 6, 2, 2).Gap(12).Children(func() {
				// 鸟种选择
				ui.Field(c, "鸟种（支持中文名、拉丁学名或代码筛选）", func() {
					if len(a.journalSpeciesOpts) > 0 {
						ui.Combobox(c.Key("j-species-combo"), &a.jDraftSpecies, a.journalSpeciesOpts).
							Placeholder("选择或搜索鸟种…").FillWidth()
					} else {
						ui.TextInput(c.Key("j-species-input"), &a.jDraftSpecies).
							Placeholder("输入鸟种中文名或学名（如：普通翠鸟）").FillWidth()
					}
				})

				// 时间 / 地点 / 数量
				ui.Row(c).Gap(10).FillWidth().Children(func() {
					ui.Field(c, "观测时间", func() {
						ui.TextInput(c.Key("j-obs-dt"), &a.jDraftObservedAt).FillWidth()
					}).Width(170)
					ui.Field(c, "观测地点", func() {
						ui.TextInput(c.Key("j-obs-loc"), &a.jDraftLocation).Placeholder("如：北京颐和园西堤").FillWidth()
					}).Grow(1)
					ui.Field(c, "数量", func() {
						ui.TextInput(c.Key("j-obs-cnt"), &a.jDraftCount).FillWidth()
					}).Width(84)
				})

				// 文字记录
				ui.Field(c, "文字记录", func() {
					ui.TextArea(c.Key("j-obs-notes"), &a.jDraftNotes).Height(96).FillWidth()
				}).Description("可填写生境、羽色特征或行为描述")

				// 照片上传区
				ui.Column(c).Padding(12).Gap(8).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "现场照片").Bold().FontSize(13)
						if ui.Button(c.Key("j-pick-photos"), "").Children(func() {
							ui.Icon(c, iconImage).FontSize(14)
							ui.Text(c, "选择图片…").SingleLine()
						}).Clicked() {
							a.pickJournalPhotosAsync()
						}
					})
					if len(a.jDraftPhotos) == 0 {
						ui.Text(c, "尚未添加图片（支持 PNG / JPG / WebP / BMP）").FontSize(12).TextColor(t.TextMuted)
					} else {
						for i, p := range a.jDraftPhotos {
							idx := i
							ui.Row(c.Key(fmt.Sprintf("j-draft-p-%d", i))).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
								ui.Text(c, filepath.Base(p)).FontSize(12.5).SingleLine().Grow(1)
								ui.Button(c.Key(fmt.Sprintf("j-rm-p-%d", i)), "移除").OnClick(func() {
									a.jDraftPhotos = append(a.jDraftPhotos[:idx], a.jDraftPhotos[idx+1:]...)
								})
							})
						}
					}
				})

				// 现场录音按键 + 导入音频
				ui.Column(c).Padding(12).Gap(10).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "鸟鸣现场录音").Bold().FontSize(13)
						ui.Row(c).Gap(8).Children(func() {
							if !recStatus.Recording {
								recBtn := ui.Button(c.Key("j-rec-start"), "").Children(func() {
									ui.Icon(c, iconMic).FontSize(14)
									ui.Text(c, "开始录音").SingleLine()
								})
								if recBtn.Clicked() {
									if err := a.recorder.Start(); err != nil {
										a.jDraftError = err.Error()
									} else {
										a.jDraftError = ""
									}
								}
							} else {
								stopBtn := ui.PrimaryButton(c.Key("j-rec-stop"), "").Background(t.Danger).Children(func() {
									ui.Icon(c, iconStop).FontSize(13)
									ui.Text(c, "停止录音").SingleLine()
								})
								if stopBtn.Clicked() {
									dst := filepath.Join(a.store.DataDir(), "journal", "media", fmt.Sprintf("rec_%d.wav", time.Now().UnixMilli()))
									if err := a.recorder.Stop(dst); err != nil {
										a.jDraftError = err.Error()
									} else {
										a.jDraftAudio = dst
										a.jDraftError = ""
									}
								}
							}

							if ui.Button(c.Key("j-pick-audio"), "导入音频…").Disabled(recStatus.Recording).Clicked() {
								a.pickJournalAudioAsync()
							}
						})
					})

					if recStatus.Recording {
						ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
							ui.Textf(c, "正在录音 %s", formatDuration(recStatus.Elapsed)).FontSize(12.5).Bold().TextColor(t.Danger)
							ui.Meter(c, recStatus.Level, 0, 1, &ui.MeterLevels{Warning: 0.75, Critical: 0.92}).
								Label("录音电平").Grow(1)
						})
					} else if a.jDraftAudio != "" {
						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Textf(c, "已就绪：%s", filepath.Base(a.jDraftAudio)).FontSize(12.5).Grow(1).SingleLine()
							ui.Row(c).Gap(6).Children(func() {
								if ui.Button(c.Key("j-preview-audio"), "试听").Clicked() {
									a.player.PlayFile(a.jDraftAudio, "录音试听", filepath.Base(a.jDraftAudio), func() {
										a.updateUI(func() {})
									})
								}
								if ui.Button(c.Key("j-clear-audio"), "清除").Clicked() {
									a.jDraftAudio = ""
								}
							})
						})
					} else {
						ui.Text(c, "点击“开始录音”直接通过麦克风录制，或导入 WAV / MP3 / OGG / FLAC / M4A").
							FontSize(12).TextColor(t.TextMuted)
					}
				})

				if a.jDraftError != "" {
					ui.Text(c, a.jDraftError).FontSize(12.5).TextColor(t.Danger)
				}
			})

			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				if ui.Button(c.Key("j-modal-cancel"), "取消").Clicked() {
					a.recorder.Cancel()
					a.journalModalOpen = false
				}
				if ui.PrimaryButton(c.Key("j-modal-save"), "保存记录").Disabled(recStatus.Recording).Clicked() {
					a.saveJournalDraft(c)
				}
			})
		})
	})

	if dlg.Dismissed() && a.recorder.Status().Recording {
		// 录音过程中避免意外点空白处中断录音
		a.journalModalOpen = true
	}
}

func (a *AppState) saveJournalDraft(c *ui.Context) {
	code, comName, sciName, order, family := a.resolveDraftSpecies()
	if code == "" || comName == "" {
		a.jDraftError = "请先选择或输入一个鸟种"
		return
	}
	cnt, err := strconv.Atoi(strings.TrimSpace(a.jDraftCount))
	if err != nil || cnt <= 0 {
		cnt = 1
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
		Count:       cnt,
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
		a.journalActiveID = saved.ID
		a.journalModalOpen = false
		a.refreshJournalList()
		a.refreshStorageStats()
		if newlyUnlocked {
			c.Toast(fmt.Sprintf("已解锁新鸟种：%s（累计已解锁 %d 种）", saved.ComName, len(a.store.UnlockedBirds())))
		} else {
			c.Toast("已保存观鸟日志")
		}
	} else {
		saved, err := a.store.UpdateJournalEntry(entry)
		if err != nil {
			a.jDraftError = err.Error()
			return
		}
		a.journalActiveID = saved.ID
		a.journalModalOpen = false
		a.refreshJournalList()
		a.refreshStorageStats()
		c.Toast("已更新观鸟日志")
	}
}

func (a *AppState) viewJournalDeleteDialog(c *ui.Context) {
	switch ui.AlertDialog(c, &a.jDeleteConfirm, "删除这条观鸟日志？", "删除后不可恢复；若该鸟种已无其他记录，解锁状态也会同步移除。", "取消", "确认删除") {
	case 1:
		_ = a.store.DeleteJournalEntry(a.jDeleteTargetID)
		a.jDeleteTargetID = ""
		a.refreshJournalList()
		a.refreshStorageStats()
		c.Toast("已删除观鸟日志")
	}
}

func (a *AppState) pickJournalPhotosAsync() {
	go func() {
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
	}()
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

// ExportJournalDialog 打开系统保存对话框并导出观鸟日志（支持 "csv" 或 "json"）。
func (a *AppState) ExportJournalDialog(format string) {
	defaultName := "birdview_journal." + format
	go func() {
		dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      a.win,
			Title:       "导出观鸟日志",
			DefaultPath: defaultName,
		})
		if err != nil || dest == "" {
			return
		}
		if format == "csv" {
			err = a.store.ExportJournalCSV(dest)
		} else {
			err = a.store.ExportJournalJSON(dest)
		}
		a.updateUI(func() {
			if err != nil {
				a.pendingToast = fmt.Sprintf("导出失败: %v", err)
			} else {
				a.pendingToast = "已导出观鸟日志至 " + filepath.Base(dest)
			}
		})
	}()
}

func (a *AppState) viewLifelist(c *ui.Context, t *ui.Theme) {
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
		fam := fallbackText(u.FamilyName, "未分类")
		familyCount[fam]++
	}
	type famStat struct {
		name  string
		count int
	}
	var famList []famStat
	for k, v := range familyCount {
		famList = append(famList, famStat{k, v})
	}
	sort.Slice(famList, func(i, j int) bool {
		if famList[i].count != famList[j].count {
			return famList[i].count > famList[j].count
		}
		return famList[i].name < famList[j].name
	})

	ui.Scroll(c.Key("lifelist-scroll")).Fill().Padding(20).Gap(16).Children(func() {
		// 统计概览卡片
		ui.Column(c).FillWidth().Padding(16).Gap(12).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, "已解锁鸟种").FontSize(18).Bold()
					ui.Textf(c, "累计记录 %d 次观测，跨越 %d 个科", len(entries), len(familyCount)).
						FontSize(12.5).TextColor(t.TextMuted)
				})
				ui.PrimaryButton(c.Key("ll-add-btn"), "记录新鸟种").OnClick(func() {
					a.OpenNewJournalModal("", "")
				})
			})

			ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "%d / %d", len(unlocked), chinaTotal).FontSize(16).Bold().FontFeatures("tnum")
				ui.Progress(c, progress).Label("中国鸟类解锁进度").Grow(1)
				ui.Textf(c, "%.1f%%", progress*100).FontSize(12.5).TextColor(t.TextMuted).FontFeatures("tnum")
			})

			if len(famList) > 0 {
				ui.Row(c).Gap(6).Wrap().Children(func() {
					maxChips := min(8, len(famList))
					for _, fs := range famList[:maxChips] {
						infoChip(c, t, fs.name, fmt.Sprintf("%d 种", fs.count))
					}
				})
			}
		})

		ui.SearchField(c.Key("ll-search"), &a.lifelistFilter).Placeholder("搜索已解锁鸟种名称或首次发现地…").Label("筛选已解锁鸟种").Width(360)

		q := strings.ToLower(strings.TrimSpace(a.lifelistFilter))
		var visible []model.UnlockedBird
		for _, u := range unlocked {
			if q == "" ||
				strings.Contains(strings.ToLower(u.ComName), q) ||
				strings.Contains(strings.ToLower(u.SciName), q) ||
				strings.Contains(strings.ToLower(u.FirstLocation), q) {
				visible = append(visible, u)
			}
		}

		if len(visible) == 0 {
			ui.Column(c).FillWidth().Padding(32).Center().Gap(8).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Icon(c, iconAward).FontSize(28).TextColor(t.TextMuted)
				ui.Text(c, "尚未解锁鸟种").Bold()
				ui.Text(c, "在「观鸟日志」中添加观测记录后，对应鸟种会自动在此点亮").FontSize(12.5).TextColor(t.TextMuted)
			})
			return
		}

		// 已解锁鸟种卡片列表
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			for _, u := range visible {
				bird := u
				ui.Row(c.Key("ll-" + bird.SpeciesCode)).FillWidth().Padding(12, 14).Gap(12).
					AlignItems(ui.Center).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
					if bird.CoverPhotoPath != "" {
						imgKey := "local:" + bird.CoverPhotoPath
						a.loadImageAsync(imgKey, bird.CoverPhotoPath, true)
						if src, ok := a.imgCache[imgKey]; ok {
							ui.Image(c, src).Size(64, 48).Fit(ui.Cover).Radius(6).Label(bird.ComName)
						} else {
							ui.Box(c).Size(64, 48).Center().Radius(6).Background(t.Background).Children(func() {
								ui.Icon(c, iconImage).FontSize(18).TextColor(t.TextMuted)
							})
						}
					} else {
						ui.Box(c).Size(64, 48).Center().Radius(6).Background(t.Background).Children(func() {
							ui.Icon(c, iconAward).FontSize(20).TextColor(t.Accent)
						})
					}

					ui.Column(c).Grow(1).Gap(2).Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Text(c, bird.ComName).FontSize(15).Bold()
							ui.Text(c, bird.SciName).Italic().FontSize(12.5).TextColor(t.TextMuted)
						})
						ui.Textf(c, "首次记录：%s · %s · 累计 %d 次观测",
							bird.FirstSeenAt, fallbackText(bird.FirstLocation, "未填地点"), bird.TotalSightings).
							FontSize(12).TextColor(t.TextMuted)
					})

					ui.Row(c).Gap(6).Children(func() {
						if ui.Button(c.Key("ll-log-"+bird.SpeciesCode), "查看日志").Clicked() {
							a.journalQuery = bird.ComName
							a.refreshJournalList()
							a.page = "journal"
						}
						if _, ok := a.taxonByCode[bird.SpeciesCode]; ok {
							if ui.Button(c.Key("ll-guide-"+bird.SpeciesCode), "图鉴").Clicked() {
								a.OpenSpeciesInGuide(bird.SpeciesCode)
							}
						}
					})
				})
			}
		})
	})
}
