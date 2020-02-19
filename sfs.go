// Package upload implements image upload handlers.
package sfs

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "go.uber.org/zap"

	"github.com/LeKovr/sfs/storage"
)

// codebeat:disable[TOO_MANY_IVARS]

// Config holds all config vars
type Config struct {
	FilesFieldName string `long:"field" default:"files[]" description:"Files form field name"`
}

// codebeat:enable[TOO_MANY_IVARS]

var (
	// ErrNoAnyFile returned when request does not contain item in field 'files[]'
	ErrNoAnyFile = errors.New("field 'file' does not contains any item")
	// ErrNoAuth returned on Internal Server Error (no auth for upload)
	ErrNoAuth = errors.New("This endpoint must be under AuthRequired")
)

// Service holds upload service
type Service struct {
	Config     *Config
	Log        *log.SugaredLogger
	ContextKey string
	store      *storage.Service
}

// New creates an Service object
func New(cfg Config, logger *log.SugaredLogger, store *storage.Service, key string) *Service {
	return &Service{&cfg, logger, key, store}
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
	r.GET("/api/files", srv.Files())
	r.GET("/file/:id", srv.File())
}

func (srv Service) HandleMultiPart(c *gin.Context) {
	// TODO: где-то тут вылетает паника при обрыве аплоада клиентом
	tokenIface, _ := c.Get(srv.ContextKey)
	if tokenIface == nil {
		c.AbortWithError(http.StatusInternalServerError, ErrNoAuth)
		return
	}
	token := tokenIface.(string)
	srv.Log.Debugw("Got user token", "token", token)

	form, _ := c.MultipartForm()
	files := form.File[srv.Config.FilesFieldName]
	if len(files) == 0 {
		c.AbortWithError(http.StatusBadRequest, ErrNoAnyFile)
		return
	}
	names := map[string]string{}
	for _, file := range files {
		fileID, err := srv.store.AddFile(token, file)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
			return
		}
		names[file.Filename] = fileID
	}
	c.JSON(http.StatusOK, gin.H{"files": names})
}

// AuthRequired is a simple middleware to check the session
func (srv Service) Files() func(c *gin.Context) {
	return func(c *gin.Context) {
		tokenIface, _ := c.Get(srv.ContextKey)
		if tokenIface == nil {
			c.AbortWithError(http.StatusInternalServerError, ErrNoAuth)
			return
		}
		token := tokenIface.(string)
		srv.Log.Debugw("Got user token", "token", token)
		files, err := srv.store.FileList(token)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, files)
	}
}

// AuthRequired is a simple middleware to check the session
func (srv Service) File() func(c *gin.Context) {
	return func(c *gin.Context) {
		tokenIface, _ := c.Get(srv.ContextKey)
		if tokenIface == nil {
			c.AbortWithError(http.StatusInternalServerError, ErrNoAuth)
			return
		}
		token := tokenIface.(string)
		file, filePath, err := srv.store.File(token, c.Param("id"))
		if err != nil {
			c.AbortWithError(http.StatusNotFound, err)
			return
		}
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", "attachment; filename="+file.Name)
		c.File(filePath)
	}
}
