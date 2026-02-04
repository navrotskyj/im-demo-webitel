package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	p "github.com/webitel/chat_preview/gen/im/api/gateway/v1"
	"github.com/webitel/wlog"
	"google.golang.org/grpc/metadata"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for this preview
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	Hub        *Hub
	GrpcClient p.MessageClient
	Log        *wlog.Logger
	IMSub      string
}

func NewServer(hub *Hub, grpcClient p.MessageClient, log *wlog.Logger) *Server {
	return &Server{
		Hub:        hub,
		GrpcClient: grpcClient,
		Log:        log,
		IMSub:      "2",
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == "OPTIONS" {
		return
	}

	if r.URL.Path == "/ws" {
		s.serveWs(w, r)
		return
	}
	if r.URL.Path == "/api/messages" && r.Method == http.MethodPost {
		s.handleSendMessage(w, r)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Log.Error("upgrade error", wlog.Err(err))
		return
	}
	client := &Client{hub: s.Hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
}

type SendMessageRequest struct {
	Text string `json:"text"`
	// In a real app we would want user info here, but for now we hardcode or send minimal
	Sub string `json:"sub"` // optional override
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Logic to send via gRPC
	// Replicating main.go logic
	ctx := context.Background()
	header := metadata.New(map[string]string{
		"x-webitel-type":   "schema",
		"x-webitel-schema": "1.2",
	})
	ctx = metadata.NewOutgoingContext(ctx, header)

	sub := "2522" // default from main.go

	if req.Sub != "" {
		sub = req.Sub
	}

	_, err := s.GrpcClient.SendText(ctx, &p.SendTextRequest{
		To: &p.Peer{
			Kind: &p.Peer_Contact{
				Contact: &p.PeerIdentity{
					Sub: sub,
					Iss: "bot",
				},
			},
		},
		Body: req.Text,
	})

	if err != nil {
		s.Log.Error("failed to send text", wlog.Err(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Broadcast injects a message into the hub to be sent to all clients
func (s *Server) Broadcast(msg []byte) {

	s.Hub.broadcast <- msg
}
