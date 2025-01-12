package ws

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/umtdemr/wb-backend/internal/data"
	"github.com/umtdemr/wb-backend/internal/jsonHelper"
	"github.com/umtdemr/wb-backend/internal/token"
	"github.com/umtdemr/wb-backend/internal/validator"
	"io"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	boardId string

	user *data.User

	cursor *Cursor
}

func (c *Client) ReadPump() {
	defer func() {
		otherMembers := c.hub.boards[c.boardId]
		for client, _ := range otherMembers {
			if client.user.ID == c.user.ID {
				continue
			}

			// let other users know about this user left
			client.sendCompressedData(messageResponse{
				Event: StcUserLeft,
				Data:  envelope{"user": c.user},
			})
		}
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		msgType, wsMessage, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Msgf("websocket unexpected error: %v", err)
			}
			break
		}

		// accept only binary messages
		if msgType != websocket.BinaryMessage {
			continue
		}

		//c.hub.broadcast <- BroadcastMsg{client: c, message: wsMessage}
		messageReader := bytes.NewReader(wsMessage)
		zlibReader, err := zlib.NewReader(messageReader)
		if err != nil {
			continue
		}

		byteMessage, msgReadErr := io.ReadAll(zlibReader)
		closeErr := zlibReader.Close()
		if closeErr != nil {
			continue
		}

		if msgReadErr != nil {
			continue
		}

		var message incomingMessageReq

		if err := json.Unmarshal(byteMessage, &message); err != nil {
			log.Error().Msgf("error on unmarshalling: %v", err)
			continue
		}

		if message.Id == "" {
			c.sendErrorResponse("ERR_ID_MUST_SET", &ErrorResponse{
				Code:    ErrCodeIdMustSet,
				Message: ErrIdMustSet.Error(),
			})
			continue
		}

		// if the user has not joined, do not handle the incoming message
		if message.Type != CmdJoin {
			if c.boardId == "" {
				continue
			}

			if _, exists := c.hub.boards[c.boardId]; !exists {
				continue
			}
		}

		err = c.handleMessage(&message)

		if err != nil {
			continue
		}
	}
}

// sendCompressedData compresses given message and sends to the client
func (c *Client) sendCompressedData(message messageResponse) {
	decoded, err := json.Marshal(message)

	if err != nil {
		return
	}

	b := bytes.NewBuffer(nil)
	msgWriter := zlib.NewWriter(b)
	_, err = msgWriter.Write(decoded)
	if err != nil {
		return
	}
	msgWriter.Close()

	c.send <- b.Bytes()
}

func (c *Client) sendErrorResponse(replyTo string, message any) {
	resp := messageResponse{
		ReplyTo: replyTo,
		Data:    envelope{"error": message},
	}

	c.sendCompressedData(resp)
}

// sendErrorAuthResponse sends unauthorized error message to client
func (c *Client) sendErrorAuthResponse(replyTo string) {
	c.sendErrorResponse(replyTo, ErrorResponse{Code: ErrCodeAuth, Message: ErrAuth.Error()})
}

// handleMessage handles incoming message with incomingMessageReq
func (c *Client) handleMessage(message *incomingMessageReq) error {
	handler, err := c.decodeMessageData(message.Type, message.Data)
	if err != nil {
		fieldError := FieldError{}
		switch {
		case errors.Is(err, ErrCmdNotFound):
			c.sendErrorResponse(message.Id, ErrorResponse{
				Code:    ErrCodeCmdNotFound,
				Message: ErrCmdNotFound.Error(),
			})
			return nil
		case errors.As(err, &fieldError):
			c.sendErrorResponse(message.Id, FieldErrorResponse{
				ErrorResponse: ErrorResponse{
					Code:    ErrCodeField,
					Message: fieldError.Error(),
				},
				Fields: fieldError.errors,
			})
		default:
			c.sendErrorResponse(message.Id, ErrorResponse{
				Code:    ErrCodeUnknown,
				Message: err.Error(),
			})
		}
		return err
	}

	return handler.Handle(message.Id, c)
}

// decodeMessageData decodes body data of incomingMessageReq and returns Message interface to handle the message
func (c *Client) decodeMessageData(messageType string, data json.RawMessage) (MessageHandler, error) {
	switch messageType {
	case CmdJoin:
		var req joinMessage
		err := jsonHelper.ReadJson(bytes.NewReader(data), &req)
		if err != nil {
			return nil, err
		}

		v := validator.New()
		req.Validate(v)

		if !v.Valid() {
			return nil, FieldError{errors: v.Errors}
		}

		return &req, nil
	case CmdCursor:
		var cursorMsg cursorMessage
		err := jsonHelper.ReadJson(bytes.NewReader(data), &cursorMsg)
		if err != nil {
			return nil, err
		}

		return &cursorMsg, nil
	}

	return nil, ErrCmdNotFound
}

// MessageHandler is an interface for different incoming request
type MessageHandler interface {
	Handle(replyTo string, client *Client) error
}

// joinMessage is a request type for the board join requests
type joinMessage struct {
	BoardSlugId   string `json:"board_slug_id"`
	UserAuthToken string `json:"user_auth_token"`
}

type joinResponseUser struct {
	User   *data.User `json:"user"`
	Cursor *Cursor    `json:"cursor"`
}

type joinResponse struct {
	OnlineUsers []*joinResponseUser `json:"online_users"`
}

func (m *joinMessage) Handle(replyTo string, client *Client) error {
	user, err := client.hub.models.User.GetForToken(token.ScopeAuthentication, m.UserAuthToken)
	if err != nil {
		client.sendErrorAuthResponse(replyTo)
		return nil
	}

	// set user for the client
	client.user = user
	client.boardId = m.BoardSlugId

	// register the user
	if _, exists := client.hub.boards[client.boardId]; !exists {
		client.hub.boards[client.boardId] = make(map[*Client]bool)
	}
	client.hub.boards[client.boardId][client] = true

	otherUsers := client.hub.boards[client.boardId]
	usersMap := make(map[int32]bool) // to avoid duplicated reports
	usersList := make([]*joinResponseUser, 0, len(client.hub.boards[client.boardId]))

	for client := range otherUsers {
		if client.user.ID == user.ID {
			continue
		}

		// let other users know about new online user
		client.sendCompressedData(messageResponse{
			Event: StcUserJoined,
			Data:  envelope{"user": user},
		})

		// if exists, ignore
		if _, ok := usersMap[client.user.ID]; ok {
			continue
		}

		usersMap[client.user.ID] = true
		usersList = append(usersList, &joinResponseUser{User: client.user, Cursor: client.cursor})
	}
	resp := messageResponse{
		ReplyTo: replyTo,
		Data:    envelope{"join": joinResponse{OnlineUsers: usersList}},
	}

	client.sendCompressedData(resp)
	return nil
}

func (m *joinMessage) Validate(v *validator.Validator) {
	v.Check(len(m.BoardSlugId) == 12, "board_slug_id", "must be 12 bytes long")
	v.Check(len(m.UserAuthToken) > 5, "user_auth_token", "required")
}

// cursorMessage is a request type to handle collaborator cursors
type cursorMessage struct {
	X float64 `json:"x"`
	Y float64 `json:"Y"`
}

func (m *cursorMessage) Handle(replyTo string, client *Client) error {
	if client.cursor == nil {
		client.cursor = &Cursor{}
	}
	// save cursor
	client.cursor.X = m.X
	client.cursor.Y = m.Y

	// send new cursor data to other users in same board
	allClients := client.hub.GetAllClientsInBoard(client.boardId)
	for _, otherClient := range allClients {
		if otherClient.user.ID == client.user.ID {
			continue
		}

		otherClient.sendCompressedData(messageResponse{
			Event: StcCursor,
			Data: envelope{"cursor": CursorWithUser{
				UserName: client.user.FullName,
				UserId:   int64(client.user.ID),
				Cursor:   client.cursor,
			}},
		})
	}

	return nil
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.WriteMessage(websocket.BinaryMessage, message)
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func CreateNewClient(hub *Hub, conn *websocket.Conn) *Client {
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}

	return client
}
