package client

import (
	"context"
	"errors"
	"log/slog"
	mathRand "math/rand"
	"os"
	"path/filepath"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"

	"github.com/kumneger0/cligram/internal/telegram/chat"
	"github.com/kumneger0/cligram/internal/telegram/message"
	"github.com/kumneger0/cligram/internal/telegram/shared"
	"github.com/kumneger0/cligram/internal/telegram/types"
	"github.com/kumneger0/cligram/internal/telegram/user"
)

type Client struct {
	*telegram.Client
	ctx            context.Context
	updateChannel  chan types.Notification
	sessionManager types.SessionManager
	userManager    types.UserManager
	messageManager types.MessageSender
	chatManager    types.ChatManager
}

type Config struct {
	AppID         int
	AppHash       string
	UpdateChannel chan types.Notification
}

var Cligram *telegram.Client

func NewClient(ctx context.Context, config Config) (*Client, error) {
	sessionManager, err := NewSessionManager()
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeSessionFailed, "failed to create session manager", err)
	}

	updateHandler := newUpdateHandler(config.UpdateChannel)
	sessionStorage := sessionManager.(*SessionManager).GetTelegramFileSessionStorage()

	Cligram = telegram.NewClient(config.AppID, config.AppHash, telegram.Options{
		SessionStorage: sessionStorage,
		UpdateHandler:  updateHandler,
		NoUpdates:      false,
		OnDead: func() {
			slog.Error("telegram connection is dead")
		},
	})

	client := &Client{
		Client:         Cligram,
		ctx:            ctx,
		updateChannel:  config.UpdateChannel,
		sessionManager: sessionManager,
	}

	client.userManager = user.NewManager(client)
	client.messageManager = message.NewSender(client)
	client.chatManager = chat.NewManager(client)

	return client, nil
}

func mustGetenv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		slog.Error("missing required environment variable", "key", key)
		panic("missing required environment variable: " + key)
	}
	return v
}

func NewClientFromEnv(ctx context.Context, updateChannel chan types.Notification) (*Client, error) {
	appIDStr := mustGetenv("TELEGRAM_API_ID")
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		return nil, types.NewTelegramError(types.ErrorCodeAuthFailed, "invalid TELEGRAM_API_ID", err)
	}

	appHash := mustGetenv("TELEGRAM_API_HASH")

	config := Config{
		AppID:         appID,
		AppHash:       appHash,
		UpdateChannel: updateChannel,
	}

	return NewClient(ctx, config)
}

func (c *Client) GetAPI() *tg.Client {
	return c.Client.API()
}

func (c *Client) Context() context.Context {
	return c.ctx
}

func (c *Client) GetUserManager() types.UserManager {
	return c.userManager
}

func (c *Client) GetMessageManager() types.MessageSender {
	return c.messageManager
}

func (c *Client) GetChatManager() types.ChatManager {
	return c.chatManager
}

func (c *Client) SendMessage(ctx context.Context, req types.SendMessageRequest) tea.Cmd {
	if req.IsFile {
		return c.messageManager.SendMedia(ctx, req.Peer, req.FilePath, req.Message, parseReplyID(req.ReplyToMessageID))
	}
	return c.messageManager.SendText(ctx, req.Peer, req.Message, parseReplyID(req.ReplyToMessageID))
}

func (c *Client) GetMessages(ctx context.Context, req types.GetMessagesRequest) tea.Cmd {
	return c.chatManager.GetChatHistoryCmd(ctx, req.Peer, req.Limit, req.OffsetID)
}

func (c *Client) GetUserChats(ctx context.Context, chatType types.ChatType, offsetDate, offsetID int) tea.Cmd {
	return c.chatManager.GetUserChatsCmd(ctx, chatType == types.BotChat, offsetDate, offsetID)
}

func (c *Client) GetUserChannels(ctx context.Context, isBroadCast bool, offsetDate, offsetID int) tea.Cmd {
	return c.chatManager.GetChannelsCmd(ctx, isBroadCast, offsetDate, offsetID)
}

func (c *Client) GetAllMessages(ctx context.Context, req types.GetMessagesRequest) tea.Cmd {
	return c.chatManager.GetChatHistoryCmd(ctx, req.Peer, req.Limit, req.OffsetID)
}

func (c *Client) DeleteMessage(ctx context.Context, req types.DeleteMessageRequest) (types.DeleteMessageResponse, error) {
	deleteMessageRequest := &tg.MessagesDeleteMessagesRequest{
		Revoke: true,
		ID:     []int{req.MessageID},
	}
	_, err := c.Client.API().MessagesDeleteMessages(ctx, deleteMessageRequest)
	if err != nil {
		return types.DeleteMessageResponse{Status: "failed"}, types.NewDeleteMessageError(err)
	}
	return types.DeleteMessageResponse{Status: "success"}, nil
}

func (c *Client) EditMessage(ctx context.Context, req types.EditMessageRequest) types.EditMessageMsg {
	inputPeer, err := convertPeerToInputPeer(req.Peer)
	if err != nil {
		return types.EditMessageMsg{
			Response: false,
			Err:      types.NewEditMessageError(err),
		}
	}

	editRequest := &tg.MessagesEditMessageRequest{
		Peer:    inputPeer,
		ID:      req.MessageID,
		Message: req.NewMessage,
	}

	_, err = c.Client.API().MessagesEditMessage(ctx, editRequest)
	if err != nil {
		return types.EditMessageMsg{
			Response: false,
			Err:      types.NewEditMessageError(err),
		}
	}
	return types.EditMessageMsg{
		Response: true,
		Err:      nil,
	}
}

func (c *Client) ForwardMessages(ctx context.Context, req types.ForwardMessagesRequest) error {
	fromPeer, err := convertPeerToInputPeer(req.FromPeer)
	if err != nil {
		return types.NewForwardMessageError(err)
	}

	toPeer, err := convertPeerToInputPeer(req.ToPeer)
	if err != nil {
		return types.NewForwardMessageError(err)
	}

	forwardRequest := &tg.MessagesForwardMessagesRequest{
		FromPeer: fromPeer,
		ToPeer:   toPeer,
		ID:       req.MessageIDs,
		RandomID: []int64{mathRand.Int63()},
	}

	_, err = c.Client.API().MessagesForwardMessages(ctx, forwardRequest)
	if err != nil {
		return types.NewForwardMessageError(err)
	}
	return nil
}

func (c *Client) MarkMessagesAsRead(ctx context.Context, req types.MarkAsReadRequest) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := convertPeerToInputPeer(req.Peer)
		if err != nil {
			return types.MarkMessagesAsReadMsg{
				Response: false,
				Err:      err,
			}
		}

		readRequest := &tg.MessagesReadHistoryRequest{
			Peer: inputPeer,
		}

		_, err = c.Client.API().MessagesReadHistory(ctx, readRequest)
		if err != nil {
			return types.MarkMessagesAsReadMsg{
				Response: false,
				Err:      err,
			}
		}

		return types.MarkMessagesAsReadMsg{
			Response: true,
			Err:      nil,
		}
	}
}

func (c *Client) SetUserTyping(ctx context.Context, req types.SetTypingRequest) error {
	inputPeer, err := convertPeerToInputPeer(req.Peer)
	if err != nil {
		return err
	}

	typingRequest := &tg.MessagesSetTypingRequest{
		Peer:   inputPeer,
		Action: &tg.SendMessageTypingAction{},
	}

	_, err = c.Client.API().MessagesSetTyping(ctx, typingRequest)
	if err != nil {
		return types.NewTelegramError(types.ErrorCodeSendFailed, "failed to set typing", err)
	}
	return nil
}

func (c *Client) GetAllStories(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		allUserStoriesClass, err := c.Client.API().StoriesGetAllStories(ctx, &tg.StoriesGetAllStoriesRequest{})

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

			inputUser := tg.InputUser{UserID: peerUser.UserID}

			userClass, err := c.Client.API().UsersGetUsers(ctx, []tg.InputUserClass{&inputUser})
			if err != nil {
				continue
			}

			if len(userClass) == 0 {
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

					AllStories = append(AllStories, types.Stories{
						UserInfo:   *userInfo,
						ID:         storyItem.ID,
						Data:       document.FileReference,
						IsSelected: false,
					})
				case *tg.MessageMediaPhoto:
					photoClass, ok := item.GetPhoto()
					if !ok {
						continue
					}
					photo, ok := photoClass.(*tg.Photo)
					if !ok {
						continue
					}

					AllStories = append(AllStories, types.Stories{
						UserInfo:   *userInfo,
						ID:         storyItem.ID,
						Data:       photo.FileReference,
						IsSelected: false,
					})
				}
			}
		}
		return types.GetAllStoriesMsg{
			Stories: AllStories,
			Err:     nil,
		}
	}
}
func (c *Client) SearchUsers(ctx context.Context, query string) {
	users, err := c.userManager.SearchUsers(ctx, query)
	c.updateChannel <- types.Notification{
		SearchResult: &types.SearchUsersMsg{
			Response: &users,
			Err:      err,
		},
	}
}

func (c *Client) GetPeerStories(ctx context.Context, peer types.Peer) tea.Cmd {
	return func() tea.Msg {
		inputPeer, err := convertPeerToInputPeer(peer)
		if err != nil {
			return types.StoriesDownloadStatusMsg{
				ID:   -1,
				Done: false,
				Err:  err,
			}
		}
		peerUserStories, err := c.Client.API().StoriesGetPeerStories(ctx, inputPeer)
		if err != nil {
			return types.StoriesDownloadStatusMsg{
				ID:   -1,
				Done: false,
				Err:  err,
			}
		}
		homeDir, _ := os.UserHomeDir()
		cligramDir := filepath.Join(homeDir, ".cligram")
		for _, v := range peerUserStories.Stories.Stories {
			if storyItem, ok := v.(*tg.StoryItem); ok {
				id, err := shared.DownloadStoryMedia(ctx, c.Client, storyItem, cligramDir)
				if err != nil {
					return types.StoriesDownloadStatusMsg{
						ID:   -1,
						Done: false,
						Err:  err,
					}
				}
				return types.StoriesDownloadStatusMsg{
					ID:   *id,
					Done: true,
					Err:  nil,
				}
			}
		}
		return types.StoriesDownloadStatusMsg{
			ID:   -1,
			Done: false,
			Err:  errors.New("we have failed to download the story for some fucking reason"),
		}
	}
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

func convertPeerToInputPeer(peer types.Peer) (tg.InputPeerClass, error) {
	peerID, err := strconv.ParseInt(peer.ID, 10, 64)
	if err != nil {
		return nil, types.NewInvalidPeerError(peer.ID)
	}

	accessHash, err := strconv.ParseInt(peer.AccessHash, 10, 64)
	if err != nil {
		return nil, types.NewInvalidPeerError(peer.AccessHash)
	}

	switch peer.ChatType {
	case types.UserChat, types.BotChat:
		return &tg.InputPeerUser{
			UserID:     peerID,
			AccessHash: accessHash,
		}, nil
	case types.ChannelChat:
		return &tg.InputPeerChannel{
			ChannelID:  peerID,
			AccessHash: accessHash,
		}, nil
	case types.GroupChat:
		return &tg.InputPeerChat{
			ChatID: peerID,
		}, nil
	default:
		return nil, types.NewTelegramError(types.ErrorCodeInvalidPeer, "unsupported chat type", nil)
	}
}
