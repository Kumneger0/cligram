package rpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
)

type TelegramClient struct {
	*telegram.Client
	ctx context.Context
}

var TGClient *TelegramClient

var once sync.Once

func mustGetenv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		log.Fatalf("%s is missing in env", key)
	}
	return v
}

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0o700)
		}
		return err
	}
	return nil
}

func GetSessionStorage() *telegram.FileSessionStorage {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error(err.Error())
		log.Fatal(err.Error())
		return nil
	}
	sessionStoragePath := filepath.Join(userHomeDir, ".cligram")
	if err := ensureDir(sessionStoragePath); err != nil {
		slog.Error(err.Error())
		log.Fatal(err.Error())
		return nil
	}
	return &telegram.FileSessionStorage{
		Path: filepath.Join(sessionStoragePath, "session.json"),
	}
}

func defaultUserInfoForID(userID int64) *UserInfo {
	pid := strconv.FormatInt(userID, 10)
	return &UserInfo{
		FirstName:   "",
		IsBot:       false,
		PeerID:      pid,
		IsTyping:    false,
		AccessHash:  "",
		UnreadCount: 10,
		IsOnline:    true,
	}
}

func channelAndGroupInfoFromTG(peer *tg.Channel) *ChannelAndGroupInfo {
	return &ChannelAndGroupInfo{
		ChannelTitle:      peer.Title,
		Username:          &peer.Username,
		ChannelID:         strconv.FormatInt(peer.ID, 10),
		AccessHash:        strconv.FormatInt(peer.AccessHash, 10),
		IsCreator:         peer.Creator,
		IsBroadcast:       peer.Broadcast,
		ParticipantsCount: &peer.ParticipantsCount,
		UnreadCount:       0,
	}
}

func getFormattedMessage(msg *tg.Message, user *UserInfo) *FormattedMessage {
	return &FormattedMessage{
		ID:                   int64(msg.ID),
		Sender:               user.FirstName,
		Content:              msg.Message,
		IsFromMe:             msg.Out,
		Media:                nil,
		Date:                 time.Unix(int64(msg.Date), 0),
		IsUnsupportedMessage: false,
		WebPage:              nil,
		Document:             nil,
		FromID:               &user.PeerID,
		SenderUserInfo:       user,
	}
}

func ensureUserInfo(userInfo *UserInfo, userID int64) *UserInfo {
	if userInfo == nil {
		return defaultUserInfoForID(userID)
	}
	return userInfo
}

func userInfoFromTG(tgUser *tg.User, userID int64) *UserInfo {
	if tgUser == nil {
		return defaultUserInfoForID(userID)
	}
	return &UserInfo{
		FirstName:   tgUser.FirstName,
		IsBot:       tgUser.Bot,
		PeerID:      strconv.FormatInt(userID, 10),
		IsTyping:    false,
		AccessHash:  strconv.FormatInt(tgUser.AccessHash, 10),
		UnreadCount: 10,
		LastSeen:    nil,
		IsOnline:    false,
	}
}

func GetTelegramClient(ctx context.Context, updateChannel chan Notification) *TelegramClient {
	once.Do(func() {
		appID := mustGetenv("TELEGRAM_API_ID")
		appIDInt, err := strconv.Atoi(appID)
		if err != nil {
			slog.Error(err.Error())
			log.Fatal("Invalid TELEGRAM_API_ID")
		}
		appHash := mustGetenv("TELEGRAM_API_HASH")

		updateHandler := getUpdateHandler(updateChannel)
		TGClient = &TelegramClient{
			Client: telegram.NewClient(appIDInt, appHash, telegram.Options{
				SessionStorage: GetSessionStorage(),
				UpdateHandler:  updateHandler,
				NoUpdates:      false,
				OnDead: func() {
					fmt.Fprintln(os.Stderr, "connection is dead")
				},
			}),
		}
		TGClient.ctx = ctx
	})
	return TGClient
}

type UserGroupsMsg struct {
	Err error
	//groupds and channels share same propertiy so we don't have to redefine its types use this instead
	Response *UserChannelResponse
}

func (c *TelegramClient) GetUserGroups() tea.Cmd {
	return func() tea.Msg {
		dilaogsSlice, err := getAllDialogs(c)
		if err != nil {
			return UserChatsMsg{Err: err}
		}
		var channel []ChannelAndGroupInfo
		for _, dialogClass := range dilaogsSlice.Chats {
			if peer, ok := dialogClass.(*tg.Channel); ok && !peer.Broadcast {
				channel = append(channel, *channelAndGroupInfoFromTG(peer))
			}
		}
		return UserGroupsMsg{Err: nil, Response: &UserChannelResponse{
			Error:  nil,
			Result: channel,
		}}
	}
}

func getUserOnlineOffline(userInfo UserInfo) UserOnlineOffline {
	return UserOnlineOffline{
		AccessHash: userInfo.AccessHash,
		FirstName:  userInfo.FirstName,
		Status:     "",
		PeerID:     userInfo.PeerID,
	}
}

func getUpdateHandler(updateChannel chan Notification) telegram.UpdateHandler {
	disp := tg.NewUpdateDispatcher()
	disp.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		msg := update.Message.(*tg.Message)
		if peerClass, ok := msg.GetFromID(); ok {
			fmt.Println("channel", peerClass)
			return nil
		}
		return nil
	})

	disp.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		msg := update.Message.(*tg.Message)
		if peerClass, ok := msg.GetFromID(); ok && TGClient != nil {
			peer, ok := peerClass.(*tg.PeerUser)
			if !ok {
				return nil
			}
			userInfo, err := TGClient.getUserInfo(peer.UserID)
			if err != nil || userInfo == nil {
				if err != nil {
					slog.Error(err.Error())
				}
			}
			user := ensureUserInfo(userInfo, peer.UserID)
			var newMessageMSG = NewMessageMsg{
				ChannelOrGroup: nil,
				User:           user,
				Message:        *getFormattedMessage(msg, user),
			}
			updateChannel <- Notification{
				NewMessageMsg:        newMessageMSG,
				UserOnlineOfflineMsg: UserOnlineOffline{},
				UserTyping:           UserTyping{},
				RPCError:             Error{},
			}
		}
		return nil
	})

	disp.OnUserTyping(func(ctx context.Context, e tg.Entities, update *tg.UpdateUserTyping) error {
		userInfo, err := TGClient.getUserInfo(update.UserID)
		if err != nil || userInfo == nil {
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		user := ensureUserInfo(userInfo, update.UserID)
		updateChannel <- Notification{
			NewMessageMsg:        NewMessageMsg{},
			UserOnlineOfflineMsg: UserOnlineOffline{},
			UserTyping: UserTyping{
				User: *user,
			},
		}
		return nil
	})

	disp.OnUserStatus(func(ctx context.Context, e tg.Entities, update *tg.UpdateUserStatus) error {
		userInfo, err := TGClient.getUserInfo(update.UserID)
		if err != nil || userInfo == nil {
			return nil
		}
		if status, ok := update.Status.(*tg.UserStatusOffline); ok {
			lastSeenTime := time.Unix(int64(status.WasOnline), 0)
			userOnlineOfflineMSG := getUserOnlineOffline(*ensureUserInfo(userInfo, update.UserID))
			userOnlineOfflineMSG.LastSeen = lastSeenTime
			updateChannel <- Notification{
				NewMessageMsg:        NewMessageMsg{},
				UserOnlineOfflineMsg: userOnlineOfflineMSG,
				UserTyping:           UserTyping{},
				RPCError: Error{
					Error: nil,
				},
			}
		}

		if _, ok := update.Status.(*tg.UserStatusOnline); ok {
			now := time.Now()
			userOnlineOfflineMSG := getUserOnlineOffline(*ensureUserInfo(userInfo, update.UserID))
			userOnlineOfflineMSG.LastSeen = now
			updateChannel <- Notification{
				NewMessageMsg:        NewMessageMsg{},
				UserOnlineOfflineMsg: userOnlineOfflineMSG,
				UserTyping:           UserTyping{},
				RPCError: Error{
					Error: nil,
				},
			}
		}
		return nil
	})

	return updates.New(updates.Config{
		Handler: disp,
	})
}

func (c *TelegramClient) getUserInfo(userID int64) (*UserInfo, error) {
	inputUserClass := &tg.InputUser{
		UserID: userID,
	}
	userClasses, err := c.Client.API().UsersGetUsers(c.ctx, []tg.InputUserClass{inputUserClass})
	if err != nil {
		return nil, err
	}
	if len(userClasses) == 0 {
		return nil, errors.New("no users found with the provided user id and accessHash")
	}
	if tgUser, ok := userClasses[0].(*tg.User); ok {
		var user = UserInfo{
			FirstName:   tgUser.FirstName,
			IsBot:       tgUser.Bot,
			PeerID:      strconv.FormatInt(userID, 10),
			IsTyping:    false,
			AccessHash:  strconv.FormatInt(tgUser.AccessHash, 10),
			UnreadCount: 10,
			LastSeen:    nil,
			IsOnline:    false,
		}
		return &user, nil
	}
	return nil, nil
}

type PeerInfo struct {
	AccessHash string `json:"accessHash"`
	PeerID     string `json:"peerId"`
	ChatType   ChatType
}
type UserInfo struct {
	FirstName   string  `json:"firstName"`
	IsBot       bool    `json:"isBot"`
	PeerID      string  `json:"peerId"`
	IsTyping    bool    `json:"isTyping"`
	AccessHash  string  `json:"accessHash"`
	UnreadCount int     `json:"unreadCount"`
	LastSeen    *string `json:"lastSeen"`
	IsOnline    bool    `json:"isOnline"`
}

type ChannelAndGroupInfo struct {
	ChannelTitle      string  `json:"title"`
	Username          *string `json:"username"`
	ChannelID         string  `json:"channelId"`
	AccessHash        string  `json:"accessHash"`
	IsCreator         bool    `json:"isCreator"`
	IsBroadcast       bool    `json:"isBroadcast"`
	ParticipantsCount *int    `json:"participantsCount"`
	UnreadCount       int     `json:"unreadCount"`
}

func (u UserInfo) Title() string {
	return u.FirstName
}

func (u UserInfo) FilterValue() string {
	return u.FirstName
}

func (c ChannelAndGroupInfo) FilterValue() string {
	return c.ChannelTitle
}

func (c ChannelAndGroupInfo) Title() string {
	return c.ChannelTitle
}

type NewMessageMsg struct {
	Message        FormattedMessage
	User           *UserInfo
	ChannelOrGroup *ChannelAndGroupInfo
}

type UserOnlineOffline struct {
	AccessHash string    `json:"accessHash,omitempty"`
	FirstName  string    `json:"firstName,omitempty"`
	Status     string    `json:"status,omitempty"`
	LastSeen   time.Time `json:"lastSeen,omitempty"`
	PeerID     string    `json:"peerId"`
}

type TelegramNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type UserTyping struct {
	User UserInfo
}

type Error struct {
	Error error
}

type Notification struct {
	NewMessageMsg        NewMessageMsg
	UserOnlineOfflineMsg UserOnlineOffline
	UserTyping           UserTyping
	RPCError             Error
}
