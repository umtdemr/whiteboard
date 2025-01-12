package main

import (
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/umtdemr/wb-backend/internal/ws"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (app *application) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Error().Err(err)
		return
	}

	client := ws.CreateNewClient(app.wsHub, conn)
	go client.WritePump()
	go client.ReadPump()
}
