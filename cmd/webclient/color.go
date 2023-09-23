package main

import (
	"strconv"
	"strings"
)

const escape = "\x1b"

// Base attributes
const (
	Reset byte = iota
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
)

// Foreground text colors
const (
	FgBlack byte = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

// Foreground Hi-Intensity text colors
const (
	FgHiBlack byte = iota + 90
	FgHiRed
	FgHiGreen
	FgHiYellow
	FgHiBlue
	FgHiMagenta
	FgHiCyan
	FgHiWhite
)

// Background text colors
const (
	BgBlack byte = iota + 40
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
)

// Background Hi-Intensity text colors
const (
	BgHiBlack byte = iota + 100
	BgHiRed
	BgHiGreen
	BgHiYellow
	BgHiBlue
	BgHiMagenta
	BgHiCyan
	BgHiWhite
)

func WithColors(s string, col ...byte) string {
	if len(col) == 0 {
		return s
	}
	f := make([]string, len(col))
	for i, v := range col {
		f[i] = strconv.Itoa(int(v))
	}

	b := strings.Builder{}
	b.WriteString(escape)
	b.WriteString("[")
	for i, v := range col {
		if i > 0 {
			b.WriteString(";")
		}
		b.WriteString(strconv.Itoa(int(v)))
	}
	b.WriteString("m")
	b.WriteString(s)
	b.WriteString(escape)
	b.WriteString("[0m")
	return b.String()
}
