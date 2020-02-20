package main

import (
	"log"
	"path/filepath"

	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"

	"github.com/LeKovr/sfs"
	"github.com/LeKovr/sfs/cauth"
	"github.com/LeKovr/sfs/pubsub"
	"github.com/LeKovr/sfs/storage"
	"github.com/LeKovr/sfs/stream"
	"github.com/LeKovr/sfs/widget"
)

// Config holds all config vars
type Config struct {
	Listen      string         `long:"listen" default:":8080" description:"Addr and port which server listens at"`
	AssetPath   string         `long:"html" default:"html" description:"Path to static html files"`
	MemoryLimit int64          `long:"mem_max" default:"8"  description:"Memory limit for multipart forms, Mb"`
	CAuth       cauth.Config   `group:"Auth Options" namespace:"auth"`
	SFS         sfs.Config     `group:"FileServer Options" namespace:"fs"`
	Store       storage.Config `group:"Storage Options" namespace:"store"`
	Stream      stream.Config  `group:"WS stream Options" namespace:"ws"`
	PubSub      pubsub.Config  `group:"PuSub Options" namespace:"ps"`
	Widget      widget.Config  `group:"Widget Options" namespace:"wg"`
}

const (
	// ErrNoSingleFile returned when does not contain single file in field 'file'
	ContextAuthKey    = "SFS-CAuth"
	ContextRequestKey = "SFS-Request-ID"
)

func run(exitFunc func(code int)) {
	var err error
	var cfg *Config
	defer func() { shutdown(exitFunc, err) }()
	cfg, err = setupConfig()
	if err != nil {
		return
	}

	router := gin.New()
	l := setupLog(cfg, router)
	defer l.Sync()

	pubsubService := pubsub.New(cfg.PubSub, l)
	defer pubsubService.Close()
	streamService := stream.New(cfg.Stream, l, pubsubService, ContextAuthKey)

	var store *storage.Service
	store, err = storage.New(cfg.Store, l, pubsubService)
	if err != nil {
		return
	}
	defer store.Close()

	widgetService := widget.New(cfg.Widget, l, pubsubService, ContextAuthKey, ContextRequestKey)

	go pubsubService.Run()
	go store.HandlersRun()
	defer store.HandlersClose()
	go widgetService.HandlersRun()
	defer widgetService.HandlersClose()

	authService := cauth.New(cfg.CAuth, l, ContextAuthKey)
	sfsService := sfs.New(cfg.SFS, l, store, ContextAuthKey)

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	router.MaxMultipartMemory = cfg.MemoryLimit << 20

	router.Static("/static", filepath.Join(cfg.AssetPath, "static"))
	router.StaticFile("/favicon.ico", filepath.Join(cfg.AssetPath, "favicon.ico"))
	router.StaticFile("/", filepath.Join(cfg.AssetPath, "index.html"))
	router.GET("/debug/vars", expvar.Handler())

	streamService.SetupRouter(router)
	authService.SetupRouter(router)
	sfsService.SetupRouter(router)
	widgetService.SetupRouter(router)

	router.Run(cfg.Listen)

}

// exit after deferred cleanups have run
func shutdown(exitFunc func(code int), e error) {
	if e != nil {
		var code int
		switch e {
		case ErrGotHelp:
			code = 3
		case ErrBadArgs:
			code = 2
		default:
			code = 1
			log.Printf("Run error: %+v", e)
		}
		exitFunc(code)
	}
}
