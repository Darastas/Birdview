package appui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Darastas/Birdview/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// encodeRGBAToPNG 将渲染后的 RGBA 分享图编码为 PNG 字节。
func encodeRGBAToPNG(img *image.RGBA) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// loadShareLocalImage 从磁盘同步解码用于排版的缩略照片（避免阻塞 UI，在后台协程调用）。
func loadShareLocalImage(path string, maxSide int) ui.ImageSource {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if bmp, derr := ui.DecodeBitmapScaled(data, maxSide); derr == nil {
		return bmp
	}
	if src, derr := ui.DecodeImage(data); derr == nil {
		return src
	}
	return nil
}

// RenderJournalEntryShareCard 自动排版单条观鸟日志精美观鸟手账明信片（2x 高清渲染）。
func RenderJournalEntryShareCard(entry model.JournalEntry, photos []ui.ImageSource) *image.RGBA {
	notes := strings.TrimSpace(entry.Notes)
	if notes == "" {
		notes = "记录下一次美好的野外邂逅，点亮属于自己的观鸟图鉴。"
	}
	hasPhoto := len(photos) > 0 && photos[0] != nil
	photoH := 0
	if hasPhoto {
		photoH = 276
	}
	noteLines := max(2, min(8, len([]rune(notes))/19+1))
	h := 248 + photoH + noteLines*24
	if entry.AudioPath != "" {
		h += 42
	}
	if h < 420 {
		h = 420
	}
	if h > 920 {
		h = 920
	}

	return ui.Render(func(c *ui.Context) {
		t := ui.LightTheme()
		c.SetTheme(t)
		ui.Box(c).Fill().Background(ui.Hex("#F3F5F1")).Padding(20).Children(func() {
			// 外层柔和温润手账画框卡片
			ui.Column(c).Fill().Background(ui.Hex("#FFFFFF")).Radius(24).
				Border(1, ui.RGBA(35, 61, 77, 0.08)).Clip().Children(func() {
				// 顶部照片区（或温润色块头图）
				if hasPhoto {
					ui.Box(c).FillWidth().Height(float32(photoH)).Background(ui.Hex("#E7ECE9")).Children(func() {
						if len(photos) == 1 {
							ui.Image(c, photos[0]).Fill().Fit(ui.Cover)
						} else {
							ui.Row(c).Fill().Gap(3).Children(func() {
								ui.Image(c, photos[0]).Grow(3).Basis(0).FillHeight().Fit(ui.Cover)
								ui.Column(c).Grow(2).Basis(0).FillHeight().Gap(3).Children(func() {
									for idx := 1; idx < min(len(photos), 3); idx++ {
										if photos[idx] != nil {
											ui.Image(c, photos[idx]).Grow(1).Basis(0).FillWidth().Fit(ui.Cover)
										}
									}
								})
							})
						}
						// 顶部渐变 + 左上角手账徽标
						ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(68).LinearGradient(ui.LinearGradient{
							From: ui.RGBA(12, 22, 20, 0.36), To: ui.Transparent, Angle: 180,
						})
						ui.Row(c).Absolute().Left(14).Top(14).Padding(5, 11).Radius(999).
							Background(ui.RGBA(255, 255, 255, 0.88)).Gap(5).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconBird).FontSize(13).TextColor(ui.Hex("#3C7E75"))
							ui.Text(c, "Birdview 观鸟手账").FontSize(11).FontWeight(600).TextColor(ui.Hex("#233D4D"))
						})
					})
				} else {
					ui.Row(c).FillWidth().Padding(18, 20, 12, 20).Background(ui.Hex("#EBF3F0")).
						Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconBird).FontSize(16).TextColor(ui.Hex("#3C7E75"))
							ui.Text(c, "Birdview 观鸟手账").FontSize(12).FontWeight(600).TextColor(ui.Hex("#2F655E"))
						})
						ui.Text(c, "FIELD NOTE").FontSize(10).FontWeight(600).LetterSpacing(1.2).TextColor(ui.Hex("#6B8E87"))
					})
				}

				// 正文排版内容区
				ui.Column(c).FillWidth().Grow(1).Padding(20, 22, 18, 22).Gap(12).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Baseline).Gap(8).Children(func() {
						ui.Text(c, entry.ComName).FontSize(26).FontWeight(700).TextColor(ui.Hex("#16232B")).SingleLine().Grow(1)
						ui.Row(c).Padding(4, 10).Radius(999).Background(ui.Hex("#EBF3F0")).Shrink(0).Children(func() {
							ui.Textf(c, "目击 %d 只", max(1, entry.Count)).FontSize(12).FontWeight(600).TextColor(ui.Hex("#2F655E"))
						})
					})

					if entry.SciName != "" {
						ui.Text(c, entry.SciName).FontSize(13.5).Italic().TextColor(ui.Hex("#64748B"))
					}

					// 分类 + 时间 + 城市元信息条
					taxLine := strings.TrimSpace(strings.Join([]string{entry.Order, entry.FamilyName}, " · "))
					taxLine = strings.Trim(taxLine, " ·")
					ui.Row(c).FillWidth().Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
						shareMetaChip(c, iconCalendar, model.FormatObsTime(entry.ObservedAt))
						shareMetaChip(c, iconMapPin, fallbackText(entry.Location, "野外现场"))
						if taxLine != "" {
							shareMetaChip(c, iconNavGuide, taxLine)
						}
					})

					// 细分割线
					ui.Box(c).FillWidth().Height(1).Background(ui.RGBA(22, 35, 43, 0.08))

					// 观察笔记正文
					ui.Column(c).FillWidth().Grow(1).Gap(6).Children(func() {
						ui.Text(c, "观察笔记").FontSize(11).FontWeight(600).LetterSpacing(0.8).TextColor(ui.Hex("#7D8E95"))
						ui.Text(c, notes).FontSize(14).LineHeight(1.68).TextColor(ui.Hex("#2D3C45"))
					})

					if entry.AudioPath != "" {
						ui.Row(c).FillWidth().Padding(8, 12).Radius(10).Background(ui.Hex("#F4F7F6")).
							Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconMic).FontSize(13).TextColor(ui.Hex("#3C7E75"))
							ui.Text(c, "附有现场鸟鸣实录音频").FontSize(12).FontWeight(500).TextColor(ui.Hex("#3C7E75"))
						})
					}

					// 底部落款印记
					ui.Row(c).FillWidth().Padding(8, 0, 0, 0).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "用双筒丈量山河，以手账留住飞羽").FontSize(11).TextColor(ui.Hex("#8B9BA3"))
						ui.Text(c, time.Now().Format("2006.01.02")).FontSize(11).FontFeatures("tnum").TextColor(ui.Hex("#8B9BA3"))
					})
				})
			})
		})
	}, 420, h, 2)
}

// RenderJournalSummaryShareCard 自动排版观鸟日志汇总海报长图（含统计与精选记录卡片）。
func RenderJournalSummaryShareCard(entries []model.JournalEntry, unlockedCount int, thumbs map[string]ui.ImageSource) *image.RGBA {
	shown := entries
	if len(shown) > 6 {
		shown = shown[:6]
	}
	h := 245 + len(shown)*92
	if len(shown) == 0 {
		h = 360
	}
	if h > 980 {
		h = 980
	}

	return ui.Render(func(c *ui.Context) {
		t := ui.LightTheme()
		c.SetTheme(t)
		ui.Box(c).Fill().Background(ui.Hex("#F1F5F2")).Padding(18).Children(func() {
			ui.Column(c).Fill().Background(ui.Hex("#FFFFFF")).Radius(24).
				Border(1, ui.RGBA(35, 61, 77, 0.08)).Clip().Children(func() {
				// 顶部草木青渐变封面头
				ui.Column(c).FillWidth().Padding(22, 22, 18, 22).Gap(14).LinearGradient(ui.LinearGradient{
					From: ui.Hex("#4E9387"), To: ui.Hex("#356E66"), Angle: 145,
				}).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconBird).FontSize(18).TextColor(ui.RGB(255, 255, 255))
							ui.Text(c, "Birdview 观鸟图志").FontSize(15).FontWeight(700).TextColor(ui.RGB(255, 255, 255))
						})
						ui.Text(c, time.Now().Format("2006年01月")).FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.85))
					})

					ui.Row(c).FillWidth().Gap(10).Children(func() {
						shareStatBlock(c, fmt.Sprintf("%d", unlockedCount), "已解锁鸟种")
						shareStatBlock(c, fmt.Sprintf("%d", len(entries)), "累计观测记录")
						latestCity := "野外探索"
						if len(entries) > 0 && strings.TrimSpace(entries[0].Location) != "" {
							latestCity = entries[0].Location
						}
						shareStatBlock(c, latestCity, "最近足迹")
					})
				})

				// 记录精选列表
				ui.Column(c).FillWidth().Grow(1).Padding(18, 20, 16, 20).Gap(10).Children(func() {
					ui.Text(c, "近期观鸟足迹").FontSize(12).FontWeight(600).LetterSpacing(0.6).TextColor(ui.Hex("#6C8088"))
					if len(shown) == 0 {
						ui.Box(c).FillWidth().Padding(28).Center().Children(func() {
							ui.Text(c, "暂无观鸟日志，带上望远镜去记录第一只鸟吧").FontSize(13).TextColor(ui.Hex("#7D8E95"))
						})
					} else {
						for i, item := range shown {
							e := item
							ui.Row(c.Key(i)).FillWidth().Padding(10, 12).Radius(14).Background(ui.Hex("#F6F8F7")).
								Gap(12).AlignItems(ui.Center).Children(func() {
								if src := thumbs[e.ID]; src != nil {
									ui.Image(c, src).Size(54, 54).Radius(11).Fit(ui.Cover).Shrink(0)
								} else {
									ui.Box(c).Size(54, 54).Radius(11).Background(ui.Hex("#E3ECE8")).Center().Shrink(0).Children(func() {
										ui.Icon(c, iconBird).FontSize(22).TextColor(ui.Hex("#4D8B80"))
									})
								}
								ui.Column(c).Grow(1).Basis(0).Gap(3).Children(func() {
									ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Baseline).Gap(6).Children(func() {
										ui.Text(c, e.ComName).FontSize(15).FontWeight(600).TextColor(ui.Hex("#1A2930")).SingleLine().Grow(1)
										ui.Textf(c, "%d 只", max(1, e.Count)).FontSize(12).FontWeight(600).TextColor(ui.Hex("#3C7E75")).Shrink(0)
									})
									ui.Text(c, fmt.Sprintf("%s · %s", model.FormatObsTime(e.ObservedAt), fallbackText(e.Location, "野外现场"))).
										FontSize(11.5).TextColor(ui.Hex("#6B7F88")).SingleLine()
									if strings.TrimSpace(e.Notes) != "" {
										ui.Text(c, strings.TrimSpace(e.Notes)).FontSize(11.5).TextColor(ui.Hex("#4A5D66")).SingleLine()
									}
								})
							})
						}
					}

					ui.Row(c).FillWidth().Padding(8, 2, 0, 2).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Generated by Birdview").FontSize(11).TextColor(ui.Hex("#8FA0A8"))
						if len(entries) > len(shown) {
							ui.Textf(c, "共 %d 条记录 · 展示最新 %d 条", len(entries), len(shown)).FontSize(11).TextColor(ui.Hex("#8FA0A8"))
						}
					})
				})
			})
		})
	}, 420, h, 2)
}

// RenderSeasonalBirdShareCard 自动排版单鸟「时令观鸟指南」分享图（含 1-12 月物候条、地点与羽色变化）。
func RenderSeasonalBirdShareCard(bird model.SeasonalBird, zone model.EcoZone, month int, regionName string, hero ui.ImageSource) *image.RGBA {
	phase := bird.PhaseFor(zone, month)
	cal := bird.MonthCalendar(zone)
	hasImg := hero != nil
	h := 650
	if !hasImg {
		h = 520
	}

	return ui.Render(func(c *ui.Context) {
		t := ui.LightTheme()
		c.SetTheme(t)
		ui.Box(c).Fill().Background(ui.Hex("#F1F5F3")).Padding(18).Children(func() {
			ui.Column(c).Fill().Background(ui.Hex("#FFFFFF")).Radius(24).
				Border(1, ui.RGBA(35, 61, 77, 0.08)).Clip().Children(func() {
				if hasImg {
					ui.Box(c).FillWidth().Height(210).Background(ui.Hex("#E3ECE8")).Children(func() {
						ui.Image(c, hero).Fill().Fit(ui.Cover)
						ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(68).LinearGradient(ui.LinearGradient{
							From: ui.RGBA(12, 22, 20, 0.42), To: ui.Transparent, Angle: 180,
						})
						ui.Row(c).Absolute().Left(14).Top(14).Padding(5, 11).Radius(999).
							Background(ui.RGBA(255, 255, 255, 0.90)).Gap(5).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconNavSeasonal).FontSize(13).TextColor(ui.Hex("#3C7E75"))
							ui.Textf(c, "%d 月 · %s观鸟指南", month, regionName).FontSize(11).FontWeight(600).TextColor(ui.Hex("#1E3632"))
						})
					})
				} else {
					ui.Row(c).FillWidth().Padding(16, 20).Background(ui.Hex("#468B7E")).
						Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Textf(c, "%d 月 · %s观鸟指南", month, regionName).FontSize(13).FontWeight(600).TextColor(ui.RGB(255, 255, 255))
						ui.Text(c, "Birdview").FontSize(12).FontWeight(600).TextColor(ui.RGBA(255, 255, 255, 0.8))
					})
				}

				ui.Column(c).FillWidth().Grow(1).Padding(18, 20, 16, 20).Gap(11).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Text(c, bird.ComName).FontSize(24).FontWeight(700).TextColor(ui.Hex("#16232B")).SingleLine()
							ui.Text(c, bird.SciName).FontSize(12.5).Italic().TextColor(ui.Hex("#6B7F88")).SingleLine()
						})
						bg, fg := seasonalPhaseColors(phase, false)
						statusPill(c, phase.Label(), bg, fg)
					})

					// 1-12 月物候月历条
					ui.Column(c).FillWidth().Gap(5).Children(func() {
						ui.Textf(c, "%s全年物候月历（当前高亮 %d 月）", model.EcoZoneLabel(zone), month).
							FontSize(11).FontWeight(600).TextColor(ui.Hex("#6B7F88"))
						ui.Row(c).FillWidth().Gap(3).Children(func() {
							for m := 1; m <= 12; m++ {
								ph := cal[m-1]
								cellBg, cellFg := seasonalPhaseColors(ph, false)
								if ph == model.PhaseAbsent {
									cellBg = ui.Hex("#EEF2F4")
									cellFg = ui.Hex("#9AAAB2")
								}
								cell := ui.Column(c.Key(m)).Grow(1).Basis(0).Height(30).Radius(7).
									Background(cellBg).AlignItems(ui.Center).Justify(ui.Center)
								if m == month {
									cell.Border(1.8, ui.Hex("#1E3632"))
								}
								cell.Children(func() {
									ui.Textf(c, "%d", m).FontSize(10.5).FontWeight(600).TextColor(cellFg)
								})
							}
						})
					})

					shareInfoSection(c, "在哪里看 · 推荐点位", bird.RecommendedSpots(zone))
					shareInfoSection(c, "羽色与季节变化", bird.PlumageChanges)
					shareInfoSection(c, "最佳时段", bird.BestTimeOfDay)

					ui.Row(c).FillWidth().Padding(4, 0, 0, 0).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Birdview 时令观鸟指南").FontSize(11).TextColor(ui.Hex("#8FA0A8"))
						ui.Text(c, time.Now().Format("2006.01.02")).FontSize(11).TextColor(ui.Hex("#8FA0A8"))
					})
				})
			})
		})
	}, 420, h, 2)
}

// RenderMemoListShareCard 自动排版「我的追鸟备忘清单」分享图。
func RenderMemoListShareCard(memos []model.BirdingMemo, regionName string, month int) *image.RGBA {
	shown := memos
	if len(shown) > 7 {
		shown = shown[:7]
	}
	h := 220 + max(1, len(shown))*82
	if h > 920 {
		h = 920
	}
	return ui.Render(func(c *ui.Context) {
		t := ui.LightTheme()
		c.SetTheme(t)
		ui.Box(c).Fill().Background(ui.Hex("#F3F6F3")).Padding(18).Children(func() {
			ui.Column(c).Fill().Background(ui.Hex("#FFFFFF")).Radius(24).
				Border(1, ui.RGBA(35, 61, 77, 0.08)).Clip().Children(func() {
				ui.Column(c).FillWidth().Padding(20, 22, 16, 22).Gap(6).LinearGradient(ui.LinearGradient{
					From: ui.Hex("#4D9488"), To: ui.Hex("#356E66"), Angle: 145,
				}).Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "我的追鸟备忘清单").FontSize(20).FontWeight(700).TextColor(ui.RGB(255, 255, 255))
						ui.Textf(c, "%d 月 · %s", month, regionName).FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.88))
					})
					ui.Textf(c, "共计划 %d 种目标鸟 · 与大自然如约而至", len(memos)).
						FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.82))
				})

				ui.Column(c).FillWidth().Grow(1).Padding(16, 20, 14, 20).Gap(10).Children(func() {
					if len(shown) == 0 {
						ui.Box(c).FillWidth().Padding(24).Center().Children(func() {
							ui.Text(c, "还没有添加追鸟备忘，去观鸟指南收藏心仪目标吧").FontSize(13).TextColor(ui.Hex("#7D8E95"))
						})
					} else {
						for i, m := range shown {
							ui.Row(c.Key(i)).FillWidth().Padding(10, 12).Radius(14).Background(ui.Hex("#F6F8F7")).
								Gap(10).AlignItems(ui.Center).Children(func() {
								dotBg := ui.Hex("#4D9488")
								if m.Completed {
									dotBg = ui.Hex("#94A3B8")
								}
								ui.Box(c).Size(22, 22).Radius(11).Background(dotBg).Center().Shrink(0).Children(func() {
									if m.Completed {
										ui.Icon(c, iconCheck).FontSize(12).TextColor(ui.RGB(255, 255, 255))
									} else {
										ui.Icon(c, iconBird).FontSize(12).TextColor(ui.RGB(255, 255, 255))
									}
								})
								ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
									ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Baseline).Gap(6).Children(func() {
										ui.Text(c, m.ComName).FontSize(15).FontWeight(600).TextColor(ui.Hex("#1A2930")).SingleLine().Grow(1)
										ui.Text(c, fallbackText(m.TargetDate, fmt.Sprintf("%d月", m.TargetMonth))).
											FontSize(11.5).FontWeight(500).TextColor(ui.Hex("#3C7E75")).Shrink(0)
									})
									ui.Text(c, fallbackText(m.Location, m.RegionName)).FontSize(11.5).TextColor(ui.Hex("#64748B")).SingleLine()
									if strings.TrimSpace(m.Notes) != "" {
										ui.Text(c, m.Notes).FontSize(11).TextColor(ui.Hex("#52646D")).SingleLine()
									}
								})
							})
						}
					}
					ui.Row(c).FillWidth().Padding(6, 2, 0, 2).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Birdview · 观鸟指南备忘").FontSize(11).TextColor(ui.Hex("#8FA0A8"))
						ui.Text(c, time.Now().Format("2006.01.02")).FontSize(11).TextColor(ui.Hex("#8FA0A8"))
					})
				})
			})
		})
	}, 420, h, 2)
}

func shareMetaChip(c *ui.Context, icon *ui.SVG, text string) {
	ui.Row(c).Padding(4, 9).Radius(999).Background(ui.Hex("#F3F6F5")).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Icon(c, icon).FontSize(11).TextColor(ui.Hex("#4A7A73"))
		ui.Text(c, text).FontSize(11.5).TextColor(ui.Hex("#475861")).SingleLine()
	})
}

func shareStatBlock(c *ui.Context, val, label string) {
	ui.Column(c).Grow(1).Basis(0).Padding(10, 8).Radius(14).Background(ui.RGBA(255, 255, 255, 0.16)).
		AlignItems(ui.Center).Gap(2).Children(func() {
		ui.Text(c, val).FontSize(17).FontWeight(700).TextColor(ui.RGB(255, 255, 255)).SingleLine()
		ui.Text(c, label).FontSize(10.5).TextColor(ui.RGBA(255, 255, 255, 0.84)).SingleLine()
	})
}

func shareInfoSection(c *ui.Context, title, body string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	ui.Column(c).FillWidth().Padding(10, 12).Radius(12).Background(ui.Hex("#F6F8F7")).Gap(4).Children(func() {
		ui.Text(c, title).FontSize(11).FontWeight(600).TextColor(ui.Hex("#3C7E75"))
		ui.Text(c, body).FontSize(12).LineHeight(1.55).TextColor(ui.Hex("#33444D"))
	})
}

// openRenderedShareCard 将自动排版的 PNG 写入临时缓存并弹出预览与分享层。
func (a *AppState) openRenderedShareCard(title, filePrefix, shareText string, img *image.RGBA) {
	if img == nil {
		return
	}
	pngBytes, err := encodeRGBAToPNG(img)
	if err != nil {
		a.updateUI(func() { a.pendingToast = fmt.Sprintf("生成分享图失败: %v", err) })
		return
	}
	shareDir := filepath.Join(a.store.DataDir(), "shares")
	_ = os.MkdirAll(shareDir, 0o755)
	filePath := filepath.Join(shareDir, fmt.Sprintf("%s_%d.png", filePrefix, time.Now().Unix()))
	if err := os.WriteFile(filePath, pngBytes, 0o644); err != nil {
		a.updateUI(func() { a.pendingToast = fmt.Sprintf("写入分享图失败: %v", err) })
		return
	}
	previewSrc, _ := ui.DecodeImage(pngBytes)
	a.updateUI(func() {
		a.sharePreviewTitle = title
		a.sharePreviewText = shareText
		a.sharePreviewPath = filePath
		a.sharePreviewSrc = previewSrc
		a.sharePreviewOpen = true
	})
}

// ShareJournalEntryImageAsync 后台自动排版单条观鸟日志手账图片并展示预览/分享。
func (a *AppState) ShareJournalEntryImageAsync(entry model.JournalEntry) {
	go func() {
		var imgs []ui.ImageSource
		for i, p := range entry.PhotoPaths {
			if i >= 3 {
				break
			}
			if src := loadShareLocalImage(p, 900); src != nil {
				imgs = append(imgs, src)
			}
		}
		card := RenderJournalEntryShareCard(entry, imgs)
		text := fmt.Sprintf("Birdview 观鸟手账：%s（%s）· %s", entry.ComName, entry.SciName, fallbackText(entry.Location, "野外现场"))
		a.openRenderedShareCard("观鸟手账分享图", "birdview_note", text, card)
	}()
}

// ShareJournalSummaryImageAsync 后台自动排版观鸟日志长图海报并展示预览/分享。
func (a *AppState) ShareJournalSummaryImageAsync() {
	entries := a.store.Entries()
	unlockedCount := len(a.store.UnlockedBirds())
	go func() {
		thumbs := make(map[string]ui.ImageSource)
		for i, e := range entries {
			if i >= 6 {
				break
			}
			if len(e.PhotoPaths) > 0 {
				if src := loadShareLocalImage(e.PhotoPaths[0], 260); src != nil {
					thumbs[e.ID] = src
				}
			}
		}
		card := RenderJournalSummaryShareCard(entries, unlockedCount, thumbs)
		text := fmt.Sprintf("我的 Birdview 观鸟图志：已解锁 %d 个鸟种，记录 %d 条目击。", unlockedCount, len(entries))
		a.openRenderedShareCard("观鸟日志分享长图", "birdview_journal", text, card)
	}()
}

// ShareSeasonalBirdImageAsync 后台自动排版单鸟时令指南卡片。
func (a *AppState) ShareSeasonalBirdImageAsync(bird model.SeasonalBird, zone model.EcoZone, month int, regionName string) {
	go func() {
		card := RenderSeasonalBirdShareCard(bird, zone, month, regionName, nil)
		text := fmt.Sprintf("Birdview %d月%s观鸟指南：%s（%s）", month, regionName, bird.ComName, bird.PhaseFor(zone, month).Label())
		a.openRenderedShareCard("时令观鸟指南卡片", "birdview_seasonal", text, card)
	}()
}

// ShareMemosImageAsync 后台自动排版追鸟备忘清单图片。
func (a *AppState) ShareMemosImageAsync() {
	memos := a.store.Memos()
	regionName := fallbackText(a.seasonalRegionLabel, regionShortNameByCode(a.store.Settings().DefaultRegion))
	month := a.seasonalMonth
	if month < 1 || month > 12 {
		month = int(time.Now().Month())
	}
	go func() {
		card := RenderMemoListShareCard(memos, regionName, month)
		text := fmt.Sprintf("我的 %d 月 %s 追鸟备忘清单（共 %d 种目标鸟）", month, regionName, len(memos))
		a.openRenderedShareCard("追鸟备忘分享图", "birdview_memos", text, card)
	}()
}

func (a *AppState) viewShareImagePreviewOverlay(c *ui.Context, t *ui.Theme) {
	if !a.sharePreviewOpen {
		return
	}
	top, _, bottom, _ := c.SafeArea()
	ui.Overlay(c, func() {
		bg := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(ui.RGBA(8, 12, 18, 0.86)).Modal()
		if bg.OverlayShortcut(0, ui.KeyEscape) || bg.OverlayShortcut(0, ui.KeyBack) {
			a.sharePreviewOpen = false
		}
		bg.Children(func() {
			ui.Column(c).Fill().Padding(top+12, 18, max(float32(18), bottom+14), 18).Gap(14).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					ui.Text(c, fallbackText(a.sharePreviewTitle, "图片分享预览")).
						FontSize(17).FontWeight(600).TextColor(ui.RGB(255, 255, 255))
					if circleIconBtn(c, "sp-close", "关闭图片预览", iconClose, ui.RGBA(255, 255, 255, 0.16), ui.RGB(255, 255, 255), 36).Clicked() {
						a.sharePreviewOpen = false
					}
				})

				ui.Box(c).Grow(1).Basis(0).FillWidth().Center().Children(func() {
					if a.sharePreviewSrc != nil {
						ui.Image(c, a.sharePreviewSrc).Fill().Fit(ui.Contain).Radius(16).Label("分享排版预览图")
					} else {
						ui.Spinner(c).Label("生成图片中")
					}
				})

				ui.Row(c).FillWidth().Gap(10).Children(func() {
					saveBtn := ui.Row(c.Key("sp-save")).Grow(1).Basis(0).Height(46).Radius(23).
						Background(ui.RGBA(255, 255, 255, 0.16)).Border(1, ui.RGBA(255, 255, 255, 0.25)).
						AlignItems(ui.Center).Justify(ui.Center).Gap(6).
						Role(ui.RoleButton).Label("保存图片")
					saveBtn.Children(func() {
						ui.Icon(c, iconDownload).FontSize(16).TextColor(ui.RGB(255, 255, 255))
						ui.Text(c, "保存图片").FontSize(14).FontWeight(500).TextColor(ui.RGB(255, 255, 255))
					})
					if saveBtn.Clicked() {
						srcPath := a.sharePreviewPath
						go func() {
							dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
								Parent:      a.win,
								Title:       "保存分享图片",
								DefaultPath: filepath.Base(srcPath),
							})
							if err != nil || dest == "" {
								return
							}
							data, rerr := os.ReadFile(srcPath)
							if rerr == nil {
								rerr = os.WriteFile(dest, data, 0o644)
							}
							a.updateUI(func() {
								if rerr != nil {
									a.pendingToast = fmt.Sprintf("保存失败: %v", rerr)
								} else {
									a.hapticSuccess()
									a.pendingToast = "已保存分享图至 " + filepath.Base(dest)
								}
							})
						}()
					}

					shareBtn := ui.Row(c.Key("sp-share")).Grow(1).Basis(0).Height(46).Radius(23).
						Background(ui.Hex("#4D9488")).
						AlignItems(ui.Center).Justify(ui.Center).Gap(6).
						Role(ui.RoleButton).Label("系统分享")
					shareBtn.Children(func() {
						ui.Icon(c, iconShare).FontSize(16).TextColor(ui.RGB(255, 255, 255))
						ui.Text(c, "立即分享").FontSize(14).FontWeight(600).TextColor(ui.RGB(255, 255, 255))
					})
					if shareBtn.Clicked() {
						srcPath := a.sharePreviewPath
						shareTxt := a.sharePreviewText
						go func() {
							err := mygo.Share(mygo.ShareOptions{
								Text:  shareTxt,
								Files: []string{srcPath},
							})
							if err != nil {
								if errors.Is(err, errors.ErrUnsupported) {
									dest, serr := mygo.Dialog.Save(mygo.SaveDialogOptions{
										Parent:      a.win,
										Title:       "保存分享图片",
										DefaultPath: filepath.Base(srcPath),
									})
									if serr == nil && dest != "" {
										if data, rerr := os.ReadFile(srcPath); rerr == nil {
											_ = os.WriteFile(dest, data, 0o644)
											a.updateUI(func() { a.pendingToast = "已导出图片至 " + filepath.Base(dest) })
										}
									}
									return
								}
								a.updateUI(func() { a.pendingToast = fmt.Sprintf("分享失败: %v", err) })
							}
						}()
					}
				})
			})
		})
	})
}
