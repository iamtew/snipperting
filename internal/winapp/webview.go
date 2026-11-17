package winapp

import (
	"fmt"
	"sync"

	"github.com/jchv/go-webview2"
)

var menuMu sync.Mutex
var menu webview2.WebView

func OpenWindow(title, url string, w, h int) {
	go func() {
		wv := webview2.NewWithOptions(webview2.WebViewOptions{
			Debug:     false,
			AutoFocus: true,
			WindowOptions: webview2.WindowOptions{
				Title:  title,
				Width:  uint(w),
				Height: uint(h),
				Center: true,
			},
		})
		if wv == nil {
			return
		}
		wv.Navigate(url)
		wv.Run()
	}()
}

func ShowMenu(port int) {
	menuMu.Lock()
	defer menuMu.Unlock()
	if menu != nil {
		menu.Destroy()
		menu = nil
	}
	x, y := CursorPos()
	url := fmt.Sprintf("http://127.0.0.1:%d/menu", port)
	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "snipperting",
			Width:  320,
			Height: 420,
			Center: false,
		},
	})
	if wv == nil {
		return
	}
	_ = x
	_ = y
	menu = wv
	go func() {
		wv.Navigate(url)
		wv.Run()
		menuMu.Lock()
		if menu == wv {
			menu = nil
		}
		menuMu.Unlock()
	}()
}

func CloseMenu() {
	menuMu.Lock()
	defer menuMu.Unlock()
	if menu != nil {
		menu.Destroy()
		menu = nil
	}
}
