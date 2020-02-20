package widget

import (
	"fmt"
	"time"

	"github.com/LeKovr/sfs/pubsub"
)

type WidgetEvent struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Data string `json:"data"`
}

func (srv Service) HandlersRun() {

	stream, err := srv.pubsub.Subscribe("widget")
	if err != nil {
		srv.Log.Errorw("Failed to subscribe", "error", err)
		return
	}
	defer stream.Unsubscribe()

	srv.Log.Debugw("Subscribed on widget")
	for {
		select {
		case msg, ok := <-stream.Messages:
			if !ok {
				// Channel closed,exit
				srv.Log.Debugw("Subscription closed")
				return
			}
			srv.Log.Debugw("Received event", "data", string(msg.Payload))

			var req PageEvent
			pubsub.Unmarshal(msg.Payload, &req)
			// Layout depended list
			go srv.widgetPrepare(req, "top")
			go srv.widgetPrepare(req, "navy")
			go srv.widgetPrepare(req, "menu")
			// Process
		case <-srv.quit:
			break
		}
	}
}

func (srv Service) HandlersClose() {
	close(srv.quit)
}

func (srv Service) widgetPrepare(req PageEvent, name string) {

	var demoSleep = map[string]int{
		"menu": 2,
		"navy": 1,
	}
	s, ok := demoSleep[name]
	if ok {
		time.Sleep(time.Duration(s) * time.Second) // эмуляция процесса
	}
	if err := srv.pubsub.Publish("once.widget."+req.RequestID,
		WidgetEvent{"widget", name, fmt.Sprintf("<h2>rendered %s</h2>", name)}); err != nil {
		srv.Log.Errorw("Widget event publish error", "name", name, "error", err)
		return
	}
}
