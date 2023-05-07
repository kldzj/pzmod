package util

import (
	"github.com/fatih/color"
)

var (
	Info      = color.New(color.FgBlue).Sprint("INFO")
	Warning   = color.New(color.FgYellow).Sprint("WARNING")
	Error     = color.New(color.FgRed).Sprint("ERROR")
	OK        = color.New(color.FgGreen).Sprint("OK")
	Yes       = color.New(color.FgGreen).Sprint("YES")
	No        = color.New(color.FgRed).Sprint("NO")
	Bold      = color.New(color.Bold).SprintFunc()
	Underline = color.New(color.Underline).SprintFunc()
)
