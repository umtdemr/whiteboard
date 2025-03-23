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
	"github.com/umtdemr/wb-backend/internal/validator"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 5 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	joinWait = 10 * time.Second
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

	joined chan struct{}
}

func CreateNewClient(hub *Hub, conn *websocket.Conn) *Client {
	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		joined: make(chan struct{}),
	}

	go func() {
		// wait joinWait seconds to join. If they do not join in, close the connection
		timer := time.NewTimer(joinWait)
		defer timer.Stop()

		for {
			select {
			case <-client.joined:
				timer.Stop()
				return
			case <-timer.C:
				client.conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "join timeout"),
					time.Now().Add(5*time.Second),
				)
				client.conn.Close()
			}
		}
	}()

	return client
}

// incomingMessageReq represents the data that ws clients should send
type incomingMessageReq struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Id   string          `json:"id"`
}

type messageResponse struct {
	ReplyTo string   `json:"reply_to,omitempty"`
	Event   string   `json:"event,omitempty"`
	Data    envelope `json:"data"`
}

func (c *Client) ReadPump() {
	defer func() {
		c.broadCastMessage(messageResponse{
			Event: EventUserLeft,
			Data:  envelope{"user": c.user},
		})

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
			c.sendErrorResponse("ERR_UNKNOWN_MESSAGE_TYPE", ErrUnknownMessageType.toResponse())
			continue
		}

		messageReader := bytes.NewReader(wsMessage)
		zlibReader, err := zlib.NewReader(messageReader)
		if err != nil {
			c.sendErrorResponse("ERR_UNKNOWN_COMPRESSION_METHOD", ErrUnknownCompressionMethod.toResponse())
			continue
		}

		closeErr := zlibReader.Close()
		if closeErr != nil {
			continue
		}

		var message incomingMessageReq

		if err := jsonHelper.ReadJson(zlibReader, &message); err != nil {
			c.sendErrorResponse("ERR_JSON_DECODE", ErrorResponse{
				Code:    ErrCodeJsonDecoding,
				Message: err.Error(),
			})
			continue
		}

		if closeErr := zlibReader.Close(); err != nil {
			log.Error().Err(closeErr).Msg("Error on closing z lib")
			continue
		}

		if message.Id == "" {
			c.sendErrorResponse("ERR_ID_MUST_SET", ErrIdMustSet.toResponse())
			continue
		}

		// if the user has not joined, do not handle the incoming message
		if message.Type != CmdJoin {
			if c.boardId == "" {
				c.sendErrorResponse(message.Id, ErrAuth.toResponse())
				continue
			}

			if _, exists := c.hub.boards[c.boardId]; !exists {
				c.sendErrorResponse(message.Id, ErrAuth.toResponse())
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
	compressed, err := compressData(message)

	if err != nil {
		return
	}

	c.send <- compressed
}

// broadCastMessage sends given message to other users in the board
func (c *Client) broadCastMessage(message messageResponse) {
	compressed, err := compressData(message)

	if err != nil {
		return
	}

	c.hub.broadcastToBoard(c.boardId, compressed, c.user.ID)
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
	c.sendErrorResponse(replyTo, ErrAuth.toResponse())
}

// handleMessage handles incoming message with incomingMessageReq
func (c *Client) handleMessage(message *incomingMessageReq) error {
	handler, err := c.decodeMessageData(message.Type, message.Data)
	if err != nil {
		fieldError := FieldError{}
		switch {
		case errors.Is(err, ErrCmdNotFound):
			c.sendErrorResponse(message.Id, ErrCmdNotFound.toResponse())
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

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			writeDeadlineErr := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if writeDeadlineErr != nil {
				log.Error().Err(writeDeadlineErr).Msg("Error on setting write wait")
			}
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				log.Error().Err(err).Msg("Error on writing message")
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
