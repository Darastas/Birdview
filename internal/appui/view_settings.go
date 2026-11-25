package appui

import (
	"context"
	"fmt"
	"time"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *AppState) viewSettings(c *ui.Context, t *ui.Theme) {
	regionLabels := make([]string, 0, len(api.ChinaRegions))
	for _, r := range api.ChinaRegions {
		regionLabels = append(regionLabels, r.Name)
	}

	ui.Scroll(c.Key("settings-scroll")).Fill().Padding(24).Gap(18).Children(func() {
		ui.Column(c).MaxWidth(760).Gap(18).Children(func() {
			ui.Text(c, "设置").FontSize(20).Bold()

			// 1. API 接口配置
			ui.Column(c).FillWidth().Padding(16).Gap(12).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "数据源与 API 密钥").FontSize(15).Bold()
					ui.Link(c, "申请免费 eBird API Key", "https://ebird.org/api/keygen")
				})

				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, "eBird API Key（用于中国/省级鸟种筛选、近期观鸟记录、清单明细与热点探索）").FontSize(12.5).TextColor(t.TextMuted)
					ui.Row(c).Gap(8).Children(func() {
						in := ui.TextInput(c.Key("set-ebird-key"), &a.setEbirdKey).
							Placeholder("输入 X-eBirdApiToken（保存在系统凭据管理器）").
							Label("eBird API Key").Grow(1)
						if !a.setShowKeys {
							in.Password()
						}
						toggleLabel := "显示"
						if a.setShowKeys {
							toggleLabel = "隐藏"
						}
						if ui.Button(c.Key("toggle-key-vis"), toggleLabel).Clicked() {
							a.setShowKeys = !a.setShowKeys
						}
					})
				})

				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, "Xeno-canto API v3 Key（可选，留空时图鉴默认使用 iNaturalist 野外鸟鸣）").FontSize(12.5).TextColor(t.TextMuted)
					in := ui.TextInput(c.Key("set-xc-key"), &a.setXenoKey).
						Placeholder("可选：填入 Xeno-canto 个人 API Key").
						Label("Xeno-canto API Key").FillWidth()
					if !a.setShowKeys {
						in.Password()
					}
				})

				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					if ui.PrimaryButton(c.Key("save-keys-btn"), "保存设置").Clicked() {
						a.saveSettingsFromUI(c)
					}
					verifyBtn := ui.Button(c.Key("verify-key-btn"), "验证 eBird 连接").Disabled(a.setVerifyBusy)
					if verifyBtn.Clicked() {
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
						ui.Text(c, a.setVerifyMsg).FontSize(12.5).TextColor(col)
					}
				})
			})

			// 2. 地区与外观偏好
			ui.Column(c).FillWidth().Padding(16).Gap(12).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Text(c, "默认地区与界面外观").FontSize(15).Bold()

				ui.Row(c).Gap(16).FillWidth().Children(func() {
					ui.Column(c).Grow(1).Gap(6).Children(func() {
						ui.Text(c, "默认观鸟省份 / 地区").FontSize(12.5).TextColor(t.TextMuted)
						ui.Select(c.Key("set-region"), &a.setRegionLabel, regionLabels).Label("默认地区").FillWidth()
					})

					ui.Column(c).Width(220).Gap(6).Children(func() {
						ui.Text(c, "界面外观").FontSize(12.5).TextColor(t.TextMuted)
						if ui.Select(c.Key("set-theme"), &a.setThemeMode, []string{"跟随系统", "浅色", "深色"}).Label("界面外观").FillWidth().Changed() {
							if a.win != nil {
								ApplyThemeSource(themeModeFromLabel(a.setThemeMode))
							}
						}
					})
				})
			})

			// 3. 存储与按需缓存管理
			ui.Column(c).FillWidth().Padding(16).Gap(12).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Text(c, "按需缓存与本地存储").FontSize(15).Bold()
				ui.Text(c, "图鉴图片与鸟鸣音频均为在线按需加载，磁盘缓存上限 64 MB（自动按最久未用清理）。").
					FontSize(12.5).TextColor(t.TextMuted)

				ui.Row(c).Gap(16).Wrap().Children(func() {
					infoChip(c, t, "在线媒体缓存", fmt.Sprintf("%s（%d 个文件）", formatBytes(a.cacheStatsBytes), a.cacheStatsFiles))
					infoChip(c, t, "个人日志与附件", formatBytes(a.localDataBytes))
					infoChip(c, t, "已解锁鸟种", fmt.Sprintf("%d 种", len(a.store.UnlockedBirds())))
				})

				ui.Row(c).Gap(10).Children(func() {
					if ui.Button(c.Key("clear-cache-btn"), "清空在线媒体缓存").Clicked() {
						if a.cache != nil {
							_ = a.cache.Clear()
						}
						a.imgCache = make(map[string]ui.ImageSource)
						a.imgOrder = nil
						a.imgErr = make(map[string]string)
						a.refreshStorageStats()
						c.Toast("已清空在线图片与音频缓存")
					}
					if ui.Button(c.Key("open-data-dir"), "打开本地日志目录").Clicked() {
						_ = mygo.Shell.OpenPath(a.store.DataDir())
					}
				})
			})
		})
	})
}

func (a *AppState) saveSettingsFromUI(c *ui.Context) {
	prevRegion := a.store.Settings().DefaultRegion
	nextRegion := regionCodeByLabel(a.setRegionLabel)
	themeMode := themeModeFromLabel(a.setThemeMode)

	if err := a.store.SaveEbirdKey(a.setEbirdKey); err != nil {
		c.Toast(fmt.Sprintf("保存 eBird Key 失败: %v", err))
		return
	}
	if err := a.store.SaveXenoKey(a.setXenoKey); err != nil {
		c.Toast(fmt.Sprintf("保存 Xeno-canto Key 失败: %v", err))
		return
	}
	cfg := a.store.Settings()
	cfg.DefaultRegion = nextRegion
	cfg.ThemeMode = themeMode
	if err := a.store.SaveSettings(cfg); err != nil {
		c.Toast(fmt.Sprintf("保存设置失败: %v", err))
		return
	}
	if a.win != nil {
		ApplyThemeSource(themeMode)
		a.regionalRegion = nextRegion
		if prevRegion != nextRegion || len(a.chinaCodes) == 0 {
			a.loadTaxonomyAsync(false)
		}
	} else {
		a.regionalRegion = nextRegion
	}
	c.Toast("设置已保存")
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
				a.setVerifyMsg = fmt.Sprintf("验证通过（响应耗时 %d ms）", elapsed.Milliseconds())
			}
		})
	}()
}
