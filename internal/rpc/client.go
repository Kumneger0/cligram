package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	JsProcess *os.Process
	RpcClient *JsonRpcClient
)

type JsonRpcClient struct {
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Cmd    *exec.Cmd
	NextID int
	Mu     sync.Mutex
}

type JsonRpcRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type BaseJsonRpcResponse struct {
	JsonRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Result  []interface{} `json:"result,omitempty"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
}

type UserGroupsMsg struct {
	Err error
	//groupds and channels share same propertiy so we don't have to redefine its types use this instead
	Response *UserChannelResponse
}

func (c *JsonRpcClient) GetUserGroups() tea.Cmd {
	userGroupsRpcResponse, err := c.Call("getUserChats", []string{"group"})
	return func() tea.Msg {
		if err != nil {
			return UserGroupsMsg{Err: err}
		}
		var response UserChannelResponse
		if err := json.Unmarshal(userGroupsRpcResponse, &response); err != nil {
			return UserGroupsMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userGroupsRpcResponse), err)}
		}
		if response.Error != nil {
			return UserGroupsMsg{Err: fmt.Errorf("ERROR: %s", response.Error.Message)}
		}
		return UserGroupsMsg{Err: nil, Response: &response}
	}
}

type PeerInfo struct {
	AccessHash string `json:"accessHash"`
	PeerID     string `json:"peerId"`
}

var (
	JsonPayloadBytesChan = make(chan struct {
		Data  []byte
		Error error
	}, 1)
	ProducerWg sync.WaitGroup
)

func (c *JsonRpcClient) Call(method string, params interface{}) ([]byte, error) {
	c.Mu.Lock()
	id := c.NextID
	c.NextID++
	c.Mu.Unlock()

	request := JsonRpcRequest{
		JsonRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	requestPayloadBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var requestBuffer bytes.Buffer
	requestBuffer.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(requestPayloadBytes)))
	requestBuffer.WriteString("\r\n")
	requestBuffer.Write(requestPayloadBytes)
	ProducerWg.Add(1)
	_, err = c.Stdin.Write(requestBuffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to write request to stdin: %w", err)
	}
	jsonPayloadBytes := <-JsonPayloadBytesChan
	ProducerWg.Wait()

	if jsonPayloadBytes.Error != nil {
		return nil, fmt.Errorf("failed to read response: %w", jsonPayloadBytes.Error)
	}

	return jsonPayloadBytes.Data, nil
}

func ReadStdOut(rpcClient *JsonRpcClient) ([]byte, error) {
	reader := bufio.NewReader((rpcClient.Stdout))
	var contentLength int = -1
	for {
		lineBytes, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF && contentLength != -1 {
				return nil, fmt.Errorf("EOF while expecting JSON content of length %d", contentLength)
			}
			return nil, fmt.Errorf("failed to read response header line: %v", err.Error())
		}
		line := string(lineBytes)

		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headerName := strings.TrimSpace(parts[0])
			headerValue := strings.TrimSpace(parts[1])
			if strings.EqualFold(headerName, "Content-Length") {
				cl, err := strconv.Atoi(headerValue)
				if err != nil {
					return nil, fmt.Errorf("invalid Content-Length value '%s': %w", headerValue, err)
				}
				contentLength = cl
			}
		}
	}

	if contentLength == -1 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("Content-Length is 0, no JSON payload expected/read")
	}

	jsonPayloadBytes := make([]byte, contentLength)
	n, err := io.ReadFull(reader, jsonPayloadBytes)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("EOF while reading JSON payload: expected %d bytes, got %d", contentLength, n)
		}
		return nil, fmt.Errorf("failed to read JSON payload (expected %d bytes, read %d): %w", contentLength, n, err)
	}
	if n != contentLength {
		return nil, fmt.Errorf("short read for JSON payload: expected %d bytes, got %d", contentLength, n)
	}
	return jsonPayloadBytes, nil
}

type UserInfo struct {
	FirstName   string  `json:"firstName"`
	IsBot       bool    `json:"isBot"`
	PeerID      string  `json:"peerId"`
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
	Message FormattedMessage
	User    UserInfo
}

type UserOnlineOffline struct {
	AccessHash string    `json:"accessHash,omitempty"`
	FirstName  string    `json:"firstName,omitempty"`
	Status     string    `json:"status,omitempty"`
	LastSeen   time.Time `json:"lastSeen,omitempty"`
}

type RpcTelegramNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type Notification struct {
	NewMessageMsg        NewMessageMsg
	UserOnlineOfflineMsg UserOnlineOffline
}

func ProcessIncomingNotifications(p chan Notification) {

	for {
		time.Sleep(1 * time.Second)
		jsonPayload, err := ReadStdOut(RpcClient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Notifications channel closed: %v\n", err.Error())
			return
		}

		var notification RpcTelegramNotification
		if err := json.Unmarshal(jsonPayload, &notification); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse JSON payload: %v\n", err.Error())
			return
		}

		if notification.Method == "newMessage" || notification.Method == "userOnlineOffline" {
			if notification.Method == "newMessage" {
				paramsBytes, err := json.Marshal(notification.Params)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to marshal params: %v\n", err.Error())
					slog.Error("Failed to marshal params", "error", err.Error())
					return
				}

				var newMessage NewMessageMsg
				if err := json.Unmarshal(paramsBytes, &newMessage); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to unmarshal params: %v\n", err.Error())
					slog.Error("Failed to unmarshal params", "error", err.Error())
					return
				}
				if p != nil {
					p <- Notification{NewMessageMsg: newMessage}
				} else {
					slog.Error("bubble tea event loop is not initialized")
				}
			}
			if notification.Method == "userOnlineOffline" {
				paramsBytes, err := json.Marshal(notification.Params)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to marshal params: %v\n", err.Error())
					slog.Error("Failed to marshal params", "error", err.Error())
					return
				}

				var userOnlineOffline UserOnlineOffline
				if err := json.Unmarshal(paramsBytes, &userOnlineOffline); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to unmarshal params: %v\n", err.Error())
					return
				}

				if p != nil {
					p <- Notification{UserOnlineOfflineMsg: userOnlineOffline}
				}
			}
		} else {
			JsonPayloadBytesChan <- struct {
				Data  []byte
				Error error
			}{
				Data:  jsonPayload,
				Error: err,
			}

			// this is not working when closed figure out and why this doesn't work
			// close(JsonPayloadBytesChan)
			ProducerWg.Done()
		}

	}
}
