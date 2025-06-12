package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/cligram/internal/rpc"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case MessageDeletionConfrimResponseMsg:
		if msg.yes {
			var peer rpc.PeerInfo
			selectedItemInChat := m.ChatUI.SelectedItem().(rpc.FormattedMessage)
			var cType rpc.ChatType

			if m.Mode == ModeUsers {
				peer = rpc.PeerInfo{
					PeerID:     m.SelectedUser.PeerID,
					AccessHash: m.SelectedUser.AccessHash,
				}
				cType = rpc.UserChat
			}

			if m.Mode == ModeChannels {
				peer = rpc.PeerInfo{
					PeerID:     m.SelectedChannel.ChannelID,
					AccessHash: m.SelectedChannel.AccessHash,
				}
				cType = rpc.ChannelChat
			}

			if m.Mode == ModeGroups {
				peer = rpc.PeerInfo{
					PeerID:     m.SelectedGroup.ChannelID,
					AccessHash: m.SelectedGroup.AccessHash,
				}
				cType = rpc.GroupChat
			}

			response, err := rpc.RpcClient.DeleteMessage(peer, int(selectedItemInChat.ID), cType)
			if err != nil {
				m.IsModalVisible = true
				m.ModalContent = GetModalContent(err.Error())
				return m, nil
			}

			if response.Error != nil {
				m.IsModalVisible = true
				m.ModalContent = GetModalContent(response.Error.Message)
				return m, nil
			}

			if response.Result.Status == "success" {
				var updatedConversations []rpc.FormattedMessage
				for _, v := range m.Conversations {
					if v.ID != selectedItemInChat.ID {
						updatedConversations = append(updatedConversations, v)
					}
				}
				m.Conversations = updatedConversations
				cmd := m.ChatUI.SetItems(formatMessages(updatedConversations))
				return m, cmd
			}

			return m, nil
		}

	case rpc.GetMessaegsMsg:
		if msg.Err != nil {
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			m.MainViewLoading = false
			return m, nil
		}
		m.Conversations = msg.Messages.Result
		m.updateConverstaions()
		m.MainViewLoading = false
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+a":
			if m.FocusedOn == Input {
				m.IsFilepickerVisible = true
				m.FocusedOn = Mainview
				return m, nil
			}
			if m.IsFilepickerVisible {
				m.IsFilepickerVisible = false
				m.FocusedOn = Input
			}
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if !m.IsFilepickerVisible {
				return changeFocusMode(&m, "tab")
			}
			return m, nil
		case "m":

			if m.FocusedOn != Input {
				m.IsModalVisible = true
				return m, nil
			}
			input, cmd := m.Input.Update(msg)
			m.Input = input
			return m, cmd
		case "c":
			return changeSideBarMode(&m, "c")
		case "u":
			return changeSideBarMode(&m, "u")
		case "g":
			return changeSideBarMode(&m, "g")
		case "enter":
			if m.FocusedOn == Input {
				return sendMessage(&m)
			}
			if m.FocusedOn == SideBar {
				return handleUserChange(&m)
			}
			selectedMessage := m.ChatUI.SelectedItem()
			if selectedMessage != nil {
				//TODO: show open list of options in modal for message action
			}
		case "r":
			if m.FocusedOn == Mainview {
				canWrite := (m.Mode == ModeUsers || m.Mode == ModeGroups) || (m.Mode == ModeChannels && m.SelectedChannel.IsCreator)
				if canWrite {
					m.IsReply = true
					if selectedMessage, ok := m.ChatUI.SelectedItem().(rpc.FormattedMessage); ok {
						m.FocusedOn = Input
						m.ReplyTo = &selectedMessage
					}
					return m, nil
				}
			}
		case "d":
			if m.FocusedOn == Mainview {
				selectedItem := m.ChatUI.SelectedItem().(rpc.FormattedMessage)
				return m, func() tea.Msg {
					return OpenModalMsg{
						ModalMode: ModalModeDeleteMessage,
						Message:   &selectedItem,
					}
				}
			}
		case "f":
			if m.FocusedOn == Mainview {
				selectedMessage, ok := m.ChatUI.SelectedItem().(rpc.FormattedMessage)
				var from list.Item

				if m.Mode == ModeUsers {
					from = m.SelectedUser
				}

				if m.Mode == ModeChannels {
					from = m.SelectedChannel
				}

				if m.Mode == ModeGroups {
					from = &m.SelectedGroup
				}

				if ok {
					return m, func() tea.Msg {
						return OpenModalMsg{
							ModalMode: ModalModeForwardMessage,
							Message:   &selectedMessage,
							UsersList: &m.Users,
							FromPeer:  &from,
						}
					}
				}
			}
		}
	case rpc.UserChatsMsg:
		if msg.Err != nil {
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}

		duplicatedUsers := msg.Response.Result
		var users []list.Item
		for _, du := range duplicatedUsers {
			users = append(users, du)
		}
		m.Users.SetItems(users)
		return m, nil
	case rpc.UserChannelMsg:
		if msg.Err != nil {
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}

		duplicatedUsers := msg.Response.Result
		var channels []list.Item
		for _, du := range duplicatedUsers {
			channels = append(channels, du)
		}
		m.Channels.SetItems(channels)
		return m, nil
	case rpc.UserGroupsMsg:
		if msg.Err != nil {
			m.IsModalVisible = true
			m.ModalContent = GetModalContent(msg.Err.Error())
			return m, nil
		}

		duplicatedGroups := msg.Response.Result
		var groups []list.Item
		for _, du := range duplicatedGroups {
			groups = append(groups, du)
		}
		m.Groups.SetItems(groups)
		return m, nil

	case ForwardMsg:
		messageToBeForwarded := msg.msg
		reciever := *msg.reciever
		fromPeer := *msg.fromPeer

		var from rpc.PeerInfo
		var toPeer rpc.PeerInfo
		var cType rpc.ChatType

		fromUser, ok := fromPeer.(rpc.UserInfo)
		if ok {
			from.PeerID = fromUser.PeerID
			from.AccessHash = fromUser.AccessHash
			cType = "user"
		}

		fromChannelOrGroup, ok := fromPeer.(rpc.ChannelAndGroupInfo)
		if ok {
			from.PeerID = fromChannelOrGroup.ChannelID
			from.AccessHash = fromChannelOrGroup.AccessHash
			cType = "channel"
		}

		userOrChannel, ok := reciever.(rpc.UserInfo)
		if ok {
			toPeer.PeerID = userOrChannel.PeerID
			toPeer.AccessHash = userOrChannel.AccessHash
		}

		channelOrGroup, ok := reciever.(rpc.ChannelAndGroupInfo)
		if ok {
			toPeer.PeerID = channelOrGroup.ChannelID
			toPeer.AccessHash = channelOrGroup.AccessHash
		}
		messageIDs := []int{int(messageToBeForwarded.ID)}

		response, err := rpc.RpcClient.ForwardMessages(from, messageIDs, toPeer, cType)
		if err != nil {
			//TODO:show toast message
			return m, nil
		}

		if response.Error != nil {
			//TODO:show toast message
		}

		return m, nil

	case tea.WindowSizeMsg:
		m.Width = msg.Width - 4
		m.Height = msg.Height - 4
		m.updateConverstaions()
	case SelectSearchedUserResult:
		if msg.user != nil {
			m.SelectedUser = *msg.user
			index := getUserIndex(m, *msg.user)
			if index != -1 {
				m.Users.Select(index)
				m.FocusedOn = SideBar
				m.Mode = ModeUsers
				return handleUserChange(&m)
			}
			newUpdatedUsers := append(m.Users.Items(), *msg.user)
			updateUserCmd := m.Users.SetItems(newUpdatedUsers)

			index = getUserIndex(m, *msg.user)

			if index != -1 {
				m.Users.Select(index)
				m.FocusedOn = SideBar
				m.Mode = ModeUsers
				m, handleUserChangeCmd := handleUserChange(&m)
				return m, tea.Batch(updateUserCmd, handleUserChangeCmd)
			}

			return m, updateUserCmd
		}

		if msg.channel != nil {
			m.SelectedChannel = *msg.channel
			index := getChannelIndex(m, *msg.channel)

			if index != -1 {
				m.Channels.Select(index)
				m.FocusedOn = SideBar
				m.Mode = ModeChannels
				return handleUserChange(&m)
			}

			newUpdatedChannelsList := append(m.Channels.Items(), *msg.channel)
			setItemsCmd := m.Channels.SetItems(newUpdatedChannelsList)
			index = getChannelIndex(m, *msg.channel)

			if index != -1 {
				m.Channels.Select(index)
				m.FocusedOn = SideBar
				m.Mode = ModeChannels
				m, handleChangeUserCmd := handleUserChange(&m)
				return m, tea.Batch(setItemsCmd, handleChangeUserCmd)
			}
			return m, setItemsCmd
		}

		if msg.group != nil {
			m.SelectedGroup = *msg.group
			index := getGroupIndex(m, *msg.group)

			if index != -1 {
				m.Groups.Select(index)
				m.FocusedOn = SideBar
				m.Mode = ModeGroups
				return handleUserChange(&m)
			}

			newUpdatedGroupsList := append(m.Groups.Items(), *msg.group)
			setItemsCmd := m.Groups.SetItems(newUpdatedGroupsList)
			index = getGroupIndex(m, *msg.group)

			if index != -1 {
				m.Groups.Select(index)
				m.FocusedOn = SideBar
				m.Mode = ModeGroups
				m, handleChangeUserCmd := handleUserChange(&m)
				return m, tea.Batch(setItemsCmd, handleChangeUserCmd)
			}
			return m, setItemsCmd
		}
	}
	return updateFocusedComponent(&m, msg)
}
