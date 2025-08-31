package shared

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/tg"
	"github.com/kumneger0/cligram/internal/telegram/types"
)

type ChannelOrUser interface {
	types.UserInfo | types.ChannelInfo
}

func FormatMessage[T ChannelOrUser](msg *tg.Message, userOrChannel *T, allMessages []tg.MessageClass) *types.FormattedMessage {
	if msg == nil {
		slog.Warn("Format message is becoming nil for some reason")
		return nil
	}
	var sender string
	var FromID *string
	var SenderUserInfo *types.UserInfo
	if msg.Out {
		sender = "you"
	} else {
		switch v := any(userOrChannel).(type) {
		case *types.ChannelInfo:
			sender = v.ChannelTitle
			FromID = &v.ID
		case *types.UserInfo:
			sender = v.FirstName
			FromID = &v.PeerID
			SenderUserInfo = v
		}
	}

	var reply *types.FormattedMessage
	if replyTo, ok := msg.GetReplyTo(); ok {
		if messageReply, ok := replyTo.(*tg.MessageReplyHeader); ok && len(allMessages) > 0 {
			replyMessage := getRelyMessage(allMessages, messageReply.ReplyToMsgID)
			reply = FormatMessage(replyMessage, userOrChannel, allMessages)
		}
	}

	return &types.FormattedMessage{
		ID:                   msg.ID,
		Sender:               sender,
		Content:              msg.Message,
		IsFromMe:             msg.Out,
		Media:                nil,
		Date:                 time.Unix(int64(msg.Date), 0),
		IsUnsupportedMessage: false,
		WebPage:              nil,
		Document:             nil,
		FromID:               FromID,
		SenderUserInfo:       SenderUserInfo,
		ReplyTo:              reply,
	}
}

func getRelyMessage(allMessages []tg.MessageClass, messageId int) *tg.Message {
	var message *tg.Message
	for _, msg := range allMessages {
		msg, ok := msg.(*tg.Message)
		if !ok {
			continue
		}
		if msg.ID == messageId {
			message = msg
			break
		}
	}
	return message
}

func GetUserInfo(ctx context.Context, client tg.Client, userID int64) (*types.UserInfo, error) {
	inputUser := &tg.InputUser{
		UserID: userID,
	}

	userClasses, err := client.UsersGetUsers(ctx, []tg.InputUserClass{inputUser})
	if err != nil {
		return nil, types.NewUserNotFoundError(userID)
	}

	if len(userClasses) == 0 {
		return nil, types.NewUserNotFoundError(userID)
	}

	tgUser, ok := userClasses[0].(*tg.User)
	if !ok {
		return nil, types.NewUserNotFoundError(userID)
	}

	return ConvertTGUserToUserInfo(tgUser), nil
}

func ConvertTGUserToUserInfo(tgUser *tg.User) *types.UserInfo {
	userInfo := &types.UserInfo{
		FirstName:  tgUser.FirstName,
		LastName:   tgUser.LastName,
		Username:   tgUser.Username,
		IsBot:      tgUser.Bot,
		PeerID:     strconv.FormatInt(tgUser.ID, 10),
		AccessHash: strconv.FormatInt(tgUser.AccessHash, 10),
		IsTyping:   false,
		IsOnline:   false,
	}

	if status := getUserOnlineStatus(tgUser.Status); status != nil {
		userInfo.IsOnline = status.IsOnline
		userInfo.LastSeen = status.LastSeen
	}

	return userInfo
}

func getUserOnlineStatus(status tg.UserStatusClass) *userOnlineStatus {
	if status == nil {
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: nil,
		}
	}

	switch s := status.(type) {
	case *tg.UserStatusOnline:
		lastSeen := "online"
		return &userOnlineStatus{
			IsOnline: true,
			LastSeen: &lastSeen,
		}
	case *tg.UserStatusLastMonth:
		lastSeen := "last seen within a month"
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	case *tg.UserStatusRecently:
		lastSeen := "last seen recently"
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	case *tg.UserStatusOffline:
		lastSeen := calculateLastSeenHumanReadable(s.WasOnline)
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	default:
		lastSeen := "last seen long time ago"
		return &userOnlineStatus{
			IsOnline: false,
			LastSeen: &lastSeen,
		}
	}
}

type userOnlineStatus struct {
	IsOnline bool
	LastSeen *string
}

func calculateLastSeenHumanReadable(wasOnline int) string {
	var s strings.Builder
	lastSeenTime := time.Unix(int64(wasOnline), 0)
	currentTime := time.Now()
	diff := currentTime.Sub(lastSeenTime)
	lastSeenFormattedOnlyTime := lastSeenTime.Format("03:04 PM")

	if diff.Seconds() < float64(60) {
		return "last seen just now"
	}
	if diff.Hours() < float64(24) {
		s.WriteString("last seen at ")
		s.WriteString(lastSeenFormattedOnlyTime)
		return s.String()
	}
	if diff.Hours() < float64(48) {
		s.WriteString("last seen yesterday at ")
		s.WriteString(lastSeenFormattedOnlyTime)
		return s.String()
	}

	lastSeenFormattedWithDate := lastSeenTime.Format("02/01/2006 03:04 PM")
	s.WriteString("last seen on ")
	s.WriteString(lastSeenFormattedWithDate)
	return s.String()
}

func GetMessageAndUserClasses(history tg.MessagesMessagesClass) ([]tg.MessageClass, []tg.UserClass, error) {
	var msgs []tg.MessageClass
	var users []tg.UserClass
	switch h := history.(type) {
	case *tg.MessagesMessagesSlice:
		msgs = h.Messages
		users = h.Users
	case *tg.MessagesChannelMessages:
		msgs = h.Messages
		users = h.Users
	case *tg.MessagesMessages:
		msgs = h.Messages
		users = h.Users
	default:
		return nil, nil, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, fmt.Sprintf("unsupported history type: %T", history), nil)
	}
	return msgs, users, nil
}
