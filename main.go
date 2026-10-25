// 广工抢课助手桌面端：mygo 壳，前端通过 mygo 生成的 TS client 直接调用 internal/service。
package main

import (
	_ "embed"
	"log"

	"course-grabber/internal/service"

	"github.com/egoist/mygo"
)

//go:embed assets/icon.png
var appIcon []byte

//go:embed assets/tray.png
var trayIcon []byte

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	version := mygo.App.Version()
	if version == "" {
		version = "dev"
	}
	svc := service.New(version)
	mygo.BindAs("App", svc)

	var win *mygo.Window
	var quitting bool

	showMain := func() {
		if win != nil {
			win.Show()
			win.Focus()
		}
	}

	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		showMain()
	})

	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		quitting = true
	})

	// 关窗 ≠ 退出：最小化到托盘继续抢课
	mygo.App.OnWindowAllClosed(func() {})

	mygo.App.WhenReady(func() {
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:           "广工抢课助手",
			URL:             "/",
			Width:           1180,
			Height:          780,
			MinWidth:        940,
			MinHeight:       600,
			StateKey:        "main",
			Hidden:          true,
			BackgroundColor: "#eef1f5",
		})
		win.SetIcon(appIcon)
		win.OnReadyToShow(win.Show)

		win.OnClose(func(e *mygo.CloseEvent) {
			if !quitting {
				e.PreventDefault()
				win.Hide()
			}
		})

		t, err := mygo.NewTray(mygo.TrayOptions{
			Icon:    trayIcon,
			ToolTip: "广工抢课助手",
			Menu: mygo.NewMenu([]*mygo.MenuItem{
				{Label: "打开主窗口", Click: func(*mygo.MenuItem, *mygo.Window) { showMain() }},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			}),
		})
		if err != nil {
			log.Println("tray:", err)
		} else {
			t.OnClick(showMain)
		}
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
