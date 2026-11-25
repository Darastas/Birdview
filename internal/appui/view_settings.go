package appui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"
	"github.com/egoist/mygo/ui/netimage"
)

func maskKeyPreview(k string) string {
	trimmed := strings.TrimSpace(k)
	if trimmed == "" {
		return "未配置（点击填写以启用中国名录与区域观测）"
	}
	if len(trimmed) <= 4 {
		return "已配置"
	}
	return fmt.Sprintf("已配置（尾号 ...%s）", trimmed[len(trimmed)-4:])
}

func settingsRow(c *ui.Context, key, title, subtitle string, t *ui.Theme, trailing func()) ui.Element {
	row := ui.Row(c.Key(key)).FillWidth().Padding(13, 16).Gap(10).AlignItems(ui.Center).
		Role(ui.RoleButton).Label(title)
	row.Children(func() {
		ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(15).FontWeight(500)
			if subtitle != "" {
				ui.Text(c, subtitle).FontSize(12.5).TextColor(t.TextMuted).SingleLine()
			}
		})
		if trailing != nil {
			trailing()
		} else {
			ui.Icon(c, iconChevronRight).FontSize(15).TextColor(t.TextMuted.Alpha(0.55))
		}
	})
	return row
}

func (a *AppState) viewSettingsHome(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Fill().Children(func() {
		ui.Box(c).FillWidth().Padding(16, 20, 8, 20).Children(func() {
			ui.Text(c, "设置").FontSize(24).FontWeight(700).LetterSpacing(-0.4).Role(ui.RoleHeading).Level(1)
		})

		ui.Scroll(c.Key("settings-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(6, 20, 32, 20).Gap(20).Children(func() {
				// 1. 数据接口与密钥
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, "接口与密钥").FontSize(12.5).FontWeight(600).TextColor(t.TextMuted).PaddingX(4)
					flatSurface(c, t).Clip().Dividers(1, t.Border).Children(func() {
						if settingsRow(c, "set-row-ebird", "eBird API Key", maskKeyPreview(a.store.LoadEbirdKey()), t, nil).Clicked() {
							a.syncSettingsDraft()
							a.setVerifyMsg = ""
							a.setKeySheetOpen = true
						}

						if settingsRow(c, "set-row-xeno", "Xeno-canto v3 Key（可选）",
							fallbackText(maskOptionalKey(a.store.LoadXenoKey()), "未配置（默认播放 iNaturalist 野外录音）"), t, nil).Clicked() {
							a.syncSettingsDraft()
							a.setVerifyMsg = ""
							a.setKeySheetOpen = true
						}
					})
				})

				// 2. 偏好设置
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, "偏好设置").FontSize(12.5).FontWeight(600).TextColor(t.TextMuted).PaddingX(4)
					flatSurface(c, t).Clip().Dividers(1, t.Border).Children(func() {
						if settingsRow(c, "set-row-region", "默认观鸟区域", a.setRegionLabel, t, nil).Clicked() {
							a.regRegionTarget = "settings"
							a.regRegionQuery = ""
							a.regRegionSheetOpen = true
						}

						ui.Column(c).FillWidth().Padding(13, 16).Gap(10).Children(func() {
							ui.Text(c, "外观主题").FontSize(15).FontWeight(500)
							themeIdx := 0
							switch a.setThemeMode {
							case "浅色":
								themeIdx = 1
							case "深色":
								themeIdx = 2
							}
							if minimalSegmentedControl(c, "set-theme-seg", &themeIdx, t, "跟随系统", "浅色", "深色") {
								modes := []string{"跟随系统", "浅色", "深色"}
								a.setThemeMode = modes[themeIdx]
								a.saveThemePreferenceOnly(c, themeModeFromLabel(a.setThemeMode))
							}
						})
					})
				})

				// 3. 存储与备份
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, "存储与备份").FontSize(12.5).FontWeight(600).TextColor(t.TextMuted).PaddingX(4)
					flatSurface(c, t).Clip().Dividers(1, t.Border).Children(func() {
						settingsRow(c, "set-row-cache", "清理在线媒体缓存",
							fmt.Sprintf("已缓存 %s（%d 个文件）", formatBytes(a.cacheStatsBytes), a.cacheStatsFiles), t, func() {
								clrBtn := ui.Row(c.Key("clear-cache-btn")).Height(28).PaddingX(12).Radius(14).
									Background(t.Background).AlignItems(ui.Center).Role(ui.RoleButton).Label("清理")
								if t.Dark {
									clrBtn.Border(1, t.Border)
								}
								clrBtn.Children(func() {
									ui.Text(c, "清理").FontSize(12).FontWeight(500).TextColor(t.Text)
								})
								if clrBtn.Clicked() {
									if a.cache != nil {
										_ = a.cache.Clear()
									}
									netimage.Clear()
									a.thumbMap = make(map[string]string)
									a.thumbTried = make(map[string]bool)
									a.localImgCache = make(map[string]ui.ImageSource)
									a.localImgOrder = nil
									a.refreshStorageStats()
									c.Toast("已清空在线媒体缓存")
								}
							})

						if settingsRow(c, "set-row-export", "个人日志与导出备份",
							fmt.Sprintf("本机占用 %s · %d 个已解锁鸟种", formatBytes(a.localDataBytes), len(a.store.UnlockedBirds())), t, nil).Clicked() {
							a.exportSheetOpen = true
						}
					})
				})

				// 4. 关于
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, "关于").FontSize(12.5).FontWeight(600).TextColor(t.TextMuted).PaddingX(4)
					flatSurface(c, t).Padding(14, 16).Gap(4).Children(func() {
						ui.Text(c, "Birdview v2.0.0").FontSize(15).FontWeight(600)
						ui.Text(c, "数据源自 eBird API v2、iNaturalist、Xeno-canto v3 与中文维基百科").
							FontSize(12.5).TextColor(t.TextMuted).LineHeight(1.5)
					})
				})
			})
		})
	})
}

func maskOptionalKey(k string) string {
	trimmed := strings.TrimSpace(k)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return "已配置"
	}
	return fmt.Sprintf("已配置（尾号 ...%s）", trimmed[len(trimmed)-4:])
}

func (a *AppState) viewSettingsKeySheet(c *ui.Context, t *ui.Theme) {
	sheetBg := t.Background
	mobile.Sheet(c, &a.setKeySheetOpen, mobile.SheetOptions{
		Detents:    []float32{0.68, 0.9},
		Background: &sheetBg,
		Label:      "API 密钥设置",
	}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 20, 14, 20).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				cancelBtn := ui.Box(c.Key("sk-cancel")).Padding(6, 4).
					Role(ui.RoleButton).Label("取消").Children(func() {
					ui.Text(c, "取消").FontSize(15).TextColor(t.TextMuted)
				})
				if cancelBtn.Clicked() {
					a.setKeySheetOpen = false
				}
				ui.Text(c, "API 密钥设置").FontSize(16.5).FontWeight(600)
				if filterPill(c, "sk-save", "保存设置", true, t).Clicked() {
					a.saveKeysFromUI(c)
				}
			})
			minimalDivider(c, t)

			ui.Scroll(c.Key("sk-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Padding(16, 20, 32, 20).Gap(18).Children(func() {
					ui.Column(c).FillWidth().Gap(8).Children(func() {
						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "eBird 数据接口密钥").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
							ui.Link(c, "免费申请 Key ↗", "https://ebird.org/api/keygen")
						})
						in1 := minimalTextInput(c, "set-ebird-key", &a.setEbirdKey,
							"输入 X-eBirdApiToken（加密保存在本机凭据库）", "eBird API Key 输入框", t)
						if !a.setShowKeys {
							in1.Password()
						}
					})

					ui.Column(c).FillWidth().Gap(8).Children(func() {
						ui.Text(c, "Xeno-canto API v3 Key（可选）").FontSize(13).FontWeight(600).TextColor(t.TextMuted)
						in2 := minimalTextInput(c, "set-xc-key", &a.setXenoKey,
							"可选：扩展专业鸟鸣库", "Xeno-canto API Key", t)
						if !a.setShowKeys {
							in2.Password()
						}
					})

					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "显示密钥明文").FontSize(14.5)
						mobile.Switch(c.Key("sk-show-sw"), &a.setShowKeys)
					})

					minimalDivider(c, t)

					ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
						if monochromeButton(c, "verify-key-btn", "测试 eBird 连接", false, t).
							Height(40).Radius(20).Disabled(a.setVerifyBusy).Clicked() {
							a.verifyEbirdKeyAsync()
						}
						if a.setVerifyBusy {
							ui.Spinner(c).Label("验证中")
						}
						if a.setVerifyMsg != "" {
							col := t.Danger
							if a.setVerifyOK {
								col = t.Success
							}
							ui.Text(c, a.setVerifyMsg).FontSize(12.5).TextColor(col).Grow(1)
						}
					})
				})
			})
		})
	})
}

func (a *AppState) saveKeysFromUI(c *ui.Context) {
	if err := a.store.SaveEbirdKey(a.setEbirdKey); err != nil {
		c.Toast(fmt.Sprintf("保存 eBird Key 失败: %v", err))
		return
	}
	if err := a.store.SaveXenoKey(a.setXenoKey); err != nil {
		c.Toast(fmt.Sprintf("保存 Xeno-canto Key 失败: %v", err))
		return
	}
	a.setKeySheetOpen = false
	if a.win != nil && len(a.chinaCodes) == 0 {
		a.loadTaxonomyAsync(false)
	}
	a.hapticSuccess()
	c.Toast("API 密钥已保存")
}

func (a *AppState) saveRegionPreferenceOnly(c *ui.Context, nextRegion string) {
	prevRegion := a.store.Settings().DefaultRegion
	cfg := a.store.Settings()
	cfg.DefaultRegion = nextRegion
	if err := a.store.SaveSettings(cfg); err != nil {
		c.Toast(fmt.Sprintf("保存设置失败: %v", err))
		return
	}
	a.regionalRegion = nextRegion
	a.regRegionLabel = regionLabelByCode(nextRegion)
	if a.win != nil && prevRegion != nextRegion {
		a.loadTaxonomyAsync(false)
	}
	c.Toast("已切换默认地区：" + a.setRegionLabel)
}

func (a *AppState) saveThemePreferenceOnly(c *ui.Context, mode string) {
	cfg := a.store.Settings()
	cfg.ThemeMode = mode
	_ = a.store.SaveSettings(cfg)
	if a.win != nil {
		ApplyThemeSource(mode)
	}
}

func (a *AppState) verifyEbirdKeyAsync() {
	key := a.setEbirdKey
	a.setVerifyBusy = true
	a.setVerifyMsg = ""
	a.invalidateUI()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		elapsed, err := a.client.VerifyEbirdKey(ctx, key)
		cancel()

		a.updateUI(func() {
			a.setVerifyBusy = false
			if err != nil {
				a.setVerifyOK = false
				a.setVerifyMsg = fmt.Sprintf("连接失败: %v", err)
			} else {
				a.setVerifyOK = true
				a.setVerifyMsg = fmt.Sprintf("验证通过（%d ms）", elapsed.Milliseconds())
			}
		})
	}()
}
