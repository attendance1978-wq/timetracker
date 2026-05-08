package color

import "fmt"

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	black  = "\033[90m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	white  = "\033[97m"
	bGreen = "\033[1;32m"
	bCyan  = "\033[1;36m"
	bWhite = "\033[1;97m"
	bRed   = "\033[1;31m"
)

func Black(s string) string   { return black + s + reset }
func Red(s string) string     { return red + s + reset }
func Green(s string) string   { return green + s + reset }
func Yellow(s string) string  { return yellow + s + reset }
func Cyan(s string) string    { return cyan + s + reset }
func White(s string) string   { return white + s + reset }
func BoldGreen(s string) string { return bGreen + s + reset }
func BoldCyan(s string) string  { return bCyan + s + reset }
func BoldWhite(s string) string { return bWhite + s + reset }
func BoldRed(s string) string   { return bRed + s + reset }
func Bold(s string) string      { return bold + s + reset }

func Blackf(f string, a ...any) string { return Black(fmt.Sprintf(f, a...)) }
func Greenf(f string, a ...any) string { return Green(fmt.Sprintf(f, a...)) }
func Cyanf(f string, a ...any) string  { return Cyan(fmt.Sprintf(f, a...)) }
