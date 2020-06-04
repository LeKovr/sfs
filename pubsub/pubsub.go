// Package pubsub contains a simple pub/sub 
package pubsub

import (
	"encoding/json"
	"errors"
	"strings"
	//	"fmt"

	log "go.uber.org/zap"
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

var (
	// ErrNoAnyFile returned when request does not contain item in field 'files[]'
	ErrNoAnyFile = errors.New("field 'file' does not contains any item")
	// ErrNoAuth returned on Internal Server Error (no auth for upload)
	ErrNoAuth = errors.New("This endpoint must be under AuthRequired")
)

type Message struct {
	Topic   string
	Payload []byte
}
type MessageStream chan Message

type Stream struct {
	Topic      string
	Messages   chan Message
	unregister chan chan Message
}

// Service holds the set of active connections and broadcasts messages
type Service struct {
	Config *Config
	Log    *log.SugaredLogger
	// Published messages
	broadcast chan Message
	// Register requests from the connections.
	register chan *Stream
	// Unregister requests from connections.
	unregister chan chan Message
	// Quit channel
	quit chan struct{}
}

// New creates an Service object
func New(cfg Config, logger *log.SugaredLogger) *Service {
	srv := Service{
		Config:     &cfg,
		Log:        logger,
		broadcast:  make(chan Message),
		register:   make(chan *Stream),
		unregister: make(chan chan Message),
	}
	return &srv
}

func (srv Service) Close() {
	if srv.quit != nil {
		srv.quit <- struct{}{}
	}
}

func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (s Stream) Unsubscribe() {
	s.unregister <- s.Messages
}

func (srv Service) Subscribe(topic string) (Stream, error) {
	s := Stream{
		Topic:      topic,
		Messages:   make(chan Message),
		unregister: srv.unregister,
	}
	srv.register <- &s
	return s, nil
}

func (srv Service) Publish(topic string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	srv.Log.Debugw("Send event", "topic", topic, "data", string(b))
	srv.broadcast <- Message{topic, b}
	return nil
}

func (srv *Service) Run() {
	subscribers := map[string]map[chan Message]bool{}
	clients := map[chan Message]string{}
	buffers := map[string][]Message{}
	srv.Log.Debugw("Hub opened")
	srv.quit = make(chan struct{})
	for {
		select {
		case c := <-srv.register:
			// Register new client in hub
			topicSubscribers, ok := subscribers[c.Topic]
			srv.Log.Debugw("Subscribe", "topic", c.Topic)
			if !ok {
				topicSubscribers = map[chan Message]bool{c.Messages: true}
			} else {
				topicSubscribers[c.Messages] = true
			}
			subscribers[c.Topic] = topicSubscribers
			clients[c.Messages] = c.Topic
			if buffer, ok := buffers[c.Topic]; ok {
				srv.Log.Debugw("Push waiting events", "count", len(buffer))
				for _, m := range buffer {
					srv.Log.Debugw("Hub buffer", "payload", string(m.Payload))
					c.Messages <- m
				}
				delete(buffers, c.Topic)
			}
		case c := <-srv.unregister:
			// Unregister client from hub
			if topic, ok := clients[c]; ok {
				srv.Log.Debugw("Unsubscribe", "topic", topic)
				delete(subscribers[topic], c)
				delete(clients, c)
				close(c)
			}
		case m := <-srv.broadcast:
			// Loop over all clients in hub.
			srv.Log.Debugw("Hub message", "topic", m.Topic, "payload", string(m.Payload))
			topicSubscribers, ok := subscribers[m.Topic]
			if !ok {
				if !strings.HasPrefix(m.Topic, "once.") {
					srv.Log.Warnw("No subscribers for topic", "topic", m.Topic)
					continue
				}
				// save in buffer
				buffer, ok := buffers[m.Topic]
				if !ok {
					buffer = []Message{m}
				} else {
					buffer = append(buffer, m)
				}
				buffers[m.Topic] = buffer
			} else {
				srv.Log.Debugw("Hub message point",
					"topic", m.Topic,
					"subscribers", len(topicSubscribers),
					"payload", string(m.Payload))
				for k := range topicSubscribers {
					k <- m
				}
			}
		case <-srv.quit:
			for k := range clients {
				close(k)
			}
			srv.Log.Debugw("Hub closed")
			return
		}
	}
}
