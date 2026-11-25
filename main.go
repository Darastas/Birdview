package main

import (
	"log"

	"github.com/Darastas/Birdview/internal/store"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	mygo.App.WhenReady(func() {
		_, err := store.NewDefault()
		if err != nil {
			log.Printf("初始化本地存储失败: %v", err)
		}
		mygo.NewWindow(mygo.WindowOptions{
			Title:     "Birdview",
			Width:     1200,
			Height:    780,
			MinWidth:  900,
			MinHeight: 600,
			StateKey:  "main",
			Content: ui.View(func(c *ui.Context) {
				ui.Column(c).Fill().Center().Gap(8).Children(func() {
					ui.Text(c, "Birdview").FontSize(22).Bold()
					ui.Text(c, "基础架构已就绪").TextColor(c.Theme().TextMuted)
				})
			}),
		})
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
