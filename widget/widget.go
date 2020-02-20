// Package upload implements image upload handlers.
package widget

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "go.uber.org/zap"

	"github.com/LeKovr/sfs/pubsub"
)

// codebeat:disable[TOO_MANY_IVARS]

// Config holds all config vars
type Config struct {
	CookieName   string `long:"cookie" default:"sfs_auth" description:"Auth cookie name"`
	HeaderName   string `long:"header" default:"X-SFS-Auth" description:"Auth header name"`
	CookieMaxAge int    `long:"cookie_ttl" default:"36000" description:"Auth cookie TTL"`
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

// Service holds upload service
type Service struct {
	Config              *Config
	Log                 *log.SugaredLogger
	ContextKey          string
	ContextRequestIDKey string
	pubsub              *pubsub.Service
	quit                chan struct{}
}

// New creates an Service object
func New(cfg Config, logger *log.SugaredLogger, ps *pubsub.Service, key, idKey string) *Service {
	return &Service{
		Config:              &cfg,
		Log:                 logger,
		ContextKey:          key,
		ContextRequestIDKey: idKey,
		pubsub:              ps,
		quit:                make(chan struct{}),
	}
}

func (srv Service) SetupRouter(r *gin.Engine) {
	r.Use(srv.RequestID())
	r.GET("/api/widget.js", srv.WidgetJS())
}

// AuthRequired is a simple middleware to check the session
func (srv Service) RequestID() func(c *gin.Context) {
	return func(c *gin.Context) {

		u, err := uuid.NewRandom()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		requestID := u.String()
		c.Set(srv.ContextRequestIDKey, requestID)
		srv.Log.Debugw("RequestID", "id", requestID)

		// Continue down the chain to handler etc
		c.Next()
	}
}

type PageEvent struct {
	RequestID string
	Layout    string
}

func (srv Service) WidgetJS() func(c *gin.Context) {
	return func(c *gin.Context) {
		reqIface, _ := c.Get(srv.ContextRequestIDKey)
		if reqIface == nil {
			c.AbortWithError(http.StatusInternalServerError, ErrNoAuth)
			return
		}
		reqID := reqIface.(string)
		layout := c.Query("layout")
		if layout == "" {
			srv.Log.Errorw("layout arg is missing")
			return
		}

		err := srv.pubsub.Publish("widget", PageEvent{reqID, layout})
		if err != nil {
			srv.Log.Errorw("Widget event publish error", "error", err)
			return
		}

		srv.Log.Debugw("widget.js ready")
		c.Header("Content-Type", "application/javascript")
		c.String(http.StatusOK, "var RequestID='%s';\n", reqID)
	}
}
