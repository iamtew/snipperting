package winapp

import _ "embed"

//go:embed icons/clipboard.ico
var icoClipboard []byte

//go:embed icons/keyboard.ico
var icoKeyboard []byte

//go:embed icons/folder.ico
var icoFolder []byte

//go:embed icons/pencil.ico
var icoPencil []byte

//go:embed icons/settings.ico
var icoSettings []byte

//go:embed icons/help.ico
var icoHelp []byte

func menuIcon(name string) []byte {
	switch name {
	case "clipboard":
		return icoClipboard
	case "keyboard":
		return icoKeyboard
	case "folder":
		return icoFolder
	case "pencil":
		return icoPencil
	case "settings":
		return icoSettings
	case "help":
		return icoHelp
	default:
		return nil
	}
}
