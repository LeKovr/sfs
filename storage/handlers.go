package storage

func (srv Service) HandlersRun() {

	stream, err := srv.pubsub.Subscribe("file")
	if err != nil {
		srv.Log.Errorw("Failed to subscribe", "error", err)
		return
	}
	defer stream.Unsubscribe()

	srv.Log.Debugw("Subscribed on file")
	for {
		select {
		case msg, ok := <-stream.Messages:
			if !ok {
				// Channel closed,exit
				srv.Log.Debugw("Subscription closed")
				return
			}
			srv.Log.Debugw("Received event", "data", string(msg.Payload))

			// Process
		case <-srv.quit:
			break
		}
	}
}

func (srv Service) HandlersClose() {
	close(srv.quit)
}
