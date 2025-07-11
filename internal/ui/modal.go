package ui

import (
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/rpc"
)

type SessionState int

const (
	MainView SessionState = iota
	ModalView
)

type OverlayMode string

const (
	Search OverlayMode = "SEARCH"
	//todo:add more modes here
)

type Manager struct {
	State        SessionState
	WindowWidth  int
	WindowHeight int
	Foreground   tea.Model
	Background   tea.Model
	Overlay      tea.Model
	OverlayMode  OverlayMode
}

func (m Manager) Init() tea.Cmd {
	return tea.Batch(
		m.Foreground.Init(),
		m.Background.Init(),
	)
}

func (m Manager) Update(message tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.WindowWidth = msg.Width
		m.WindowHeight = msg.Height

	case CloseOverlay:
		m.State = MainView
		return m, nil

	case OpenModalMsg:
		m.State = ModalView
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if m.State == MainView {
				err := rpc.JsProcess.Signal(os.Interrupt)
				if err != nil {
					slog.Error("Failed to send interrupt signal to JS process", "error", err.Error())
				}
				return m, tea.Quit
			}
			m.State = MainView
			return m, nil
		case "ctrl+k":
			if m.State == MainView {
				m.State = ModalView
			} else {
				m.State = MainView
			}
		}
		if m.State == ModalView {
			fg, fgCmd := m.Foreground.Update(message)
			m.Foreground = fg
			return m, fgCmd
		}
		if m.State == MainView {
			bg, bgCmd := m.Background.Update(message)
			m.Background = bg
			return m, bgCmd
		}
	}

	fg, fgCmd := m.Foreground.Update(message)
	m.Foreground = fg

	bg, bgCmd := m.Background.Update(message)
	m.Background = bg

	cmds := []tea.Cmd{}
	cmds = append(cmds, fgCmd, bgCmd)

	return m, tea.Batch(cmds...)
}

func (m Manager) View() string {
	if m.State == ModalView {
		return m.Overlay.View()
	}
	return m.Background.View()
}
