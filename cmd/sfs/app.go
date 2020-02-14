package main

import (
	"log"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/LeKovr/sfs"
	"github.com/LeKovr/sfs/cauth"
)

// Config holds all config vars
type Config struct {
	Listen      string       `long:"listen" default:":8080" description:"Addr and port which server listens at"`
	AssetPath   string       `long:"html" default:"html" description:"Path to static html files"`
	MemoryLimit int64        `long:"mem_max" default:"8"  description:"Memory limit for multipart forms, Mb"`
	CAuth       cauth.Config `group:"Auth Options" namespace:"auth"`
	SFS         sfs.Config   `group:"FileServer Options" namespace:"fs"`
}

const (
	// ErrNoSingleFile returned when does not contain single file in field 'file'
	ContextAuthKey = "SFS-CAuth"
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

	authService := cauth.New(cfg.CAuth, l.Named("cauth"), ContextAuthKey)
	sfsService := sfs.New(cfg.SFS, l.Named("sfs"), ContextAuthKey)

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	router.MaxMultipartMemory = cfg.MemoryLimit << 20

	router.Static("/static", filepath.Join(cfg.AssetPath, "static"))
	router.StaticFile("/favicon.ico", filepath.Join(cfg.AssetPath, "favicon.ico"))
	router.StaticFile("/", filepath.Join(cfg.AssetPath, "index.html"))

	authService.SetupRouter(router)
	sfsService.SetupRouter(router)

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
