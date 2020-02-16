// Package upload implements image upload handlers.
package stream

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "go.uber.org/zap"

	"github.com/LeKovr/sfs/pubsub"
)

// codebeat:disable[TOO_MANY_IVARS]

// Config holds all config vars
type Config struct {
	CookieName   string `long:"cookie" default:"sfs_auth" description:"Auth cookie name"`
	HeaderName   string `long:"header" default:"X-SFS-Auth" description:"Auth header name"`
	CookieMaxAge int    `long:"cookie_ttl" default:"3600" description:"Auth cookie TTL"`
}

// codebeat:enable[TOO_MANY_IVARS]

const (
	// ErrNoSingleFile returned when does not contain single file in field 'file'
	ErrNoSingleFile = "field 'file' does not contains single item"
)
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var (
	// ErrNoAuth returned on Internal Server Error (no auth for upload)
	ErrNoAuth = errors.New("This endpoint must be under AuthRequired")
)

// Service holds upload service
type Service struct {
	Config     *Config
	Log        *log.SugaredLogger
	ContextKey string
	pubsub     *pubsub.Service
}

// New creates an Service object
func New(cfg Config, logger *log.SugaredLogger, ps *pubsub.Service, key string) *Service {
	return &Service{&cfg, logger, key, ps}
}

func (srv Service) SetupRouter(r *gin.Engine) {
	r.GET("/ws/:key", func(c *gin.Context) {
		token := c.Param("key")
		if token == "" {
			c.AbortWithError(http.StatusUnauthorized, ErrNoAuth)
			return
		}
		srv.Log.Debugw("Got user token", "token", token)

		srv.wshandler(c.Writer, c.Request, token)
	})
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (srv Service) wshandler(w http.ResponseWriter, r *http.Request, token string) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		srv.Log.Errorw("Failed to set websocket upgrade", "error", err)
		return
	}
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	stream, err := srv.pubsub.Subscribe("user." + token)
	if err != nil {
		srv.Log.Errorw("Failed to subscribe", "error", err)
		return
	}
	defer stream.Unsubscribe()

	srv.Log.Debugw("Subscribed on", "prefix", "user", "topic", token)
	for {
		select {
		case msg, ok := <-stream.Messages:
			if !ok {
				// Channel closed,exit
				srv.Log.Debugw("Subscription closed")
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			srv.Log.Debugw("Received event", "data", string(msg.Payload))
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			err = conn.WriteMessage(websocket.TextMessage, msg.Payload)
			if err != nil {
				srv.Log.Warnw("Failed to send ws message", "error", err)
				return
			}
		case <-ticker.C:
			//srv.Log.Debugw("WS timeout")
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				srv.Log.Debugw("WS ping failed")
				return
			}
		}
	}
}
