// Package upload implements image upload handlers.
package sfs

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "go.uber.org/zap"
)

// codebeat:disable[TOO_MANY_IVARS]

// Config holds all config vars
type Config struct {
	DataPath       string `long:"data" default:"data" description:"Path to served files"`
	FilesFieldName string `long:"field" default:"files[]" description:"Files form field name"`
}

// codebeat:enable[TOO_MANY_IVARS]

var (

	// ErrNoAnyFile returned when request does not contain item in field 'files[]'
	ErrNoAnyFile = errors.New("field 'file' does not contains single item")

	ErrNoAuth = errors.New("This endpoint must be under AuthRequired")
)

// Service holds upload service
type Service struct {
	Config     *Config
	Log        *log.SugaredLogger
	ContextKey string
}

// New creates an Service object
func New(cfg Config, logger *log.SugaredLogger, key string) *Service {
	return &Service{&cfg, logger, key}
}

func (srv Service) SetupRouter(r *gin.Engine) {
	r.POST("/upload", func(c *gin.Context) {
		switch c.ContentType() {
		case "multipart/form-data":
			srv.HandleMultiPart(c)
		default:
			c.String(http.StatusNotImplemented, "Content type (%s) not supported", c.ContentType())
		}
	})
}

var i int

func (srv Service) HandleMultiPart(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File[srv.Config.FilesFieldName]
	names := map[int]string{}

	tokenIface, _ := c.Get(srv.ContextKey)
	if tokenIface == nil {
		c.AbortWithError(http.StatusInternalServerError, ErrNoAuth)
		return
	}
	token := tokenIface.(string)
	srv.Log.Debugw("Got user token", "token", token)
	for _, file := range files {
		srv.Log.Debugw("Store file", "id", i, "name", file.Filename, "size", file.Size)
		// TODO: для каждого файла генерим uuid
		dst := fmt.Sprintf("data/%d.data", i)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
			return
		}
		names[i] = file.Filename
		i++
	}
	c.JSON(http.StatusOK, gin.H{"files": names})
}
