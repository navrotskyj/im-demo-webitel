package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net/http"
	"os"

	p "github.com/webitel/chat_preview/gen/im/api/gateway/v1"
	"github.com/webitel/chat_preview/infra/pubsub"
	"github.com/webitel/chat_preview/infra/wbt"
	"github.com/webitel/chat_preview/internal/server"
	"github.com/webitel/wlog"
)

const ServiceName = "im-gateway-service"

// MessageWrapper представляє кореневий об'єкт
type MessageWrapper struct {
	ID       string  `json:"ID"`
	Message  Message `json:"message"`
	UserID   string  `json:"user_id"`
	DomainID int64   `json:"domain_id"`
}

// Message описує вкладений об'єкт повідомлення
type Message struct {
	ID        string          `json:"ID"`
	ThreadID  string          `json:"thread_id"`
	DomainID  int             `json:"domain_id"`
	From      ImEndpoint      `json:"from"`
	To        ImEndpoint      `json:"to"`
	Text      string          `json:"text"`
	CreatedAt int64           `json:"created_at"` // Unix timestamp у мілісекундах
	Me        bool            `json:"me"`
	Images    json.RawMessage `json:"images"`
}

// From описує відправника
type ImEndpoint struct {
	ID     string `json:"id"`
	Type   int    `json:"type"`
	Sub    string `json:"sub"`
	Issuer string `json:"issuer"`
	Name   string `json:"name"`
}

type TLSConfig struct {
	CertPath string
	KeyPath  string
	CAPath   string
}

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	consul := getEnv("CONSUL", "localhost:8500")

	tlsConfig := TLSConfig{
		CertPath: getEnv("SERVICE_CONN_CLIENT_CERT", "flow-manager-client.pem"),
		KeyPath:  getEnv("SERVICE_CONN_CLIENT_KEY", "flow-manager-client-key.pem"),
		CAPath:   getEnv("SERVICE_CONN_CLIENT_CA", "ca.pem"),
	}

	t, _ := LoadTlsCreds(tlsConfig)
	var opts []wbt.Option
	opts = append(opts, wbt.WithGrpcOptions(
		grpc.WithTransportCredentials(credentials.NewTLS(t)),
	))

	cli, err := wbt.NewClient(consul, ServiceName, p.NewMessageClient, opts...)
	if err != nil {
		panic(err)
	}

	// Initialize Server and Hub
	hub := server.NewHub()
	go hub.Run()

	imSub := getEnv("IM_SUB", "2522")
	uiDir := getEnv("UI_DIR", "./web/dist")

	srv := server.NewServer(hub, cli.Api, wlog.GlobalLogger(), imSub, uiDir)

	// Start RabbitMQ consumer
	amqpAddr := getEnv("AMQP", "amqp://user:pass@localhost:5672")
	startps(srv, amqpAddr)

	port := getEnv("PORT", ":8080")

	// Start HTTP Server
	go func() {
		wlog.Info(fmt.Sprintf("Listening on port %s", port))

		if err := http.ListenAndServe(port, srv); err != nil {
			wlog.Error("HTTP server error", wlog.Err(err))
		}
	}()

	// Keep main alive
	select {}
}

func startps(srv *server.Server, addr string) {
	ps, err := pubsub.New(wlog.GlobalLogger(), addr)
	if err != nil {
		panic(err.Error())
	}
	ps.AddOnConnect(func(channel *pubsub.Channel) error {
		qname := fmt.Sprintf("chat_preview_%s", NewId()[:6])
		if err := channel.DeclareDurableQueue(qname, pubsub.Headers{
			"x-queue-type": "quorum",
			"x-expires":    10000,
		}); err != nil {
			return err
		}

		if err := channel.BindQueue(qname, "#", "im_delivery.broadcast", pubsub.Headers{}); err != nil {
			return err
		}

		delivery, err := channel.ConsumeQueue(qname, false)
		if err != nil {
			return err
		}
		go func() {
			for {
				select {
				case msg, ok := <-delivery:
					if !ok {
						return
					}

					println(string(msg.Body))

					var m MessageWrapper
					if err := json.Unmarshal(msg.Body, &m); err == nil {
						// Log locally
						println("Received: " + m.Message.Text)
					}

					if m.Message.Images != nil {
						println("images not empty")
					}
					m.Message.Me = m.Message.From.Sub == "2"
					js, err := json.Marshal(m)
					if err != nil {
						panic(err.Error())
					}

					// Broadcast to WebSockets
					srv.Broadcast(js)

					// Log raw body just in case
					// println(string(msg.Body))
					msg.Ack(true)

				}
			}
		}()
		return nil
	})
	ps.Start()

}

func LoadTlsCreds(cfg TLSConfig) (*tls.Config, error) {

	if len(cfg.CertPath) == 0 || len(cfg.KeyPath) == 0 || len(cfg.CAPath) == 0 {
		return nil, nil
	}

	// Load client's certificate and private key
	clientCert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}

	// Load the CA certificate to verify server
	caCert, err := os.ReadFile(cfg.CAPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Configure TLS
	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
		ServerName:   "im-gateway-service", // Common Name of the server cert
	}, nil
}

var encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769")

func NewId() string {
	var b bytes.Buffer
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	b.Truncate(26) // removes the '==' padding
	return b.String()
}
