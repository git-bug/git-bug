package colors

import "github.com/fatih/color"

var (
	Bold       = color.New(color.Bold).SprintFunc()
	Black      = color.New(color.FgBlack).SprintFunc()
	BlackBold  = color.New(color.FgBlack, color.Bold).SprintfFunc()
	BlackBg    = color.New(color.BgBlack, color.FgWhite).SprintFunc()
	White      = color.New(color.FgWhite).SprintFunc()
	WhiteBold  = color.New(color.FgWhite, color.Bold).SprintFunc()
	WhiteBg    = color.New(color.BgWhite).SprintFunc()
	Yellow     = color.New(color.FgYellow).SprintFunc()
	YellowBold = color.New(color.FgYellow, color.Bold).SprintFunc()
	YellowBg   = color.New(color.BgYellow, color.FgBlack).SprintFunc()
	Green      = color.New(color.FgGreen).SprintFunc()
	GreenBg    = color.New(color.BgGreen, color.FgBlack).SprintFunc()
	Red        = color.New(color.FgRed).SprintFunc()
	Cyan       = color.New(color.FgCyan).SprintFunc()
	CyanBg     = color.New(color.BgCyan, color.FgBlack).SprintFunc()
	Blue       = color.New(color.FgBlue).SprintFunc()
	BlueBg     = color.New(color.BgBlue).SprintFunc()
	Magenta    = color.New(color.FgMagenta).SprintFunc()
)
