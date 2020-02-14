package main

import (
	"errors"
	"time"

	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
	"github.com/mattn/go-colorable"
	log "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// ErrGotHelp returned after showing requested help
	ErrGotHelp = errors.New("help printed")
	// ErrBadArgs returned after showing command args error message
	ErrBadArgs = errors.New("option error printed")
)

// setupConfig loads flags from args (if given) or command flags and ENV otherwise
func setupConfig(args ...string) (*Config, error) {
	cfg := &Config{}
	p := flags.NewParser(cfg, flags.Default) //  HelpFlag | PrintErrors | PassDoubleDash
	var err error
	if len(args) == 0 {
		_, err = p.Parse()
	} else {
		_, err = p.ParseArgs(args)
	}
	if err != nil {
		//fmt.Printf("Args error: %#v", err)
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil, ErrGotHelp
		}
		return nil, ErrBadArgs
	}
	return cfg, nil
}

// setupLog creates logger
func setupLog(cfg *Config, r *gin.Engine) *log.SugaredLogger {

	var l *log.Logger

	if gin.IsDebugging() {
		//l, _ = log.NewDevelopment()

		aa := log.NewDevelopmentEncoderConfig()
		aa.EncodeLevel = zapcore.CapitalColorLevelEncoder
		l = log.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(aa),
			zapcore.AddSync(colorable.NewColorableStdout()),
			zapcore.DebugLevel,
		),
			log.AddCaller(),
		)

	} else {
		l, _ = log.NewProduction()
	}

	r.Use(ginzap.Ginzap(l, time.RFC3339, false))

	// Logs all panic to error log
	//   - stack means whether output the stack info.
	r.Use(ginzap.RecoveryWithZap(l, true))

	return l.Sugar()
}
