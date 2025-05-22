package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ()

type CustomDelegate struct {
	list.DefaultDelegate
}

func (d CustomDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string
	
	if entry, ok := item.(UserInfo); ok {
		title = entry.Title()
	} else if entry, ok := item.(ChannelInfo); ok {
		title = entry.Title()
	} else {
		return
	}

	str := lipgloss.NewStyle().Width(50).Render(title)
	if index == m.Index() {
		fmt.Fprint(w, selectedStyle.Render(" "+str+" "))
	} else {
		fmt.Fprint(w, normalStyle.Render(" "+str+" "))
	}
}

func GetFakeData() []list.Item {
	cwd, _ := os.Getwd()
	fakeUsersPath := filepath.Join(cwd, "internal", "ui", "testUsers.txt")
	testJson, err := os.ReadFile(fakeUsersPath)
	if err != nil {
		fmt.Println(err.Error())
	}
	var fakeUsers []UserInfo
	err = json.Unmarshal(testJson, &fakeUsers)

	if err != nil {
		fmt.Println(err.Error())
	}

	var users []list.Item

	for _, v := range fakeUsers {
		users = append(users, list.Item(v))
	}

	return users
}

func GetFakeChannels() []list.Item {
	cwd, _ := os.Getwd()
	fakeChannelsPath := filepath.Join(cwd, "internal", "ui", "fakeChannels.txt")
	testJson, err := os.ReadFile(fakeChannelsPath)

	if err != nil {
		fmt.Println(err.Error())
	}
	var fakeChannels []ChannelInfo
	err = json.Unmarshal(testJson, &fakeChannels)

	if err != nil {
		fmt.Println(err.Error())
	}

	var channels []list.Item

	for _, v := range fakeChannels {
		channels = append(channels, list.Item(v))
	}

	return channels

}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			currentlyFoucsedOn := m.FocusedOn
			if currentlyFoucsedOn == "sideBar" {
				m.FocusedOn = "mainView"
			} else if currentlyFoucsedOn == "mainView" {
				m.FocusedOn = "input"
			} else {
				m.FocusedOn = "sideBar"
			}
			return m, nil
		case "c":
			if m.Mode == "users" {
				m.Mode = "channels"
			}
			return m, nil
		case "u":
			if m.Mode == "channels" {
				m.Mode = "users"
			}
			return m, nil
		case "enter":
			if m.Mode == "users" {
				m.SelectedUser = m.Users.SelectedItem().(UserInfo)
			}
			if m.Mode == "channels" {
				m.SelectedChannel = m.Channels.SelectedItem().(ChannelInfo)
			}
			//TODO: kick off new rpc request to js backend to get the conversation for new chat
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width - 4
		m.Height = msg.Height - 4
		m.updateViewport()
	}

	var cmd tea.Cmd

	if m.FocusedOn == "input" {
		m.Input.Focus()
		m.Input, cmd = m.Input.Update(msg)
	} else if m.FocusedOn == "sideBar" {
		m.Input.Blur()
        if m.Mode == "channels" {
            m.Channels, cmd = m.Channels.Update(msg)
        } else if m.Mode == "users"{
			m.Users, cmd = m.Users.Update(msg)
		} else {
			//if we reach at this point we must be in group mode 
			// as of wrting this im not handling the group mode 
			// TODO: handle group mode
		}

	} else {
		m.Vp, cmd = m.Vp.Update(msg)
	}
	return m, cmd
}

func (m *Model) updateViewport() {
	sidebarWidth := m.Width * 30 / 100
	mainWidth := m.Width - sidebarWidth
	contentHeight := m.Height * 90 / 100

	headerHeight := 2

	w, h := mainWidth*70/100, contentHeight*90/100-headerHeight
	m.Vp.Width = w
	m.Vp.Height = h

	m.Vp.YPosition = headerHeight

	m.Vp.SetContent(formatMessages(m.Conversations))
	m.Vp.GotoBottom()
}

func getItemBorder(isSelected bool) lipgloss.Border {
	if isSelected {
		return lipgloss.DoubleBorder()
	}
	return lipgloss.NormalBorder()
}

func setItemStyles(m Model) string {
	sidebarWidth := m.Width * 30 / 100
	mainWidth := m.Width - sidebarWidth
	contentHeight := m.Height * 90 / 100
	inputHeight := m.Height - contentHeight
	m.Users.SetHeight(contentHeight - 4)
	m.Users.SetWidth(sidebarWidth)
	m.Channels.SetWidth(sidebarWidth)
	m.Channels.SetHeight(contentHeight - 4)

	mainStyle := getMainStyle(mainWidth, contentHeight, m)

	var userNameOrChannelName string
	if m.Mode == "users" {
		userNameOrChannelName = m.SelectedUser.Title()
	}
	if m.Mode == "channels" {
		userNameOrChannelName = m.SelectedChannel.FilterValue()
	}

	title := titleStyle.Render(userNameOrChannelName)
	line := strings.Repeat("─", max(0, mainWidth-4-lipgloss.Width(title)))
	headerView := lipgloss.JoinVertical(lipgloss.Center, title, line)

	mainContent := lipgloss.JoinVertical(
		lipgloss.Top,
		headerView,
		m.Vp.View(),
	)

	chatView := mainStyle.Render(mainContent)
    
	var sideBarContent string
	if m.Mode == "users" {
		sideBarContent = m.Users.View()
	} else if m.Mode == "channels" {
		sideBarContent = m.Channels.View()
	} else {
		sideBarContent = ""
	}

	sidebar := getSideBarStyles(sidebarWidth, contentHeight, m).Render(sideBarContent)

	if m.FocusedOn == "input" {
		m.Input.Focus()
	}
	inputView := getInputStyle(m, inputHeight).Render(m.Input.View())
	row := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chatView)
	ui := lipgloss.JoinVertical(lipgloss.Top, row, inputView)

	return ui
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) View() string {
	m.Users.Title = "Chats"
	m.Channels.Title = "Channels"
	m.Channels.SetShowStatusBar(false)
	m.Users.SetShowStatusBar(false)
	ui := setItemStyles(m)
	return ui
}
