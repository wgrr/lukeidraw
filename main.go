// +build linux darwin freebsd openbsd netbsd dragonfly solaris

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const usagestr = `lukeidraw: usage: [ file ]
`

var (
	charwScale = 1.0
	charhScale = 1.0
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, usagestr)
		os.Exit(1)
	}
	img, err := os.Open(os.Args[1])
	if err != nil {
		fatal(err.Error())
	}

	var cfg image.Config
	switch filepath.Ext(os.Args[1]) {
	case ".jpg":
		fallthrough
	case ".jpeg":
		cfg, err = jpeg.DecodeConfig(img)
		if err != nil {
			fatal(err.Error())
		}
	case ".gif":
		cfg, err = gif.DecodeConfig(img)
		if err != nil {
			fatal(err.Error())
		}
	case ".png":
		cfg, err = png.DecodeConfig(img)
		if err != nil {
			fatal(err.Error())
		}
	default:
		// try to guess, it might be a image without an filename extension
		cfg, _, err = image.DecodeConfig(img)
		if err != nil {
			fatal(err.Error())
		}
	}

	win, err := unix.IoctlGetWinsize(1, unix.TIOCGWINSZ)
	if err != nil {
		fatal(err.Error())
	}
	// just being paranoid about kernel input, im not being about terminal
	// input because math.Ceil will round up character pixel dimensions, no
	// div by 0 possible there.
	if win.Row == 0 {
		win.Row = 1
	}
	if win.Col == 0 {
		win.Col = 1
	}

	charw := int(math.Ceil((float64(win.Ypixel) * charwScale) / float64(win.Row)))
	charh := int(math.Ceil((float64(win.Xpixel) * charhScale) / float64(win.Col)))

	var term unix.Termios
	tmp, err := unix.IoctlGetTermios(0, unix.TCGETS)
	if err != nil {
		fatal("couldn't setup termio to listen to terminal input: " + err.Error())
	}

	term = *tmp
	term.Lflag &= (^uint32(unix.ICANON) & ^uint32(unix.ECHO))
	err = unix.IoctlSetTermios(0, unix.TCSETS, &term)
	if err != nil {
		fatal("couldn't setup termio to listen to terminal input: " + err.Error())
	}

	os.Stdout.WriteString("\x1b[6n")
	r := bufio.NewReader(os.Stdin)
	termin, err := r.ReadBytes(byte('R'))
	if err != nil {
		// try restore user terminal
		unix.IoctlSetTermios(0, unix.TCSETS, tmp)
		fatal("unexpected input from terminal")
	}

	tmpbuf := bytes.NewBuffer(termin[2:])
	row, err := tmpbuf.ReadBytes(byte(';'))
	if err != nil {
		fatal("unexpected input from terminal")
	}
	col, err := tmpbuf.ReadBytes(byte('R'))
	if err != nil {
		fatal("unexpected input from terminal")
	}

	err = unix.IoctlSetTermios(0, unix.TCSETS, tmp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lukeidraw: %syour terminal is broken, fix it manually by typing reset\n", err)
	}
	fmt.Printf("%d %d %s %s\n", cfg.Width/charw, cfg.Height/charh, string(row[:len(row)-1]), string(col[:len(col)-1]))
}

// noreturn
func fatal(err string) {
	fmt.Fprintf(os.Stderr, "lukeidraw: %v\n", err)
	os.Exit(1)
}
