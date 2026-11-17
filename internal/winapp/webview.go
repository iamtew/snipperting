package winapp

import "github.com/jchv/go-webview2"

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
