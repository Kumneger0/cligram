package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type SessionState int

const (
	MainView SessionState = iota
	ModalView
)

type OverlayMode string

const (
	Search OverlayMode = "SEARCH"
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
	var cmds []tea.Cmd
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
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.State == ModalView {
				m.State = MainView
				return m, nil
			}
		case "q":
			if m.State == MainView {
				if bg, ok := m.Background.(Model); ok && bg.FocusedOn == Input {
					break
				}
				return m, tea.Quit
			}
		case "ctrl+k":
			if m.State == MainView {
				m.State = ModalView
				openModalMsg := func() tea.Msg {
					return OpenModalMsg{
						ModalMode: ModalModeSearch,
					}
				}
				cmds = append(cmds, openModalMsg)
				return m, tea.Batch(cmds...)
			}
			m.State = MainView
		case "ctrl+r":
			m.State = ModalView
			bgModel, cmd := m.Background.Update(message)
			m.Background = bgModel
			openModalMsg := func() tea.Msg {
				return OpenModalMsg{
					ModalMode: ModalModeSendReaction,
				}
			}
			cmds = append(cmds, cmd, openModalMsg)
			return m, tea.Batch(cmds...)
		case "alt+s":
			m.State = ModalView
			bgModel, cmd := m.Background.Update(message)
			m.Background = bgModel
			var stories []types.Stories
			if bg, ok := m.Background.(Model); ok {
				stories = bg.Stories
			}
			openModalMsg := func() tea.Msg {
				return OpenModalMsg{
					ModalMode: ModalModeShowStories,
					Stories:   stories,
				}
			}
			cmds = append(cmds, cmd, openModalMsg)
			if len(stories) == 0 && telegram.Cligram != nil && telegram.Cligram.Context() != nil {
				cmds = append(cmds, telegram.Cligram.GetAllStories(telegram.Cligram.Context()))
			}
			return m, tea.Batch(cmds...)
		case "alt+m", "alt+h":
			bg, bgCmd := m.Background.Update(message)
			m.Background = bg
			return m, bgCmd
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
	return m, tea.Batch(fgCmd, bgCmd)
}
func (m Manager) View() string {
	if m.State == ModalView {
		fg := m.Foreground.View()
		bg := m.Background.View()
		return overlay.Composite(fg, bg, overlay.Center, overlay.Center, 0, 0)
	}
	return m.Background.View()
}
