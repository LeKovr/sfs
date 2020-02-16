// Package upload implements image upload handlers.
package cauth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "go.uber.org/zap"
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
	Config     *Config
	Log        *log.SugaredLogger
	ContextKey string
}

// New creates an Service object
func New(cfg Config, logger *log.SugaredLogger, key string) *Service {
	return &Service{&cfg, logger, key}
}

func (srv Service) SetupRouter(r *gin.Engine) {
	r.Use(srv.AuthRequired())
	r.GET("/api/profile", srv.Profile())
}

// AuthRequired is a simple middleware to check the session
func (srv Service) AuthRequired() func(c *gin.Context) {
	return func(c *gin.Context) {

		isNew := false
		tokenString, err := c.Cookie(srv.Config.CookieName)
		if err != nil {
			// no cookie, try header
			tokenString = c.GetHeader(srv.Config.HeaderName)
			if tokenString == "" {
				isNew = true
			}
		}
		srv.Log.Debugw("Got user token", "token", tokenString)

		var token string
		if isNew {
			u, err := uuid.NewRandom()
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
			token = u.String()
			c.SetCookie(srv.Config.CookieName, token, srv.Config.CookieMaxAge, "/", "", false, true)
			srv.Log.Debugw("Set user token", "token", token)
		} else {
			u, err := uuid.Parse(tokenString)
			if err != nil {
				c.AbortWithError(http.StatusUnauthorized, err)
				return
			}
			token = u.String()
		}

		c.Set(srv.ContextKey, token)
		// Continue down the chain to handler etc
		c.Next()
	}
}

func (srv Service) Profile() func(c *gin.Context) {
	return func(c *gin.Context) {
		tokenIface, _ := c.Get(srv.ContextKey)
		if tokenIface == nil {
			c.AbortWithError(http.StatusInternalServerError, ErrNoAuth)
			return
		}
		token := tokenIface.(string)
		srv.Log.Debugw("Got user token", "token", token)
		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}
