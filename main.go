package main

import (
	"log"
	"os"

	"github.com/Darastas/Birdview/internal/api"
	"github.com/Darastas/Birdview/internal/appui"
	"github.com/Darastas/Birdview/internal/store"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/netimage"
)

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	session := &appui.MobileSession{}
	mygo.KeepState(session)

	mygo.App.WhenReady(func() {
		st, err := store.NewDefault()
		if err != nil {
			log.Fatalf("初始化本地数据存储失败: %v", err)
		}
		cache, err := store.NewDefaultCache()
		if err != nil {
			log.Fatalf("初始化缓存目录失败: %v", err)
		}

		client := api.NewClient()
		netimage.SetClient(client.HTTP)

		appui.ApplyThemeSource(st.Settings().ThemeMode)
		mygo.SetOrientations(mygo.OrientationPortrait)
		mygo.SystemBars.SetStyle(mygo.BarStyleAuto)

		appState := appui.NewAppState(st, cache, client, session)

		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "Birdview",
			Width:     402,
			Height:    844,
			MinWidth:  360,
			MinHeight: 640,
			StateKey:  "mobile",
			Content:   ui.View(appState.View),
		})
		appState.AttachWindow(win)
	})

	if err := mygo.App.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
