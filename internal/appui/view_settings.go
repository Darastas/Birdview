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

func maskKeyShort(k string) string {
	trimmed := strings.TrimSpace(k)
	if trimmed == "" {
		return "未配置"
	}
	if len(trimmed) <= 4 {
		return "已配置"
	}
	return fmt.Sprintf("...%s", trimmed[len(trimmed)-4:])
}

func settingsRow(c *ui.Context, key, title, value string, t *ui.Theme, trailing func()) ui.Element {
	row := ui.Row(c.Key(key)).FillWidth().Padding(14, 16).Gap(10).AlignItems(ui.Center).
		Role(ui.RoleButton).Label(title)
	row.Children(func() {
		ui.Text(c, title).FontSize(15).FontWeight(500).Grow(1).SingleLine()
		if value != "" {
			ui.Text(c, value).FontSize(13).TextColor(t.TextMuted).SingleLine()
		}
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
		ui.Box(c).FillWidth().Padding(16, 18, 10, 18).Children(func() {
			ui.Text(c, "设置").FontSize(26).FontWeight(600).LetterSpacing(-0.1).Role(ui.RoleHeading).Level(1)
		})

		ui.Box(c).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Scroll(c.Key("settings-scroll")).Fill().Children(func() {
				ui.Column(c).FillWidth().Padding(10, 18, 106, 18).Gap(14).Children(func() {
					// 1. 数据接口与密钥
					tactileCard(c, t).Clip().Dividers(1, subtleHairlineColor(t)).Children(func() {
						if settingsRow(c, "set-row-ebird", "eBird API Key", maskKeyShort(a.store.LoadEbirdKey()), t, nil).Clicked() {
							a.syncSettingsDraft()
							a.setVerifyMsg = ""
							a.setKeySheetOpen = true
						}

						if settingsRow(c, "set-row-xeno", "Xeno-canto Key", maskKeyShort(a.store.LoadXenoKey()), t, nil).Clicked() {
							a.syncSettingsDraft()
							a.setVerifyMsg = ""
							a.setKeySheetOpen = true
						}
					})

					// 2. 偏好设置
					tactileCard(c, t).Clip().Dividers(1, subtleHairlineColor(t)).Children(func() {
						if settingsRow(c, "set-row-region", "默认观鸟区域", regionShortNameByCode(a.store.Settings().DefaultRegion), t, nil).Clicked() {
							a.regRegionTarget = "settings"
							a.regRegionQuery = ""
							a.regRegionSheetOpen = true
						}

						ui.Column(c).FillWidth().Padding(14, 16).Gap(10).Children(func() {
							ui.Text(c, "外观主题").FontSize(15).FontWeight(500)
							themeIdx := 0
							switch a.setThemeMode {
							case "浅色":
								themeIdx = 1
							case "深色":
								themeIdx = 2
							}
							if tactileSegmentedControl(c, "set-theme-seg", &themeIdx, t, "跟随系统", "浅色", "深色") {
								modes := []string{"跟随系统", "浅色", "深色"}
								a.setThemeMode = modes[themeIdx]
								a.saveThemePreferenceOnly(c, themeModeFromLabel(a.setThemeMode))
							}
						})
					})

					// 3. 存储与备份
					tactileCard(c, t).Clip().Dividers(1, subtleHairlineColor(t)).Children(func() {
						settingsRow(c, "set-row-cache", "在线媒体缓存", formatBytes(a.cacheStatsBytes), t, func() {
							clrBtn := applySculptedSecondaryPillStyle(
								ui.Row(c.Key("clear-cache-btn")).Height(30).PaddingX(13).Radius(15).
									AlignItems(ui.Center).Role(ui.RoleButton).Label("清理"),
								t,
							)
							clrBtn.Children(func() {
								ui.Text(c, "清理").FontSize(13).FontWeight(500).TextColor(t.Text)
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

						if settingsRow(c, "set-row-export", "日志导出与备份", formatBytes(a.localDataBytes), t, nil).Clicked() {
							a.exportSheetOpen = true
						}
					})

					// 4. 关于
					tactileCard(c, t).Clip().Children(func() {
						if settingsRow(c, "set-row-about", "关于 Birdview", "v2.0.0", t, nil).Clicked() {
							a.setAboutSheetOpen = true
						}
					})
				})
			})
		})
	})
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
				ui.Text(c, "API 密钥设置").FontSize(16).FontWeight(600)
				if filterPill(c, "sk-save", "保存设置", true, t).Clicked() {
					a.saveKeysFromUI(c)
				}
			})
			minimalDivider(c, t)

			ui.Scroll(c.Key("sk-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Padding(14, 18, 32, 18).Gap(14).Children(func() {
					tactileCard(c, t).Padding(16).Gap(12).Children(func() {
						ui.Column(c).FillWidth().Gap(8).Children(func() {
							ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "eBird 数据接口密钥").FontSize(13).FontWeight(500).TextColor(t.TextMuted)
								ui.Link(c, "免费申请 Key ↗", "https://ebird.org/api/keygen")
							})
							in1 := minimalTextInput(c, "set-ebird-key", &a.setEbirdKey,
								"输入 X-eBirdApiToken", "eBird API Key 输入框", t)
							if !a.setShowKeys {
								in1.Password()
							}
							ui.Text(c, "用于拉取中国与全球鸟类名录、近期目击及观鸟热点，密钥仅加密保存在本机。").
								FontSize(12).TextColor(t.TextMuted).LineHeight(1.5)
						})

						ui.Column(c).FillWidth().Gap(8).Children(func() {
							ui.Text(c, "Xeno-canto API v3 Key（可选）").FontSize(13).FontWeight(500).TextColor(t.TextMuted)
							in2 := minimalTextInput(c, "set-xc-key", &a.setXenoKey,
								"输入 Xeno-canto API v3 Key", "Xeno-canto API Key", t)
							if !a.setShowKeys {
								in2.Password()
							}
							ui.Text(c, "可选配置；未填写时鸟鸣参考默认播放 iNaturalist 野外录音。").
								FontSize(12).TextColor(t.TextMuted).LineHeight(1.5)
						})

						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "显示密钥明文").FontSize(14)
							mobile.Switch(c.Key("sk-show-sw"), &a.setShowKeys)
						})
					})

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
							ui.Text(c, a.setVerifyMsg).FontSize(12).TextColor(col).Grow(1)
						}
					})
				})
			})
		})
	})
}

func (a *AppState) viewSettingsAboutSheet(c *ui.Context, t *ui.Theme) {
	sheetBg := t.Background
	mobile.Sheet(c, &a.setAboutSheetOpen, mobile.SheetOptions{
		Detents:    []float32{0.52, 0.72},
		Background: &sheetBg,
		Label:      "关于 Birdview",
	}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 20, 14, 20).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "关于 Birdview").FontSize(16).FontWeight(600)
				if filterPill(c, "ab-close", "关闭", false, t).Clicked() {
					a.setAboutSheetOpen = false
				}
			})
			minimalDivider(c, t)

			ui.Scroll(c.Key("ab-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Padding(14, 18, 32, 18).Gap(14).Children(func() {
					tactileCard(c, t).Padding(16).Gap(4).Children(func() {
						ui.Text(c, "Birdview").FontSize(16).FontWeight(600)
						ui.Text(c, "版本 v2.0.0").FontSize(13).TextColor(t.TextMuted)
					})

					tactileCard(c, t).Padding(16).Gap(10).Children(func() {
						ui.Text(c, "数据与媒体来源").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
						sources := []struct {
							name string
							desc string
						}{
							{"eBird API v2", "名录、目击与热点"},
							{"iNaturalist", "物种照片、现场实拍与野外录音"},
							{"Xeno-canto v3", "专业鸟鸣录音扩展"},
							{"中文维基百科", "物种中文摘要"},
						}
						ui.Column(c).FillWidth().Dividers(1, subtleHairlineColor(t)).Children(func() {
							for i, src := range sources {
								ui.Row(c.Key(fmt.Sprintf("ab-src-%d", i))).FillWidth().Padding(9, 0).
									Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(12).Children(func() {
									ui.Text(c, src.name).FontSize(14).FontWeight(500)
									ui.Text(c, src.desc).FontSize(12).TextColor(t.TextMuted)
								})
							}
						})
					})

					tactileCard(c, t).Padding(16).Gap(6).Children(func() {
						ui.Text(c, "本地与在线说明").FontSize(12).FontWeight(500).LetterSpacing(0.2).TextColor(t.TextMuted)
						ui.Text(c, "个人观鸟日志、现场照片与录音仅存储在本机，媒体资源按需在线流式加载。").
							FontSize(13).LineHeight(1.5).TextColor(secondaryTextColor(t))
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
