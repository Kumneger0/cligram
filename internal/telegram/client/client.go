package client

import (
	"context"
	"errors"
	"log/slog"
	mathRand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"golang.org/x/time/rate"

	configManager "github.com/kumneger0/cligram/internal/config"
	"github.com/kumneger0/cligram/internal/telegram/shared"
	"github.com/kumneger0/cligram/internal/telegram/types"

	floodwait "github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
)

type Client struct {
	*telegram.Client
	ctx           context.Context
	updateChannel chan types.Notification
}

type Config struct {
	AppID         int
	AppHash       string
	UpdateChannel chan types.Notification
}

var Cligram *telegram.Client

func NewClient(ctx context.Context, config Config) (*Client, error) {
	sessionStorage, err := newFileSessionStorage()
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeSessionFailed, "failed to create session storage", err)
	}

	updateHandler := newUpdateHandler(config.UpdateChannel)

	waiter := floodwait.NewSimpleWaiter()

	options := telegram.Options{
		Middlewares: []telegram.Middleware{
			waiter,
			ratelimit.New(rate.Every(time.Millisecond*200), 3),
		},
		SessionStorage: sessionStorage,
		UpdateHandler:  updateHandler,
		NoUpdates:      false,
		OnDead: func(err error) {
			slog.Error("telegram connection is dead", "error", err)
		},
	}

	Cligram = telegram.NewClient(config.AppID, config.AppHash, options)

	return &Client{
		Client:        Cligram,
		ctx:           ctx,
		updateChannel: config.UpdateChannel,
	}, nil
}

func NewClientFromEnv(ctx context.Context, updateChannel chan types.Notification, telegramAPIID, telegramAPIHash string) (*Client, error) {
	appID, err := strconv.Atoi(telegramAPIID)
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeAuthFailed, "invalid TELEGRAM_API_ID", err)
	}

	return NewClient(ctx, Config{
		AppID:         appID,
		AppHash:       telegramAPIHash,
		UpdateChannel: updateChannel,
	})
}

func (c *Client) GetAPI() *tg.Client {
	return c.Client.API()
}

func (c *Client) Context() context.Context {
	return c.ctx
}

func (c *Client) SendMessage(ctx context.Context, req types.SendMessageRequest) tea.Cmd {
	if req.IsFile {
		return c.sendMedia(ctx, req.Peer, req.FilePath, req.Message, parseReplyID(req.ReplyToMessageID))
	}
	return c.sendText(ctx, req.Peer, req.Message, parseReplyID(req.ReplyToMessageID))
}

func (c *Client) sendText(ctx context.Context, peer types.Peer, text string, replyTo *int) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := shared.ConvertPeerToInputPeer(peer)
		if err != nil {
			return types.SendMessageMsg{Err: types.NewSendMessageError(err)}
		}

		var replyToClass tg.InputReplyToClass
		if replyTo != nil {
			replyToClass = &tg.InputReplyToMessage{ReplyToMsgID: *replyTo}
		}

		_, err = c.GetAPI().MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer:     inputPeer,
			ReplyTo:  replyToClass,
			Message:  text,
			RandomID: mathRand.Int63(),
		})
		if err != nil {
			return types.SendMessageMsg{Err: types.NewSendMessageError(err)}
		}

		return types.SendMessageMsg{Response: &types.SendMessageResponse{}}
	}
}

func (c *Client) sendMedia(ctx context.Context, peer types.Peer, filePath string, caption string, replyTo *int) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := shared.ConvertPeerToInputPeer(peer)
		if err != nil {
			return types.SendMessageMsg{Err: types.NewSendMessageError(err)}
		}

		messageID, err := c.sendMediaFile(ctx, filePath, caption, inputPeer, replyTo)
		if err != nil {
			return types.SendMessageMsg{Err: types.NewSendMessageError(err)}
		}

		return types.SendMessageMsg{Response: &types.SendMessageResponse{MessageID: messageID}}
	}
}

func (c *Client) sendMediaFile(ctx context.Context, path string, caption string, peer tg.InputPeerClass, replyTo *int) (*int, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, types.NewTelegramError(types.ErrorCodeInvalidFile, "file is a directory", nil)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	upload := uploader.NewUpload(filepath.Base(path), file, fileInfo.Size())
	fileUpload, err := uploader.NewUploader(c.GetAPI()).Upload(ctx, upload)
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeUploadFailed, "failed to upload file", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])

	attributes := []tg.DocumentAttributeClass{
		&tg.DocumentAttributeFilename{FileName: filepath.Base(path)},
	}

	var replyToClass tg.InputReplyToClass
	if replyTo != nil {
		replyToClass = &tg.InputReplyToMessage{ReplyToMsgID: *replyTo}
	}

	sendMediaUpdateClass, err := c.GetAPI().MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer: peer,
		Media: &tg.InputMediaUploadedDocument{
			File:       fileUpload,
			MimeType:   mimeType,
			Attributes: attributes,
		},
		Message:  caption,
		RandomID: mathRand.Int63(),
		ReplyTo:  replyToClass,
	})
	if err != nil {
		return nil, err
	}

	var id *int
	switch u := sendMediaUpdateClass.(type) {
	case *tg.Updates:
		for _, up := range u.Updates {
			switch x := up.(type) {
			case *tg.UpdateMessageID:
				i := x.ID
				id = &i
			case *tg.UpdateNewMessage:
				if m, ok := x.Message.(*tg.Message); ok {
					i := m.ID
					id = &i
				}
			case *tg.UpdateNewChannelMessage:
				if m, ok := x.Message.(*tg.Message); ok {
					i := m.ID
					id = &i
				}
			}
		}
	case *tg.UpdatesCombined:
		for _, up := range u.Updates {
			if x, ok := up.(*tg.UpdateMessageID); ok {
				i := x.ID
				id = &i
			}
		}
	case *tg.UpdateShortSentMessage:
		i := u.ID
		id = &i
	}
	return id, nil
}

func (c *Client) GetMessages(ctx context.Context, req types.GetMessagesRequest) tea.Cmd {
	return c.GetChatHistoryCmd(ctx, req.Peer, req.Limit, req.OffsetID)
}

func (c *Client) GetUserChats(ctx context.Context, chatType types.ChatType, offsetDate, offsetID int) tea.Cmd {
	return c.GetUserChatsCmd(ctx, chatType == types.BotChat, offsetDate, offsetID)
}

func (c *Client) GetUserChannels(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) tea.Cmd {
	return c.GetChannelsCmd(ctx, isBroadCast, offsetDate, offsetID)
}

func (c *Client) GetChatHistoryCmd(ctx context.Context, peer types.Peer, limit int, offsetID *int) tea.Cmd {
	return func() tea.Msg {
		messages, err := c.GetChatHistory(ctx, peer, limit, offsetID)
		if err != nil {
			return types.GetMessagesMsg{Messages: [50]types.FormattedMessage{}, Err: err}
		}
		var msgArray [50]types.FormattedMessage
		copy(msgArray[:], messages)
		return types.GetMessagesMsg{Messages: msgArray}
	}
}

func (c *Client) GetChatHistory(ctx context.Context, peer types.Peer, limit int, offsetID *int) ([]types.FormattedMessage, error) {
	inputPeer, err := shared.ConvertPeerToInputPeer(peer)
	if err != nil {
		return nil, err
	}

	req := &tg.MessagesGetHistoryRequest{Peer: inputPeer, Limit: limit}
	if offsetID != nil {
		req.OffsetID = *offsetID
	}

	history, err := c.GetAPI().MessagesGetHistory(ctx, req)
	if err != nil {
		return nil, types.NewGetMessagesError(err)
	}

	entities, err := shared.GetMessageAndUserClasses(history)
	if err != nil {
		return nil, err
	}

	areWeInUserModeOrBotMode := peer.ChatType == types.BotChat || peer.ChatType == types.UserChat

	var userInfo *types.UserInfo
	var channel *types.ChannelInfo

	if areWeInUserModeOrBotMode {
		if id, err := strconv.ParseInt(peer.ID, 10, 64); err == nil {
			userInfo = getUserFromClasses(entities.Users, id)
		}
	} else {
		if id, err := strconv.ParseInt(peer.ID, 10, 64); err == nil {
			channel = getChannelFromClasses(entities.Chats, id)
		}
	}

	var formattedMessages []types.FormattedMessage
	for _, msgClass := range entities.Messages {
		msg, ok := msgClass.(*tg.Message)
		if !ok {
			continue
		}
		if areWeInUserModeOrBotMode {
			formattedMessage := shared.FormatMessage(msg, userInfo, entities.Messages)
			formattedMessage.PeerID = &peer.ID
			formattedMessages = append(formattedMessages, *formattedMessage)
		} else if peer.ChatType == types.GroupChat {
			fromID, ok := msg.FromID.(*tg.PeerUser)
			if ok {
				ui := getUserFromClasses(entities.Users, fromID.UserID)
				formattedMessage := shared.FormatMessage(msg, ui, entities.Messages)
				formattedMessage.PeerID = &peer.ID
				formattedMessages = append(formattedMessages, *formattedMessage)
			} else {
				formattedMessage := shared.FormatMessage(msg, channel, entities.Messages)
				formattedMessage.PeerID = &peer.ID
				formattedMessages = append(formattedMessages, *formattedMessage)
			}
		} else {
			formattedMessage := shared.FormatMessage(msg, channel, entities.Messages)
			formattedMessage.PeerID = &peer.ID
			formattedMessages = append(formattedMessages, *formattedMessage)
		}
	}

	slices.Reverse(formattedMessages)
	return formattedMessages, nil
}

func (c *Client) GetUserChatsCmd(ctx context.Context, isBot bool, offsetDate, offsetID int) tea.Cmd {
	return func() tea.Msg {
		result, err := c.getUserChats(ctx, isBot, offsetDate, offsetID)
		return types.UserChatsMsg{Response: &result, Err: err}
	}
}

func (c *Client) GetChannelsCmd(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) tea.Cmd {
	return func() tea.Msg {
		result, err := c.getChannels(ctx, isBroadCast, offsetDate, offsetID)
		if isBroadCast {
			return types.ChannelsMsg{Response: &result, Err: err}
		}
		return types.GroupsMsg{Response: &result, Err: err}
	}
}

func (c *Client) GetAllChats(ctx context.Context, offsetDate int, offsetID int) (types.GetAllChatsResponse, error) {
	ds, err := c.getAllDialogs(ctx, offsetDate, offsetID)
	if err != nil {
		return types.GetAllChatsResponse{}, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "failed to get dialogs", err)
	}
	if ds == nil {
		return types.GetAllChatsResponse{PrivateChats: []types.UserInfo{}, Channels: []types.ChannelInfo{}, Groups: []types.ChannelInfo{}}, nil
	}

	var users []types.UserInfo
	for _, tgUser := range ds.Users {
		u := shared.ConvertTGUserToUserInfo(tgUser)
		u.UnreadCount = getUnreadCount(ds.Dialogs, tgUser.ID)
		u.NotifySettings = getNotifySettings(ds.Dialogs, tgUser.ID)
		users = append(users, *u)
	}

	var channels, groups []types.ChannelInfo
	for _, chatClass := range ds.Chats {
		if channel, ok := chatClass.(*tg.Channel); ok {
			info := convertToChannelInfo(channel)
			if info == nil {
				continue
			}
			info.UnreadCount = getUnreadCount(ds.Dialogs, channel.ID)
			info.NotifySettings = getNotifySettings(ds.Dialogs, channel.ID)
			if channel.Broadcast {
				channels = append(channels, *info)
			} else {
				groups = append(groups, *info)
			}
		}
		if chat, ok := chatClass.(*tg.Chat); ok {
			info := convertToChannelInfo(chat)
			if info == nil {
				continue
			}
			info.UnreadCount = getUnreadCount(ds.Dialogs, chat.ID)
			info.NotifySettings = getNotifySettings(ds.Dialogs, chat.ID)
			groups = append(groups, *info)
		}
	}

	return types.GetAllChatsResponse{
		PrivateChats: users,
		Channels:     channels,
		Groups:       groups,
		OffsetDate:   ds.OffsetDate,
		OffsetID:     ds.OffsetID,
	}, nil
}

func (c *Client) getUserChats(ctx context.Context, isBot bool, offsetDate, offsetID int) (types.GetUserChatsResult, error) {
	ds, err := c.getAllDialogs(ctx, offsetDate, offsetID)
	if err != nil {
		return types.GetUserChatsResult{}, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "failed to get dialogs", err)
	}
	if ds == nil {
		return types.GetUserChatsResult{Data: []types.UserInfo{}}, nil
	}

	var users []types.UserInfo
	for _, tgUser := range ds.Users {
		if tgUser.Bot == isBot {
			u := shared.ConvertTGUserToUserInfo(tgUser)
			u.UnreadCount = getUnreadCount(ds.Dialogs, int64(tgUser.ID))
			users = append(users, *u)
		}
	}

	return types.GetUserChatsResult{Data: users, OffsetDate: ds.OffsetDate, OffsetID: ds.OffsetID}, nil
}

func (c *Client) getChannels(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) (types.GetChannelsResult, error) {
	ds, err := c.getAllDialogs(ctx, offsetDate, offsetID)
	if err != nil {
		return types.GetChannelsResult{}, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "failed to get dialogs", err)
	}
	if ds == nil {
		return types.GetChannelsResult{Data: []types.ChannelInfo{}}, nil
	}

	var channels []types.ChannelInfo
	for _, chatClass := range ds.Chats {
		if channel, ok := chatClass.(*tg.Channel); ok && channel.Broadcast == isBroadCast {
			if info := convertToChannelInfo(channel); info != nil {
				channels = append(channels, *info)
			}
		}
	}

	return types.GetChannelsResult{Data: channels, OffsetDate: ds.OffsetDate, OffsetID: ds.OffsetID}, nil
}

func (c *Client) GetChannelInfo(ctx context.Context, peer types.Peer) (*types.ChannelInfo, error) {
	channelID, err := strconv.ParseInt(peer.ID, 10, 64)
	if err != nil {
		return nil, err
	}
	accessHash, err := strconv.ParseInt(peer.AccessHash, 10, 64)
	if err != nil {
		return nil, err
	}

	chatClass, err := c.GetAPI().ChannelsGetChannels(ctx, []tg.InputChannelClass{
		&tg.InputChannel{ChannelID: channelID, AccessHash: accessHash},
	})
	if err != nil {
		return nil, err
	}

	messageChatClass, ok := chatClass.(*tg.MessagesChats)
	if !ok || len(messageChatClass.Chats) == 0 {
		return nil, errors.New("failed to get channel info")
	}

	chat, ok := messageChatClass.Chats[0].(*tg.Channel)
	if !ok {
		return nil, errors.New("unexpected chat type")
	}

	return &types.ChannelInfo{
		ChannelTitle:      chat.Title,
		Username:          &chat.Username,
		ID:                peer.ID,
		AccessHash:        strconv.FormatInt(chat.AccessHash, 10),
		IsCreator:         chat.Creator,
		IsBroadcast:       chat.Broadcast,
		ParticipantsCount: &chat.ParticipantsCount,
	}, nil
}

func (c *Client) UserInfoFromPeerClass(ctx context.Context, peerClass *tg.PeerUser) *types.UserInfo {
	userClasses, err := c.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{
		&tg.InputUser{UserID: peerClass.UserID},
	})
	if err != nil || len(userClasses) == 0 {
		return nil
	}
	user, ok := userClasses[0].(*tg.User)
	if !ok {
		return nil
	}
	return shared.ConvertTGUserToUserInfo(user)
}

func (c *Client) DeleteMessage(ctx context.Context, req types.DeleteMessageRequest) (types.DeleteMessageResponse, error) {
	_, err := c.GetAPI().MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
		Revoke: true,
		ID:     []int{req.MessageID},
	})
	if err != nil {
		return types.DeleteMessageResponse{Status: "failed"}, types.NewDeleteMessageError(err)
	}
	return types.DeleteMessageResponse{Status: "success"}, nil
}

func (c *Client) EditMessage(ctx context.Context, req types.EditMessageRequest) types.EditMessageMsg {
	inputPeer, err := shared.ConvertPeerToInputPeer(req.Peer)
	if err != nil {
		return types.EditMessageMsg{Err: types.NewEditMessageError(err)}
	}

	_, err = c.GetAPI().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
		Peer:    inputPeer,
		ID:      req.MessageID,
		Message: req.NewMessage,
	})
	if err != nil {
		return types.EditMessageMsg{Err: types.NewEditMessageError(err)}
	}
	return types.EditMessageMsg{Response: true}
}

func (c *Client) ForwardMessages(ctx context.Context, req types.ForwardMessagesRequest) error {
	fromPeer, err := shared.ConvertPeerToInputPeer(req.FromPeer)
	if err != nil {
		return types.NewForwardMessageError(err)
	}

	toPeer, err := shared.ConvertPeerToInputPeer(req.ToPeer)
	if err != nil {
		return types.NewForwardMessageError(err)
	}

	_, err = c.GetAPI().MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: fromPeer,
		ToPeer:   toPeer,
		ID:       req.MessageIDs,
		RandomID: []int64{mathRand.Int63()},
	})
	if err != nil {
		return types.NewForwardMessageError(err)
	}
	return nil
}

func (c *Client) MarkMessagesAsRead(ctx context.Context, req types.MarkAsReadRequest) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := shared.ConvertPeerToInputPeer(req.Peer)
		if err != nil {
			return types.MarkMessagesAsReadMsg{Err: err}
		}

		_, err = c.GetAPI().MessagesReadHistory(ctx, &tg.MessagesReadHistoryRequest{Peer: inputPeer})
		if err != nil {
			return types.MarkMessagesAsReadMsg{Err: err}
		}
		return types.MarkMessagesAsReadMsg{Response: true}
	}
}

func (c *Client) SetUserTyping(ctx context.Context, req types.SetTypingRequest) error {
	inputPeer, err := shared.ConvertPeerToInputPeer(req.Peer)
	if err != nil {
		return err
	}

	_, err = c.GetAPI().MessagesSetTyping(ctx, &tg.MessagesSetTypingRequest{
		Peer:   inputPeer,
		Action: &tg.SendMessageTypingAction{},
	})
	if err != nil {
		return types.NewTelegramError(types.ErrorCodeSendFailed, "failed to set typing", err)
	}
	return nil
}

func (c *Client) GetUserInfo(ctx context.Context, userID int64) (*types.UserInfo, error) {
	userClasses, err := c.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUser{UserID: userID}})
	if err != nil || len(userClasses) == 0 {
		return nil, types.NewUserNotFoundError(userID)
	}
	tgUser, ok := userClasses[0].(*tg.User)
	if !ok {
		return nil, types.NewUserNotFoundError(userID)
	}
	return shared.ConvertTGUserToUserInfo(tgUser), nil
}

func (c *Client) GetUserStatus(ctx context.Context, userID int64) (*types.UserStatus, error) {
	userClasses, err := c.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUser{UserID: userID}})
	if err != nil || len(userClasses) == 0 {
		return nil, types.NewUserNotFoundError(userID)
	}
	tgUser, ok := userClasses[0].(*tg.User)
	if !ok {
		return nil, types.NewUserNotFoundError(userID)
	}

	var lastSeen time.Time
	isOnline := false
	if tgUser.Status != nil {
		switch s := tgUser.Status.(type) {
		case *tg.UserStatusOnline:
			isOnline = true
		case *tg.UserStatusOffline:
			lastSeen = time.Unix(int64(s.WasOnline), 0)
		}
	}
	return &types.UserStatus{IsOnline: isOnline, LastSeen: lastSeen}, nil
}

func (c *Client) SearchUsers(ctx context.Context, searchQuery string) {
	users, err := c.searchUsers(ctx, searchQuery)
	c.updateChannel <- types.Notification{
		SearchResult: &types.SearchUsersMsg{
			Response: &users,
			Err:      err,
		},
	}
}

func (c *Client) searchUsers(ctx context.Context, q string) ([]types.UserInfo, error) {
	searchResult, err := c.GetAPI().ContactsSearch(ctx, &tg.ContactsSearchRequest{Q: q, Limit: 50})
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeUserNotFound, "failed to search users", err)
	}

	var users []types.UserInfo
	for _, userClass := range searchResult.Users {
		if tgUser, ok := userClass.(*tg.User); ok {
			users = append(users, *shared.ConvertTGUserToUserInfo(tgUser))
		}
	}
	return users, nil
}

func (c *Client) GetAvailableReactions(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		result, err := c.GetAPI().MessagesGetAvailableReactions(ctx, 0)
		if err != nil {
			slog.Error(err.Error())
			return types.AvailableReactions{Err: err}
		}

		var reactions []types.Reaction
		switch r := result.(type) {
		case *tg.MessagesAvailableReactions:
			for _, reaction := range r.Reactions {
				reactions = append(reactions, types.Reaction{AvailableReaction: reaction})
			}
		}
		return types.AvailableReactions{Reactions: reactions}
	}
}

func (c *Client) SendReaction(ctx context.Context, req types.SendReactionRequest) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := shared.ConvertPeerToInputPeer(req.Peer)
		if err != nil {
			return types.SendReactionResponseMsg{Err: err}
		}

		_, err = c.GetAPI().MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
			Peer:     inputPeer,
			MsgID:    req.MessageID,
			Reaction: []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: req.Emoticon}},
		})
		if err != nil {
			return types.SendReactionResponseMsg{Err: err, MessageID: req.MessageID, Emoticon: req.Emoticon}
		}
		return types.SendReactionResponseMsg{Response: true, MessageID: req.MessageID, Emoticon: req.Emoticon}
	}
}

func (c *Client) GetAllStories(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		allUserStoriesClass, err := c.GetAPI().StoriesGetAllStories(ctx, &tg.StoriesGetAllStoriesRequest{})
		if err != nil {
			slog.Error(context.Canceled.Error())
			return nil
		}
		allUserStories, ok := allUserStoriesClass.(*tg.StoriesAllStories)
		if !ok {
			return nil
		}

		var AllStories []types.Stories
		for _, peerStorie := range allUserStories.PeerStories {
			peerUser, ok := peerStorie.Peer.(*tg.PeerUser)
			if !ok {
				continue
			}

			userClass, err := c.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUser{UserID: peerUser.UserID}})
			if err != nil || len(userClass) == 0 {
				continue
			}

			tgUser, ok := userClass[0].(*tg.User)
			if !ok {
				continue
			}

			userInfo := shared.ConvertTGUserToUserInfo(tgUser)
			userInfo.HasStories = true

			for _, storyItemClass := range peerStorie.Stories {
				storyItem, ok := storyItemClass.(*tg.StoryItem)
				if !ok {
					continue
				}
				switch item := storyItem.Media.(type) {
				case *tg.MessageMediaDocument:
					documentClass, ok := item.GetDocument()
					if !ok {
						continue
					}
					document, ok := documentClass.(*tg.Document)
					if !ok {
						continue
					}
					AllStories = append(AllStories, types.Stories{UserInfo: *userInfo, ID: storyItem.ID, Data: document.FileReference})
				case *tg.MessageMediaPhoto:
					photoClass, ok := item.GetPhoto()
					if !ok {
						continue
					}
					photo, ok := photoClass.(*tg.Photo)
					if !ok {
						continue
					}
					AllStories = append(AllStories, types.Stories{UserInfo: *userInfo, ID: storyItem.ID, Data: photo.FileReference})
				}
			}
		}
		return types.GetAllStoriesMsg{Stories: AllStories}
	}
}

func (c *Client) GetPeerStories(ctx context.Context, peer types.Peer) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := shared.ConvertPeerToInputPeer(peer)
		if err != nil {
			return types.StoriesDownloadStatusMsg{IDs: []int{}, Err: err, Peer: peer}
		}
		peerUserStories, err := c.GetAPI().StoriesGetPeerStories(ctx, inputPeer)
		if err != nil {
			return types.StoriesDownloadStatusMsg{IDs: []int{}, Err: err, Peer: peer}
		}

		homeDir, _ := os.UserHomeDir()
		cligramDir := filepath.Join(homeDir, ".cligram")
		var readStoriesIDs []int
		for _, v := range peerUserStories.Stories.Stories {
			if storyItem, ok := v.(*tg.StoryItem); ok {
				id, err := shared.DownloadStoryMedia(ctx, c.Client, storyItem, cligramDir, peer.ID)
				if err != nil {
					continue
				}
				readStoriesIDs = append(readStoriesIDs, *id)
			}
		}

		if len(readStoriesIDs) != 0 {
			maxID := slices.Max(readStoriesIDs)
			if configManager.GetConfig().ReadStories {
				if readErr := shared.ReadStories(ctx, c.Client, peer, maxID); readErr != nil {
					slog.Error("failed to mark stories as read", "err", readErr)
				}
			}
			return types.StoriesDownloadStatusMsg{IDs: readStoriesIDs, Done: true, Peer: peer}
		}
		return types.StoriesDownloadStatusMsg{
			IDs:  []int{},
			Done: false,
			Err:  errors.New("failed to download story"),
			Peer: peer,
		}
	}
}

type dialogsResult struct {
	Chats      []tg.ChatClass
	Users      []*tg.User
	Dialogs    []*tg.Dialog
	OffsetDate int
	OffsetID   int
}

func (c *Client) getAllDialogs(ctx context.Context, offsetDate, offsetID int) (*dialogsResult, error) {
	q := query.NewQuery(c.GetAPI())
	it := dialogs.NewIterator(q.GetDialogs().OffsetID(offsetID).OffsetDate(offsetDate).BatchSize(20), 50)

	var result dialogsResult
	result.OffsetDate = -1
	result.OffsetID = -1

	for it.Next(ctx) {
		value := it.Value()

		if dialog, ok := value.Dialog.(*tg.Dialog); ok {
			result.Dialogs = append(result.Dialogs, dialog)
		}

		if user, ok := value.Peer.(*tg.InputPeerUser); ok {
			for _, u := range value.Entities.Users() {
				if u.ID == user.UserID {
					result.Users = append(result.Users, u)
					break
				}
			}
		}
		if peerChat, ok := value.Peer.(*tg.InputPeerChat); ok {
			if chat, ok := value.Entities.Chat(peerChat.ChatID); ok {
				result.Chats = append(result.Chats, chat)
			}
		}
		if channel, ok := value.Peer.(*tg.InputPeerChannel); ok {
			if chat, ok := value.Entities.Channel(channel.ChannelID); ok {
				result.Chats = append(result.Chats, chat)
			}
		}
		result.OffsetDate = value.Last.GetDate()
		result.OffsetID = value.Last.GetID()
	}

	if err := it.Err(); err != nil {
		return nil, err
	}
	return &result, nil
}

func getUserFromClasses(users []tg.UserClass, peerID int64) *types.UserInfo {
	for _, userClass := range users {
		if user, ok := userClass.(*tg.User); ok && user.ID == peerID {
			return shared.ConvertTGUserToUserInfo(user)
		}
	}
	return nil
}

func getChannelFromClasses(chats []tg.ChatClass, peerID int64) *types.ChannelInfo {
	for _, chatClass := range chats {
		if channel, ok := chatClass.(*tg.Channel); ok && channel.ID == peerID {
			return convertToChannelInfo(channel)
		}
		if chat, ok := chatClass.(*tg.Chat); ok && chat.ID == peerID {
			return convertToChannelInfo(chat)
		}
	}
	return nil
}

func getUnreadCount(chatDialogs []*tg.Dialog, peerID int64) int {
	for _, p := range chatDialogs {
		if tgPeerUser, ok := p.Peer.(*tg.PeerUser); ok && tgPeerUser.UserID == peerID {
			return p.UnreadCount
		}
		if tgPeerChannel, ok := p.Peer.(*tg.PeerChannel); ok && tgPeerChannel.ChannelID == peerID {
			return p.UnreadCount
		}
		if tgPeerChat, ok := p.Peer.(*tg.PeerChat); ok && tgPeerChat.ChatID == peerID {
			return p.UnreadCount
		}
	}
	return 0
}

func getNotifySettings(chatDialogs []*tg.Dialog, peerID int64) *tg.PeerNotifySettings {
	for _, p := range chatDialogs {
		if tgPeerUser, ok := p.Peer.(*tg.PeerUser); ok && tgPeerUser.UserID == peerID {
			return &p.NotifySettings
		}
		if tgPeerChannel, ok := p.Peer.(*tg.PeerChannel); ok && tgPeerChannel.ChannelID == peerID {
			return &p.NotifySettings
		}
		if tgPeerChat, ok := p.Peer.(*tg.PeerChat); ok && tgPeerChat.ChatID == peerID {
			return &p.NotifySettings
		}
	}
	return nil
}

func convertToChannelInfo[T *tg.Channel | *tg.Chat](channel T) *types.ChannelInfo {
	switch v := any(channel).(type) {
	case *tg.Channel:
		return &types.ChannelInfo{
			ChannelTitle:      v.Title,
			Username:          &v.Username,
			ID:                strconv.FormatInt(v.ID, 10),
			AccessHash:        strconv.FormatInt(v.AccessHash, 10),
			IsCreator:         v.Creator,
			IsBroadcast:       v.Broadcast,
			ParticipantsCount: &v.ParticipantsCount}
	case *tg.Chat:
		return &types.ChannelInfo{
			ChannelTitle:      v.Title,
			ID:                strconv.FormatInt(v.ID, 10),
			Username:          nil,
			IsCreator:         v.Creator,
			IsBroadcast:       false,
			ParticipantsCount: &v.ParticipantsCount,
		}
	}
	return nil
}
func parseReplyID(replyID string) *int {
	if replyID == "" {
		return nil
	}
	if id, err := strconv.Atoi(replyID); err == nil {
		return &id
	}
	return nil
}
func (c *Client) GetMe(ctx context.Context) (*types.UserInfo, error) {
	userClass, err := c.GetAPI().UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUserSelf{}})
	if err != nil || len(userClass) == 0 {
		return nil, errors.New("failed to get current user info")
	}
	user, ok := userClass[0].(*tg.User)
	if !ok {
		return nil, errors.New("unexpected user type")
	}
	return shared.ConvertTGUserToUserInfo(user), nil
}

func (c *Client) GetSingleMessage(ctx context.Context, peer types.Peer, messageID int) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := shared.ConvertPeerToInputPeer(peer)
		if err != nil {
			return types.SingleMessageMsg{Err: err}
		}

		var id tg.InputMessageClass = &tg.InputMessageID{ID: messageID}
		var messagesClass tg.MessagesMessagesClass

		if peer.ChatType == types.ChannelChat || peer.ChatType == types.GroupChat {
			inputChannel, ok := inputPeer.(*tg.InputPeerChannel)
			if ok {
				messagesClass, err = c.GetAPI().ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
					Channel: &tg.InputChannel{ChannelID: inputChannel.ChannelID, AccessHash: inputChannel.AccessHash},
					ID:      []tg.InputMessageClass{id},
				})
			} else {
				messagesClass, err = c.GetAPI().MessagesGetMessages(ctx, []tg.InputMessageClass{id})
			}
		} else {
			messagesClass, err = c.GetAPI().MessagesGetMessages(ctx, []tg.InputMessageClass{id})
		}

		if err != nil {
			return types.SingleMessageMsg{Err: err}
		}

		entities, err := shared.GetMessageAndUserClasses(messagesClass)
		if err != nil {
			return types.SingleMessageMsg{Err: err}
		}

		if len(entities.Messages) == 0 {
			return types.SingleMessageMsg{Err: errors.New("message not found")}
		}

		msg, ok := entities.Messages[0].(*tg.Message)
		if !ok {
			return types.SingleMessageMsg{Err: errors.New("unexpected message type")}
		}

		var userInfo *types.UserInfo
		if peer.ChatType == types.UserChat || peer.ChatType == types.BotChat {
			if id, err := strconv.ParseInt(peer.ID, 10, 64); err == nil {
				userInfo = getUserFromClasses(entities.Users, id)
			}
		}

		formatted := shared.FormatMessage(msg, userInfo, entities.Messages)
		formatted.PeerID = &peer.ID

		return types.SingleMessageMsg{Message: formatted}
	}
}
