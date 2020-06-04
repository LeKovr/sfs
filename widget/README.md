# Widget processing sample

* widget.go - /api/widget.js request handler. Fetch RequestID and raise "widget" event
* handlers.go - "widget" event handler. Generates widgets and publish them via "once.widget."+req.RequestID
