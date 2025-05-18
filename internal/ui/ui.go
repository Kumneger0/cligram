package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LastSeen struct {
	Type   string
	Time   *time.Time
	Status *string
}

func (ls *LastSeen) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ls.Type, ls.Time, ls.Status = "", nil, nil
		return nil
	}

	var aux struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("LastSeen: cannot unmarshal wrapper: %w", err)
	}

	ls.Type = aux.Type

	switch aux.Type {
	case "time":
		var t time.Time
		if err := json.Unmarshal(aux.Value, &t); err != nil {
			return fmt.Errorf("LastSeen: invalid time value: %w", err)
		}
		ls.Time = &t
		ls.Status = nil

	case "status":
		var s string
		if err := json.Unmarshal(aux.Value, &s); err != nil {
			return fmt.Errorf("LastSeen: invalid status value: %w", err)
		}
		ls.Status = &s
		ls.Time = nil

	default:
		return fmt.Errorf("LastSeen: unknown type %q", aux.Type)
	}
	return nil
}

type UserInfo struct {
	FirstName   string   `json:"firstName"`
	IsBot       bool     `json:"isBot"`
	PeerID      string   `json:"peerId"`
	AccessHash  string   `json:"accessHash"`
	UnreadCount int      `json:"unreadCount"`
	LastSeen    LastSeen `json:"lastSeen"`
	IsOnline    bool     `json:"isOnline"`
}

var (
	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true)
)

func (u UserInfo) Title() string {
	return u.FirstName
}

func (u UserInfo) FilterValue() string {
	return u.FirstName
}

type CustomDelegate struct {
	list.DefaultDelegate
}

func (d CustomDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	entry, ok := item.(UserInfo)
	if !ok {
		return
	}

	str := lipgloss.NewStyle().Width(50).Render(entry.Title())
	if index == m.Index() {
		fmt.Fprint(w, selectedStyle.Render(" "+str+" "))
	} else {
		fmt.Fprint(w, normalStyle.Render(" "+str+" "))
	}
}

func GetFakeData() []list.Item {
	testJson := `[
        {
            "firstName": "Alice",
            "isBot": false,
            "peerId": "12345678901234567890",
            "accessHash": "98765432109876543210",
            "unreadCount": 0,
            "lastSeen": { "type": "time", "value": "2025-05-18T08:15:30Z" },
            "isOnline": true
        },
        {
            "firstName": "Botify",
            "isBot": true,
            "peerId": "11111111111111111111",
            "accessHash": "22222222222222222222",
            "unreadCount": 5,
            "lastSeen": { "type": "status", "value": "typing..." },
            "isOnline": true
        },
        {
            "firstName": "Carlos",
            "isBot": false,
            "peerId": "33333333333333333333",
            "accessHash": "44444444444444444444",
            "unreadCount": 2,
            "lastSeen": null,
            "isOnline": false
        },
        {
            "firstName": "Dana",
            "isBot": false,
            "peerId": "55555555555555555555",
            "accessHash": "66666666666666666666",
            "unreadCount": 10,
            "lastSeen": { "type": "time", "value": "2025-05-17T22:45:00Z" },
            "isOnline": false
        },
        {
            "firstName": "EchoBot",
            "isBot": true,
            "peerId": "77777777777777777777",
            "accessHash": "88888888888888888888",
            "unreadCount": 0,
            "lastSeen": { "type": "status", "value": "awaiting input" },
            "isOnline": false
        },
        {
            "firstName": "François",
            "isBot": false,
            "peerId": "99999999999999999999",
            "accessHash": "00000000000000000000",
            "unreadCount": 7,
            "lastSeen": { "type": "time", "value": "2025-05-18T05:00:00Z" },
            "isOnline": true
        },
        {
            "firstName": "Gina",
            "isBot": false,
            "peerId": "10101010101010101010",
            "accessHash": "20202020202020202020",
            "unreadCount": 3,
            "lastSeen": { "type": "status", "value": "away" },
            "isOnline": false
        },
        {
            "firstName": "Hubert",
            "isBot": false,
            "peerId": "30303030303030303030",
            "accessHash": "40404040404040404040",
            "unreadCount": 1,
            "lastSeen": null,
            "isOnline": false
        },
        {
            "firstName": "Ivy",
            "isBot": false,
            "peerId": "50505050505050505050",
            "accessHash": "60606060606060606060",
            "unreadCount": 12,
            "lastSeen": { "type": "time", "value": "2025-05-18T12:00:00Z" },
            "isOnline": true
        },
        {
            "firstName": "JasperBot",
            "isBot": true,
            "peerId": "70707070707070707070",
            "accessHash": "80808080808080808080",
            "unreadCount": 4,
            "lastSeen": { "type": "status", "value": "processing" },
            "isOnline": true
        }
    ]`
	var fakeUsers []UserInfo
	err := json.Unmarshal([]byte(testJson), &fakeUsers)

	if err != nil {
		fmt.Println(err.Error())
	}

	var users []list.Item

	for _, v := range fakeUsers {
		users = append(users, list.Item(v))
	}

	return users
}

type Model struct {
	Users list.Model
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
		case "enter":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.Users, cmd = m.Users.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.Users.View()
}
