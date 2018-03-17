package window

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	. "github.com/itchyny/bed/common"
)

// Manager manages the windows and files.
type Manager struct {
	height   int64
	windows  []*window
	index    int
	files    []file
	eventCh  chan<- Event
	redrawCh chan<- struct{}
}

type file struct {
	name string
	file *os.File
	perm os.FileMode
}

// NewManager creates a new Manager.
func NewManager() *Manager {
	return &Manager{}
}

// Init initializes the Manager.
func (m *Manager) Init(eventCh chan<- Event, redrawCh chan<- struct{}) error {
	m.eventCh, m.redrawCh = eventCh, redrawCh
	return nil
}

// Open a new window.
func (m *Manager) Open(filename string) error {
	if filename == "" {
		window, err := newWindow(bytes.NewReader(nil), "", "", m.height, 16, m.redrawCh)
		if err != nil {
			return err
		}
		m.windows = append(m.windows, window)
		m.index = len(m.windows) - 1
		return nil
	}
	f, err := os.Open(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		window, err := newWindow(bytes.NewReader(nil), filename, filepath.Base(filename), m.height, 16, m.redrawCh)
		if err != nil {
			return err
		}
		m.windows = append(m.windows, window)
		m.index = len(m.windows) - 1
		return nil
	}
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	m.files = append(m.files, file{name: filename, file: f, perm: info.Mode().Perm()})
	window, err := newWindow(f, filename, filepath.Base(filename), m.height, 16, m.redrawCh)
	if err != nil {
		return err
	}
	m.windows = append(m.windows, window)
	m.index = len(m.windows) - 1
	return nil
}

// SetHeight sets the height.
func (m *Manager) SetHeight(height int) {
	m.height = int64(height)
}

// Run the Manager.
func (m *Manager) Run() {
	m.windows[m.index].Run()
}

// Emit an event to the current window.
func (m *Manager) Emit(event Event) {
	switch event.Type {
	case EventCursorGotoAbs:
		fallthrough
	case EventCursorGotoRel:
		if len(event.Args) > 1 {
			m.eventCh <- Event{Type: EventError, Error: fmt.Errorf("too many arguments for %s", event.CmdName)}
		} else if len(event.Args) == 1 {
			event.Count = parseGotoPos(event.Args[0])
			m.windows[m.index].eventCh <- event
		}
	case EventEdit:
		if len(event.Args) > 1 {
			m.eventCh <- Event{Type: EventError, Error: fmt.Errorf("too many arguments for %s", event.CmdName)}
		} else if len(event.Args) == 0 {
			m.eventCh <- Event{Type: EventError, Error: errors.New("no file name")}
		} else {
			if err := m.Open(event.Args[0]); err != nil {
				m.eventCh <- Event{Type: EventError, Error: err}
			}
			go m.Run()
			m.eventCh <- Event{Type: EventError, Error: nil}
		}
	case EventWrite:
		if len(event.Args) > 1 {
			m.eventCh <- Event{Type: EventError, Error: fmt.Errorf("too many arguments for %s", event.CmdName)}
		} else {
			var name string
			if len(event.Args) > 0 {
				name = event.Args[0]
			}
			if filename, n, err := m.writeFile(name); err != nil {
				m.eventCh <- Event{Type: EventError, Error: err}
			} else {
				m.eventCh <- Event{Type: EventInfo, Error: fmt.Errorf("%s: %d (0x%x) bytes written", filename, n, n)}
			}
		}
	case EventWriteQuit:
		if len(event.Args) > 0 {
			m.eventCh <- Event{Type: EventError, Error: fmt.Errorf("too many arguments for %s", event.CmdName)}
		} else {
			if _, _, err := m.writeFile(""); err != nil {
				m.eventCh <- Event{Type: EventError, Error: err}
			} else {
				m.eventCh <- Event{Type: EventQuit}
			}
		}
	default:
		m.windows[m.index].eventCh <- event
	}
}

func parseGotoPos(pos string) int64 {
	switch pos {
	case "$":
		return math.MaxInt64
	case "+":
		return 1
	case "-":
		return -1
	}
	count, sign := int64(0), int64(1)
	for _, c := range pos {
		count *= 0x10
		if '0' <= c && c <= '9' {
			count += int64(c - '0')
		} else if 'a' <= c && c <= 'f' {
			count += int64(c - 'a' + 0x0a)
		} else if c == '-' {
			sign = -1
		}
	}
	return sign * count
}

// State returns the state of the windows.
func (m *Manager) State() ([]WindowState, error) {
	state, err := m.windows[m.index].State()
	if err != nil {
		return nil, err
	}
	return []WindowState{state}, nil
}

func (m *Manager) writeFile(name string) (string, int64, error) {
	window := m.windows[m.index]
	perm := os.FileMode(0644)
	if name == "" {
		name = window.filename
	}
	if name == "" {
		return name, 0, errors.New("no file name")
	}
	if window.filename == "" {
		window.filename = name
	}
	for _, f := range m.files {
		if f.name == name {
			perm = f.perm
		}
	}
	tmpf, err := os.OpenFile(
		name+"-"+strconv.FormatUint(rand.Uint64(), 16), os.O_RDWR|os.O_CREATE|os.O_EXCL, perm,
	)
	if err != nil {
		return name, 0, err
	}
	defer os.Remove(tmpf.Name())
	window.buffer.Seek(0, io.SeekStart)
	n, err := io.Copy(tmpf, window.buffer)
	tmpf.Close()
	if err != nil {
		return name, 0, err
	}
	return name, n, os.Rename(tmpf.Name(), name)
}

// Close the Manager.
func (m *Manager) Close() {
	for _, f := range m.files {
		f.file.Close()
	}
	for _, w := range m.windows {
		w.Close()
	}
}
