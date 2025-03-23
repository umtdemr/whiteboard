package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebsocketHandler(t *testing.T) {
	app := &application{}

	ts := httptest.NewServer(http.HandlerFunc(app.websocketHandler))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("could not open ws connection: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)); err != nil {
		t.Errorf("connection not alive: %v", err)
	}
}
