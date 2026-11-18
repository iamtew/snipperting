package winapp

import (
	"strings"

	"golang.org/x/sys/windows"
)

const (
	aboutSite = "https://iamtew.github.io/snipperting/"
	aboutRepo = "https://github.com/iamtew/snipperting"
)

func aboutLines(ver, rev string) (verLine, revLine string) {
	verLine = "Development build"
	if v := strings.TrimPrefix(ver, "v"); v != "" && v != "dev" {
		verLine = "Version v" + v
	}
	if rev != "" {
		revLine = "Commit " + rev
	}
	return verLine, revLine
}

func openURL(u string) {
	_ = windows.ShellExecute(0, nil, windows.StringToUTF16Ptr(u), nil, nil, windows.SW_SHOWNORMAL)
}
