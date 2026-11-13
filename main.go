package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	app := newAppFromEnv()
	mygo.App.WhenReady(func() {
		window := mygo.NewWindow(mygo.WindowOptions{
			Title:           "Memos",
			Width:           1440,
			Height:          900,
			MinWidth:        320,
			MinHeight:       440,
			StateKey:        "memos-main",
			BackgroundColor: "light-dark(" + appTokens(false)["bg"] + ", " + appTokens(true)["bg"] + ")",
			Content:         ui.View(app.view),
		})
		app.window = window
		if app.client != nil && app.authenticated {
			app.refreshMemos()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
