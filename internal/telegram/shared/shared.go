package shared

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"os/exec"

	"github.com/gotd/td/telegram/downloader"

	"github.com/gotd/td/telegram"
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
		if userOrChannel == nil {
			sender = "unknown"
		} else {
			switch v := any(userOrChannel).(type) {
			case *types.ChannelInfo:
				if v != nil {
					sender = v.ChannelTitle
					FromID = &v.ID
				} else {
					sender = "unknown channel"
				}
			case *types.UserInfo:
				if v != nil {
					sender = v.FirstName
					FromID = &v.PeerID
					SenderUserInfo = v
				} else {
					sender = "unknown user"
				}
			default:
				sender = "unknown"
			}
		}
	}

	var reply *types.FormattedMessage
	if replyTo, ok := msg.GetReplyTo(); ok {
		if messageReply, ok := replyTo.(*tg.MessageReplyHeader); ok && len(allMessages) > 0 {
			replyMessage := getRelyMessage(allMessages, messageReply.ReplyToMsgID)
			reply = FormatMessage(replyMessage, userOrChannel, allMessages)
		}
	}

	isUnsupportedMessage := msg.Media != nil

	var content = msg.Message

	if isUnsupportedMessage {
		content = "This Message is not supported by this Telegram client."
	}

	return &types.FormattedMessage{
		ID:                   msg.ID,
		Sender:               sender,
		Content:              content,
		IsFromMe:             msg.Out,
		Media:                nil,
		Date:                 time.Unix(int64(msg.Date), 0),
		IsUnsupportedMessage: isUnsupportedMessage,
		WebPage:              nil,
		Document:             nil,
		FromID:               FromID,
		SenderUserInfo:       SenderUserInfo,
		ReplyTo:              reply,
	}
}

func getRelyMessage(allMessages []tg.MessageClass, messageID int) *tg.Message {
	var message *tg.Message
	for _, msg := range allMessages {
		msg, ok := msg.(*tg.Message)
		if !ok {
			continue
		}
		if msg.ID == messageID {
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
		return nil, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "users.getUsers failed", err)
	}

	if len(userClasses) == 0 {
		return nil, types.NewUserNotFoundError(userID)
	}

	tgUser, ok := userClasses[0].(*tg.User)
	if !ok {
		return nil, types.NewTelegramError(types.ErrorCodeGetMessagesFailed, "users.getUsers: unexpected type", nil)
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

func DownloadStoryMedia(ctx context.Context, client *telegram.Client, story *tg.StoryItem, outDir string) (*int, error) {
	var peerID string
	peer, ok := story.GetFromID()

	if ok {
		switch p := peer.(type) {
		case *tg.PeerUser:
			peerID = strconv.FormatInt(p.UserID, 10)
		case *tg.PeerChannel:
			peerID = strconv.FormatInt(p.ChannelID, 10)
		case *tg.PeerChat:
			peerID = strconv.FormatInt(p.ChatID, 10)
		default:
			peerID = string(rune(time.Now().Day()))
		}
	}

	switch media := story.Media.(type) {
	case *tg.MessageMediaDocument:
		if doc, ok := media.Document.AsNotEmpty(); ok {
			ext := "bin"
			switch doc.MimeType {
			case "video/mp4":
				ext = "mp4"
			case "image/jpeg":
				ext = "jpg"
			}
			filePath := filepath.Join(outDir, fmt.Sprintf("story_%d,%s.%s", story.ID, peerID, ext))
			return &story.ID, saveMediaToFileSystem(ctx, client.API(), filePath, doc.AsInputDocumentFileLocation())
		}

	case *tg.MessageMediaPhoto:
		if ph, ok := media.Photo.AsNotEmpty(); ok {
			filePath := filepath.Join(outDir, fmt.Sprintf("story_%d,%s.%s", story.ID, peerID, "jpg"))
			if fileInfo, err := os.Stat(filePath); err == nil {
				if fileInfo.Size() > 0 {
					err := OpenFileInDefaultApp(filePath)
					if err != nil {
						return nil, err
					}
					return &story.ID, nil
				}
			}

			photoFileLocation := &tg.InputPhotoFileLocation{
				ID:            ph.ID,
				AccessHash:    ph.AccessHash,
				FileReference: ph.FileReference,
				ThumbSize:     "y",
			}
			return &story.ID, saveMediaToFileSystem(ctx, client.API(), filePath, photoFileLocation)
		}

	default:
		return nil, errors.New("No downloadable media in this story")
	}

	return nil, errors.New("i have no idea for some fucking reason we are not able to get the type of story")
}

func saveMediaToFileSystem(ctx context.Context, client *tg.Client, filePath string, inputFileLocation tg.InputFileLocationClass) error {
	dl := downloader.NewDownloader()
	if fileInfo, err := os.Stat(filePath); err == nil {
		if fileInfo.Size() > 0 {
			err := OpenFileInDefaultApp(filePath)
			if err != nil {
				return err
			}
			return nil
		}
	}

	dBuilder := dl.Download(client, inputFileLocation)
	_, err := dBuilder.ToPath(ctx, filePath)
	if err != nil {
		return err
	}

	err = OpenFileInDefaultApp(filePath)
	if err != nil {
		return err
	}
	return nil
}

func OpenFileInDefaultApp(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		//fuck you windows 🖕 i hate you
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
