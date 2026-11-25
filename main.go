package main

import (
	"log"
	"os"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/appui"
	"github.com/Darastas/Birdview/internal/store"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	mygo.App.WhenReady(func() {
		st, err := store.NewDefault()
		if err != nil {
			log.Fatalf("初始化本地数据存储失败: %v", err)
		}
		cache, err := store.NewDefaultCache()
		if err != nil {
			log.Fatalf("初始化缓存目录失败: %v", err)
		}

		appui.ApplyThemeSource(st.Settings().ThemeMode)
		appState := appui.NewAppState(st, cache, api.NewClient())

		menu := mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Label: "文件", Submenu: []*mygo.MenuItem{
				{Label: "新建观鸟日志…", Accelerator: "CmdOrCtrl+N", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					appState.OpenNewJournalModal("", "")
				}},
				{Label: "刷新当前数据", Accelerator: "CmdOrCtrl+R", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					appState.RefreshCurrent()
				}},
				mygo.Separator(),
				{Role: mygo.RoleClose},
				{Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu},
			{Label: "视图", Submenu: []*mygo.MenuItem{
				{Label: "鸟类图鉴", Accelerator: "CmdOrCtrl+1", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { appState.SetPage("guide") }},
				{Label: "观鸟日志", Accelerator: "CmdOrCtrl+2", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { appState.SetPage("journal") }},
				{Label: "已解锁鸟种", Accelerator: "CmdOrCtrl+3", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { appState.SetPage("lifelist") }},
				{Label: "区域观鸟动态", Accelerator: "CmdOrCtrl+4", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { appState.SetPage("regional") }},
				{Label: "观鸟热点探索", Accelerator: "CmdOrCtrl+5", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { appState.SetPage("hotspots") }},
				mygo.Separator(),
				{Label: "设置", Accelerator: "CmdOrCtrl+,", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { appState.SetPage("settings") }},
			}},
		})
		mygo.App.SetMenu(menu)

		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "Birdview",
			Width:     1200,
			Height:    780,
			MinWidth:  900,
			MinHeight: 600,
			StateKey:  "main",
			Content:   ui.View(appState.View),
		})
		appState.AttachWindow(win)
	})

	if err := mygo.App.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
