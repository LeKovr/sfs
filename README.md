# sfs. Sample file server

This project shows golang + valilla js example for

* miltifile upload with async processing
* publish processing results via websockets
* build page with delayed widget generation

Project status: PoC is ready

## Project components

### cauth

Cookie Auth

* generate user token and save cookie
* read token from cookie

### pubsub

Simple golang pub/sub (in memory)

### sfs.go

File upload handlers

* /upload
* /api/files
* /file/:id

### storage

Simple file storage powered by [badger](https://github.com/dgraph-io/badger)

### stream

Stream data to client via websocket

Handler for `/ws/:RequestID/:Token/` request. Subscribe client on messages:
* `user.:Token`
* `once.widget.:RequestID`

### widget

Widget processing sample

* widget.go - /api/widget.js request handler. Fetch RequestID and raise "widget" event
* handlers.go - "widget" event handler. Generates widgets and publish them via "once.widget."+req.RequestID

Widget ideas based on [post from Yandex](https://habr.com/ru/company/yandex/blog/486146/) 
