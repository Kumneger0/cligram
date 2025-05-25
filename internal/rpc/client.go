package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

type UserChatsMsg struct {
	Response *UserChatsJsonRpcResponse
	Err      error
}

type DuplicatedLastSeen struct {
	Type   string
	Time   *time.Time
	Status *string
}

type DuplicatedUserInfo struct {
	FirstName   string             `json:"firstName"`
	IsBot       bool               `json:"isBot"`
	PeerID      string             `json:"peerId"`
	AccessHash  string             `json:"accessHash"`
	UnreadCount int                `json:"unreadCount"`
	LastSeen    DuplicatedLastSeen `json:"lastSeen"`
	IsOnline    bool               `json:"isOnline"`
}

// there is a lot of room here for refacoring
// the the first one is i redefined userInfo here b/c of import cycle
// figure out and eliminate
func (du DuplicatedUserInfo) Title() string {
	return du.FirstName
}

func (du DuplicatedUserInfo) FilterValue() string {
	return du.FirstName
}

type UserChatsJsonRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []DuplicatedUserInfo `json:"result,omitempty"`
}

func (c *JsonRpcClient) GetUserChats() tea.Cmd {
	userChatRpcResponse, err := c.Call("getUserChats", []string{"user"})

	return func() tea.Msg {
		if err != nil {
			return UserChatsMsg{Err: err}
		}
		var response UserChatsJsonRpcResponse
		if err := json.Unmarshal(userChatRpcResponse, &response); err != nil {
			return UserChatsMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChatRpcResponse), err)}
		}
		if response.Error != nil {
			return UserChatsMsg{Err: fmt.Errorf(response.Error.Message)}
		}
		return UserChatsMsg{Err: nil, Response: &response}
	}
}

type DuplicatedChannelAndGroupInfo struct {
	ChannelTitle      string  `json:"title"`
	Username          *string `json:"username"`
	ChannelID         string  `json:"channelId"`
	AccessHash        string  `json:"accessHash"`
	IsCreator         bool    `json:"isCreator"`
	IsBroadcast       bool    `json:"isBroadcast"`
	ParticipantsCount *int    `json:"participantsCount"`
	UnreadCount       int     `json:"unreadCount"`
}

func (dc DuplicatedChannelAndGroupInfo) Title() string {
	return dc.ChannelTitle
}

func (dc DuplicatedChannelAndGroupInfo) FilterValue() string {
	return dc.ChannelTitle
}

type UserChannelResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result []DuplicatedChannelAndGroupInfo `json:"result,omitempty"`
}

type UserChannelMsg struct {
	Err      error
	Response *UserChannelResponse
}

func (c *JsonRpcClient) GetUserChannel() tea.Cmd {
	userChannelRpcResponse, err := c.Call("getUserChats", []string{"channel"})
	return func() tea.Msg {
		if err != nil {
			return UserChannelMsg{Err: err}
		}
		var response UserChannelResponse
		if err := json.Unmarshal(userChannelRpcResponse, &response); err != nil {
			return UserChannelMsg{Err: fmt.Errorf("failed to unmarshal response JSON '%s': %w", string(userChannelRpcResponse), err)}
		}
		if response.Error != nil {
			return UserChannelMsg{Err: fmt.Errorf(response.Error.Message)}
		}
		return UserChannelMsg{Err: nil, Response: &response}
	}
}

type UserGroupsMsg struct {
	Err      error
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
			return UserGroupsMsg{Err: fmt.Errorf(response.Error.Message)}
		}
		return UserGroupsMsg{Err: nil, Response: &response}
	}
}

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

	_, err = c.Stdin.Write(requestBuffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to write request to stdin: %w", err)
	}

	reader := bufio.NewReader(c.Stdout)
	var contentLength int = -1

	for {
		lineBytes, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF && contentLength != -1 {
				return nil, fmt.Errorf("EOF while expecting JSON content of length %d", contentLength)
			}
			return nil, fmt.Errorf("failed to read response header line: %w", err)
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

// this needs to be removed this is redifinded
// ideally we should use the LastSeen struct it has UnmarshalJSON function
// i just redefined b/c of import cyle issue
// will remove this after refactoring the the code base
// this was just quick fix
// TODO:remove this function it is duplicated

func (ls *DuplicatedLastSeen) UnmarshalJSON(data []byte) error {
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
