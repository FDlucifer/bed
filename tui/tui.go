package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"

	. "github.com/itchyny/bed/common"
)

// Tui implements UI
type Tui struct {
	width   int
	height  int
	eventCh chan<- Event
	mode    Mode
	screen  tcell.Screen
}

// NewTui creates a new Tui.
func NewTui() *Tui {
	return &Tui{}
}

// Init initializes the Tui.
func (ui *Tui) Init(eventCh chan<- Event) (err error) {
	ui.eventCh = eventCh
	ui.mode = ModeNormal
	if ui.screen, err = tcell.NewScreen(); err != nil {
		return
	}
	return ui.screen.Init()
}

// Run the Tui.
func (ui *Tui) Run(kms map[Mode]*KeyManager) {
	kms[ModeNormal].Register(EventQuit, "q")
	kms[ModeNormal].Register(EventQuit, "c-c")
	for {
		e := ui.screen.PollEvent()
		switch ev := e.(type) {
		case *tcell.EventKey:
			if event := kms[ui.mode].Press(eventToKey(ev)); event.Type != EventNop {
				ui.eventCh <- event
			} else {
				ui.eventCh <- Event{Type: EventRune, Rune: ev.Rune()}
			}
		case nil:
			return
		}
	}
}

// Height returns the height for the hex view.
func (ui *Tui) Height() int {
	_, height := ui.screen.Size()
	return height - 3
}

// Width returns the width for the hex view.
func (ui *Tui) Width() int {
	width, _ := ui.screen.Size()
	return width
}

func (ui *Tui) setLine(line int, offset int, str string, style tcell.Style) {
	for _, c := range str {
		ui.screen.SetContent(offset, line, c, nil, style)
		offset += runewidth.RuneWidth(c)
	}
}

// Redraw redraws the state.
func (ui *Tui) Redraw(state State) error {
	ui.mode = state.Mode
	ui.newTuiWindow().drawWindow(state.Windows[0])
	ui.drawCmdline(state)
	ui.screen.Show()
	return nil
}

func (ui *Tui) newTuiWindow() *tuiWindow {
	return &tuiWindow{
		width:  ui.Width(),
		height: ui.Height(),
		screen: ui.screen,
	}
}

func (ui *Tui) drawCmdline(state State) {
	if state.Error != nil {
		style := tcell.StyleDefault.Foreground(tcell.ColorRed)
		if state.ErrorType == MessageInfo {
			style = style.Foreground(tcell.ColorYellow)
		}
		ui.setLine(ui.Height()+2, 0, state.Error.Error()+strings.Repeat(" ", ui.Width()), style)
	} else if state.Mode == ModeCmdline {
		ui.setLine(ui.Height()+2, 0, ":"+string(state.Cmdline)+strings.Repeat(" ", ui.Width()), 0)
		ui.screen.ShowCursor(1+runewidth.StringWidth(string(state.Cmdline[:state.CmdlineCursor])), ui.Height()+2)
	}
}

func prettyByte(b byte) byte {
	switch {
	case 0x20 <= b && b < 0x7f:
		return b
	default:
		return 0x2e
	}
}

func prettyRune(b byte) string {
	switch {
	case b == 0x07:
		return "\\a"
	case b == 0x08:
		return "\\b"
	case b == 0x09:
		return "\\t"
	case b == 0x0a:
		return "\\n"
	case b == 0x0b:
		return "\\v"
	case b == 0x0c:
		return "\\f"
	case b == 0x0d:
		return "\\r"
	case b < 0x20:
		return fmt.Sprintf("\\x%02x", b)
	case b == 0x27:
		return "\\'"
	case b < 0x7f:
		return string(rune(b))
	default:
		return fmt.Sprintf("\\u%04x", b)
	}
}

func prettyMode(mode Mode) string {
	switch mode {
	case ModeInsert:
		return "[INSERT] "
	case ModeReplace:
		return "[REPLACE] "
	default:
		return ""
	}
}

// Close terminates the Tui.
func (ui *Tui) Close() error {
	ui.screen.Fini()
	return nil
}
