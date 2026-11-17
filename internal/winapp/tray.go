package winapp

import (
	"fmt"
	"os"

	"github.com/energye/systray"
)

func RunTray(port int, icon []byte, onExit func()) {
	systray.Run(func() {
		if len(icon) > 0 {
			systray.SetIcon(icon)
		}
		systray.SetTitle("snipperting")
		systray.SetTooltip("snipperting")
		quit := systray.AddMenuItem("Exit", "Quit snipperting")
		quit.Click(func() { systray.Quit() })
		base := fmt.Sprintf("http://127.0.0.1:%d", port)
		_ = base
		systray.SetOnClick(func(systray.IMenu) {
			RememberForeground()
			ShowMenu(port)
		})
		systray.SetOnRClick(func(systray.IMenu) {
			RememberForeground()
			ShowMenu(port)
		})
	}, func() {
		CloseMenu()
		if onExit != nil {
			onExit()
		}
		os.Exit(0)
	})
}
