package ui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/telegram"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"go.dalton.dog/bubbleup"
)

var telegramDomainPrefixes = []string{
	"https://t.me/",
	"http://t.me/",
	"t.me/",
	"https://telegram.me/",
	"http://telegram.me/",
	"telegram.me/",
	"https://www.t.me/",
	"http://www.t.me/",
	"www.t.me/",
	"https://www.telegram.me/",
	"http://www.telegram.me/",
	"www.telegram.me/",
}

// RouteParsedLinkMsg is emitted to trigger link routing asynchronously.
type RouteParsedLinkMsg struct {
	Link *types.ParsedLink
}

// ParseTelegramLink inspects a raw string and parses it into a structured ParsedLink if valid.
func ParseTelegramLink(raw string) *types.ParsedLink {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	// 1. tg:// deep links
	if strings.HasPrefix(trimmed, "tg://") {
		u, err := url.Parse(trimmed)
		if err == nil {
			switch u.Host {
			case "join":
				invite := u.Query().Get("invite")
				if invite != "" {
					return &types.ParsedLink{
						Type:       types.LinkTypeInvite,
						Raw:        trimmed,
						InviteHash: cleanHash(invite),
					}
				}
			case "resolve":
				domain := u.Query().Get("domain")
				if domain != "" {
					var mid int
					if post := u.Query().Get("post"); post != "" {
						mid, _ = strconv.Atoi(post)
					}
					return &types.ParsedLink{
						Type:      types.LinkTypePublic,
						Raw:       trimmed,
						Username:  cleanUsername(domain),
						MessageID: mid,
					}
				}
			}
		}
	}

	// 2. Direct invite hash starting with "+" e.g. "+abc123XYZ"
	if strings.HasPrefix(trimmed, "+") && len(trimmed) > 1 {
		hash := cleanHash(trimmed[1:])
		if hash != "" {
			return &types.ParsedLink{
				Type:       types.LinkTypeInvite,
				Raw:        trimmed,
				InviteHash: hash,
			}
		}
	}

	// 3. Direct handle starting with "@" e.g. "@username"
	if strings.HasPrefix(trimmed, "@") && len(trimmed) > 1 {
		username := cleanUsername(trimmed[1:])
		if username != "" {
			return &types.ParsedLink{
				Type:     types.LinkTypePublic,
				Raw:      trimmed,
				Username: username,
			}
		}
	}

	// 4. Web URLs with telegram domains (t.me, telegram.me, etc.)
	var pathPart string
	matchedPrefix := false
	for _, prefix := range telegramDomainPrefixes {
		if after, ok := strings.CutPrefix(trimmed, prefix); ok {
			pathPart = after
			matchedPrefix = true
			break
		}
	}

	if !matchedPrefix {
		return nil
	}

	// Strip query parameters and fragments from path
	if idx := strings.IndexAny(pathPart, "?#"); idx != -1 {
		pathPart = pathPart[:idx]
	}
	pathPart = strings.Trim(pathPart, "/")

	if pathPart == "" {
		return nil
	}

	// 4a. Invite link with "+" prefix: t.me/+hash
	if strings.HasPrefix(pathPart, "+") {
		hash := cleanHash(pathPart[1:])
		if hash != "" {
			return &types.ParsedLink{
				Type:       types.LinkTypeInvite,
				Raw:        trimmed,
				InviteHash: hash,
			}
		}
		return nil
	}

	// 4b. Legacy invite link: t.me/joinchat/hash
	if after, ok := strings.CutPrefix(pathPart, "joinchat/"); ok {
		hash := cleanHash(after)
		if hash != "" {
			return &types.ParsedLink{
				Type:       types.LinkTypeInvite,
				Raw:        trimmed,
				InviteHash: hash,
			}
		}
		return nil
	}

	// 4c. Internal private channel message link: t.me/c/<id>/<mid>
	if after, ok := strings.CutPrefix(pathPart, "c/"); ok {
		parts := strings.Split(after, "/")
		if len(parts) > 0 {
			channelID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil && channelID > 0 {
				var msgID int
				if len(parts) >= 2 {
					msgID, _ = strconv.Atoi(parts[1])
				}
				return &types.ParsedLink{
					Type:      types.LinkTypeInternalMessage,
					Raw:       trimmed,
					ChannelID: channelID,
					MessageID: msgID,
				}
			}
		}
		return nil
	}

	// 4d. Public handle or post: t.me/<username>/<mid> or t.me/<username>
	parts := strings.Split(pathPart, "/")
	username := cleanUsername(parts[0])
	if username == "" {
		return nil
	}

	var msgID int
	if len(parts) >= 2 {
		msgID, _ = strconv.Atoi(parts[1])
	}

	return &types.ParsedLink{
		Type:      types.LinkTypePublic,
		Raw:       trimmed,
		Username:  username,
		MessageID: msgID,
	}
}

// ExtractTelegramLinks scans a text message for any embedded Telegram links.
func ExtractTelegramLinks(text string) []*types.ParsedLink {
	words := strings.Fields(text)
	var links []*types.ParsedLink
	for _, w := range words {
		cleaned := strings.Trim(w, "()[]{}<>,.;:\"'`!?")
		if pl := ParseTelegramLink(cleaned); pl != nil {
			links = append(links, pl)
		}
	}
	return links
}

func cleanHash(s string) string {
	if idx := strings.IndexAny(s, "?#&/"); idx != -1 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

func cleanUsername(s string) string {
	if idx := strings.IndexAny(s, "?#&/"); idx != -1 {
		s = s[:idx]
	}
	return strings.TrimSpace(strings.TrimPrefix(s, "@"))
}

// routeTelegramLink dispatches appropriate actions for a parsed link.
func routeTelegramLink(m *Model, link *types.ParsedLink) tea.Cmd {
	if link == nil {
		return nil
	}

	switch link.Type {
	case types.LinkTypeInvite:
		if telegram.Cligram == nil || telegram.Cligram.Context() == nil {
			return nil
		}
		return telegram.Cligram.CheckChatInvite(telegram.Cligram.Context(), link.InviteHash)

	case types.LinkTypeInternalMessage:
		return handleInternalChannelLink(m, link)

	case types.LinkTypePublic:
		var msgIDs []string
		if link.MessageID > 0 {
			msgIDs = []string{strconv.Itoa(link.MessageID)}
		}
		return func() tea.Msg {
			return types.OpenNewChatWithPeerMsg{
				Chat: &types.EntityPreviewInfo{
					Entity:    link.Username,
					MessageID: msgIDs,
				},
			}
		}
	}

	return nil
}

func handleInternalChannelLink(m *Model, link *types.ParsedLink) tea.Cmd {
	channelIDStr := strconv.FormatInt(link.ChannelID, 10)

	// Check channels list
	for i, it := range m.Channels.Items() {
		if ch, ok := it.(types.ChannelInfo); ok && (ch.ID == channelIDStr || strings.TrimPrefix(ch.ID, "-100") == channelIDStr) {
			m.Mode = ModeChannels
			m.Channels.Select(i)
			m.SelectedChannel = ch

			var offsetID *int
			if link.MessageID > 0 {
				offsetID = &link.MessageID
			}
			highlightCmd := func() tea.Msg {
				if offsetID == nil {
					return nil
				}
				return types.ShouldHighlightSpecificMessageMsg{MessageID: *offsetID}
			}
			newM, navCmd := handleUserChange(m, offsetID, highlightCmd)
			*m = newM
			return navCmd
		}
	}

	// Check groups list
	for i, it := range m.Groups.Items() {
		if gr, ok := it.(types.ChannelInfo); ok && (gr.ID == channelIDStr || strings.TrimPrefix(gr.ID, "-100") == channelIDStr) {
			m.Mode = ModeGroups
			m.Groups.Select(i)
			m.SelectedGroup = gr

			var offsetID *int
			if link.MessageID > 0 {
				offsetID = &link.MessageID
			}
			highlightCmd := func() tea.Msg {
				if offsetID == nil {
					return nil
				}
				return types.ShouldHighlightSpecificMessageMsg{MessageID: *offsetID}
			}
			newM, navCmd := handleUserChange(m, offsetID, highlightCmd)
			*m = newM
			return navCmd
		}
	}

	// Check All list
	for i, it := range m.All.Items() {
		if ch, ok := it.(types.ChannelInfo); ok && (ch.ID == channelIDStr || strings.TrimPrefix(ch.ID, "-100") == channelIDStr) {
			m.Mode = ModeAll
			m.All.Select(i)
			if ch.IsBroadcast {
				m.SelectedChannel = ch
			} else {
				m.SelectedGroup = ch
			}

			var offsetID *int
			if link.MessageID > 0 {
				offsetID = &link.MessageID
			}
			highlightCmd := func() tea.Msg {
				if offsetID == nil {
					return nil
				}
				return types.ShouldHighlightSpecificMessageMsg{MessageID: *offsetID}
			}
			newM, navCmd := handleUserChange(m, offsetID, highlightCmd)
			*m = newM
			return navCmd
		}
	}

	// Channel not found in user's joined chats
	m.Alert = m.Alert.WithAllowEscToClose().WithPosition(bubbleup.TopLeftPosition)
	return m.Alert.NewAlertCmd(bubbleup.ErrorKey, fmt.Sprintf("Private channel link (ID: %d). You are not a member of this chat.", link.ChannelID))
}
