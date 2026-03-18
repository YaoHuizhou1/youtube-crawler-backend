package handlers

import (
	"net/http"

	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/websocket"
	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"github.com/google/uuid"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type WebSocketHandler struct {
	hub *websocket.Hub
}

func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

func (h *WebSocketHandler) Handle(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("WebSocket upgrade error: %v", err)
		return
	}

	clientID := uuid.New().String()
	client := websocket.NewClient(clientID, conn, h.hub)
	h.hub.Register(client)

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()

	logger.Infof("WebSocket client connected: %s", clientID)
}
