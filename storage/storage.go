// Package storage implements disk file operations.
package storage

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	log "go.uber.org/zap"

	"github.com/LeKovr/sfs/pubsub"
)

// codebeat:disable[TOO_MANY_IVARS]

// Config holds all config vars
type Config struct {
	DataPath  string `long:"data" default:"var/data" description:"Path to served files"`
	CachePath string `long:"cache" default:"var/cache" description:"Path to cache files"`
}

// codebeat:enable[TOO_MANY_IVARS]

type File struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CType     string    `json:"type"`
	Token     string    `json:"token"`
	State     string    `json:"state"`
	SHA1      string    `json:"sha1"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	// ErrNoSingleFile returned when does not contain single file in field 'file'
	ErrNoSingleFile = "field 'file' does not contains single item"
)

var (
	seqFileID = []byte("fileID")
)

// Service holds upload service
type Service struct {
	Config *Config
	Log    *log.SugaredLogger
	db     *badger.DB
	ticker *time.Ticker
	quit   chan struct{}
	quitGC chan struct{}
	pubsub *pubsub.Service
}

// New creates an Service object
func New(cfg Config, logger *log.SugaredLogger, ps *pubsub.Service) (*Service, error) {
	db, err := badger.Open(badger.DefaultOptions(cfg.CachePath))
	if err != nil {
		return nil, err
	}

	srv := &Service{
		Config: &cfg,
		Log:    logger,
		db:     db,
		ticker: time.NewTicker(5 * time.Minute),
		quit:   make(chan struct{}),
		quitGC: make(chan struct{}),
		pubsub: ps,
	}
	go srv.gc()
	return srv, nil
}

func (srv Service) Close() {
	close(srv.quitGC)
	srv.ticker.Stop()
	srv.db.Close()
}

func (srv Service) gc() {
	for {
		select {
		case <-srv.ticker.C:
			srv.Log.Debug("GC run")
			err := srv.db.RunValueLogGC(0.7)
			if err != nil && err != badger.ErrNoRewrite { // TODO && !db.close - "error": "Value log GC attempt didn't result in any cleanup"
				srv.Log.Warnw("GC error", "error", err)
				return
			}
		case <-srv.quitGC:
			srv.ticker.Stop()
			return
		}
	}
}

func (srv Service) filePath(id string) string {
	return filepath.Join(srv.Config.DataPath, id[0:3], id[3:6]) // TODO: sharding
}

func (srv Service) AddFile(token string, file *multipart.FileHeader) (key string, err error) {
	seq, err := srv.db.GetSequence(seqFileID, 1)
	if err != nil {
		return "", err
	}
	defer seq.Release()
	num, err := seq.Next()
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("%07d", num)
	srv.Log.Debugw("Store file", "id", id, "name", file.Filename, "size", file.Size, "ctype", file.Header["Content-Type"])
	dstDir := srv.filePath(id)
	os.MkdirAll(dstDir, os.ModePerm)
	dst := filepath.Join(dstDir, id+".data")

	if err != nil {
		return "", err
	}
	var ctype string
	if len(file.Header["Content-Type"]) > 0 {
		ctype = file.Header["Content-Type"][0]
	}
	f := File{
		ID:        id,
		Name:      file.Filename,
		Size:      file.Size,
		CType:     ctype,
		Token:     token,
		State:     "received",
		CreatedAt: time.Now(),
	}
	err = srv.db.Update(func(txn *badger.Txn) error {
		err := setFileMeta(txn, &f)
		if err == nil {
			err = txn.Set([]byte("user."+token+"."+id), []byte("1"))
		}
		return err
	})
	go func() {
		err := srv.saveUploadedFile(file, dst, token, id)
		if err != nil {
			srv.Log.Errorw("File save error", "token", token, "file", id, "error", err)
			err = srv.pubsub.Publish("user."+token, UserEvent{id, "error"})
			if err != nil {
				srv.Log.Errorw("File save publish error", "token", token, "file", id, "error", err)
			}
		}
	}()
	return id, nil

}

type UserEvent struct {
	FileID string
	State  string
}

// SaveUploadedFile is a copy of gin.Context.SaveUploadedFile without context dep
func (srv Service) saveUploadedFile(file *multipart.FileHeader, dst, token, id string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	if err != nil {
		return err
	}
	time.Sleep(2 * time.Second) // эмуляция записи большого файла
	err = srv.FileStateChange(id, "saved")

	return err
}

func (srv Service) FileStateChange(id, state string) error {
	var token string
	err := srv.db.Update(func(txn *badger.Txn) error {
		f, err := getFileMeta(txn, id)
		if err != nil {
			return err
		}
		f.State = state
		token = f.Token
		return setFileMeta(txn, f)
	})
	if err != nil {
		return err
	}
	//	srv.Log.Debugw("Raise event", "data", fmt.Sprintf("%+v", ev))
	err = srv.pubsub.Publish("user."+token, UserEvent{id, state})
	if err != nil {
		return err
	}
	err = srv.pubsub.Publish("file", UserEvent{id, state})
	return err
}

func (srv Service) FileList(token string) (files []File, err error) {

	err = srv.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("user." + token + ".")

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			fileID := k[len(prefix):]
			//srv.Log.Debugw("Got file key", "key", string(fileID))
			var f *File
			f, err = getFileMeta(txn, string(fileID))
			if err != nil {
				return err
			}
			files = append(files, *f)
		}
		srv.Log.Debugw("FileList", "fileCount", len(files))
		return nil
	})
	return
}

//	file, err := srv.store.File(token, c.Param("id"))

func (srv Service) File(token, id string) (fileMeta *File, filePath string, err error) {

	// check if file is owned by token
	err = srv.db.View(func(txn *badger.Txn) error {
		fileMeta, err = getFileMeta(txn, string(id))
		return err
	})
	if err != nil {
		return
	}
	if fileMeta.Token != token {
		err = errors.New("Owner not matched")
		return
	}

	filePath = filepath.Join(srv.filePath(id), id+".data")
	return
}

func getFileMeta(txn *badger.Txn, id string) (*File, error) {
	val, err := txn.Get([]byte("file." + id))
	if err != nil {
		return nil, err
	}
	var valCopy []byte
	err = val.Value(func(val []byte) error {
		valCopy = append([]byte{}, val...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(valCopy)
	dec := gob.NewDecoder(buf)
	var f File
	err = dec.Decode(&f)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func setFileMeta(txn *badger.Txn, f *File) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(f)
	if err != nil {
		return err
	}
	return txn.Set([]byte("file."+f.ID), buf.Bytes())
}
