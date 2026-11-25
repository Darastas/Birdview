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
		return "未配置（点击填写以启用中国名录与区域实时观测）"
	}
	if len(trimmed) <= 4 {
		return "已配置"
	}
	return fmt.Sprintf("已配置（尾号 ...%s）", trimmed[len(trimmed)-4:])
}

func (a *AppState) viewSettingsHome(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Fill().Children(func() {
		ui.Box(c).FillWidth().Padding(12, 16, 10, 16).Children(func() {
			ui.Text(c, "设置").FontSize(26).Bold().Role(ui.RoleHeading).Level(1)
		})

		ui.Scroll(c.Key("settings-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(4, 16, 28, 16).Gap(18).Children(func() {
				// 1. 数据接口与密钥
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					ui.Text(c, "接口与密钥").FontSize(13).FontWeight(600).TextColor(t.TextMuted).PaddingX(4)
					elevatedCard(c, t).Clip().Dividers(0.5, subtleHairlineColor(t)).Children(func() {
						if mobile.ListRow(c.Key("set-row-ebird"), mobile.RowOptions{
							Title:      "eBird API Key",
							Subtitle:   maskKeyPreview(a.store.LoadEbirdKey()),
							Disclosure: true,
						}).Clicked() {
							a.syncSettingsDraft()
							a.setVerifyMsg = ""
							a.setKeySheetOpen = true
						}

						if mobile.ListRow(c.Key("set-row-xeno"), mobile.RowOptions{
							Title:      "Xeno-canto v3 Key（可选）",
							Subtitle:   fallbackText(maskOptionalKey(a.store.LoadXenoKey()), "未配置（图鉴默认播放 iNaturalist 野外录音）"),
							Disclosure: true,
						}).Clicked() {
							a.syncSettingsDraft()
							a.setVerifyMsg = ""
							a.setKeySheetOpen = true
						}
					})
				})

				// 2. 观鸟地区与外观
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					ui.Text(c, "偏好设置").FontSize(13).FontWeight(600).TextColor(t.TextMuted).PaddingX(4)
					elevatedCard(c, t).Clip().Dividers(0.5, subtleHairlineColor(t)).Children(func() {
						if mobile.ListRow(c.Key("set-row-region"), mobile.RowOptions{
							Title:      "默认观鸟区域",
							Subtitle:   a.setRegionLabel,
							Disclosure: true,
						}).Clicked() {
							a.regRegionTarget = "settings"
							a.regRegionQuery = ""
							a.regRegionSheetOpen = true
						}

						ui.Column(c).FillWidth().Padding(12, 16).Gap(8).Children(func() {
							ui.Text(c, "外观主题").FontSize(15)
							themeIdx := 0
							switch a.setThemeMode {
							case "浅色":
								themeIdx = 1
							case "深色":
								themeIdx = 2
							}
							if segmentedPillStrip(c, "set-theme-seg", &themeIdx, t, "跟随系统", "浅色", "深色") {
								modes := []string{"跟随系统", "浅色", "深色"}
								a.setThemeMode = modes[themeIdx]
								a.saveThemePreferenceOnly(c, themeModeFromLabel(a.setThemeMode))
							}
						})
					})
				})

				// 3. 存储与备份
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					ui.Text(c, "存储与备份").FontSize(13).FontWeight(600).TextColor(t.TextMuted).PaddingX(4)
					elevatedCard(c, t).Clip().Dividers(0.5, subtleHairlineColor(t)).Children(func() {
						mobile.ListRow(c.Key("set-row-cache"), mobile.RowOptions{
							Title:    "清理在线媒体缓存",
							Subtitle: fmt.Sprintf("图鉴与音频缓存 %s（%d 个文件）", formatBytes(a.cacheStatsBytes), a.cacheStatsFiles),
							Trailing: func() {
								if filterPill(c, "clear-cache-btn", "清理", false, t).Clicked() {
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
							},
						})

						if mobile.ListRow(c.Key("set-row-export"), mobile.RowOptions{
							Title:      "个人日志与导出备份",
							Subtitle:   fmt.Sprintf("本机已使用 %s · %d 个已解锁鸟种", formatBytes(a.localDataBytes), len(a.store.UnlockedBirds())),
							Disclosure: true,
						}).Clicked() {
							a.exportSheetOpen = true
						}
					})
				})

				// 4. 关于
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					ui.Text(c, "关于").FontSize(13).FontWeight(600).TextColor(t.TextMuted).PaddingX(4)
					elevatedCard(c, t).Clip().Children(func() {
						mobile.ListRow(c.Key("set-row-about"), mobile.RowOptions{
							Title:    "Birdview v2.0.0",
							Subtitle: "数据源自 eBird API v2、iNaturalist、Xeno-canto v3 与中文维基百科",
						})
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
	mobile.Sheet(c, &a.setKeySheetOpen, mobile.SheetOptions{Detents: []float32{0.68, 0.9}, Label: "API 密钥设置"}, func() {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Padding(8, 16, 12, 16).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				if ui.Button(c.Key("sk-cancel"), "取消").Clicked() {
					a.setKeySheetOpen = false
				}
				ui.Text(c, "API 密钥设置").FontSize(17).FontWeight(600)
				if ui.PrimaryButton(c.Key("sk-save"), "保存设置").Radius(16).Clicked() {
					a.saveKeysFromUI(c)
				}
			})

			ui.Scroll(c.Key("sk-scroll")).Grow(1).Basis(0).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Padding(4, 16, 28, 16).Gap(14).Children(func() {
					elevatedCard(c, t).Padding(16).Gap(10).Children(func() {
						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "eBird 数据接口密钥").FontSize(14).FontWeight(600)
							ui.Link(c, "免费申请 Key ↗", "https://ebird.org/api/keygen")
						})
						in1 := ui.TextInput(c.Key("set-ebird-key"), &a.setEbirdKey).
							Placeholder("输入 X-eBirdApiToken（加密保存在系统凭据库）").
							Label("eBird API Key 输入框").FillWidth().Height(42)
						if !a.setShowKeys {
							in1.Password()
						}

						ui.Text(c, "Xeno-canto API v3 Key（可选）").FontSize(14).FontWeight(600).Padding(6, 0, 0, 0)
						in2 := ui.TextInput(c.Key("set-xc-key"), &a.setXenoKey).
							Placeholder("可选：扩展专业鸟鸣库").
							Label("Xeno-canto API Key").FillWidth().Height(42)
						if !a.setShowKeys {
							in2.Password()
						}

						ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Padding(4, 0, 0, 0).Children(func() {
							ui.Text(c, "显示密钥明文").FontSize(14)
							mobile.Switch(c.Key("sk-show-sw"), &a.setShowKeys)
						})
					})

					ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
						if ui.Button(c.Key("verify-key-btn"), "测试 eBird 连接").Disabled(a.setVerifyBusy).Height(42).Radius(21).Clicked() {
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
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
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
