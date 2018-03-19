package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/getblank/wango"
	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/zipfs"
	"gopkg.in/gemnasium/logrus-graylog-hook.v2"

	"github.com/getblank/blank-sr/config"
	"github.com/getblank/blank-sr/registry"
	"github.com/getblank/blank-sr/sessionstore"
	blankSync "github.com/getblank/blank-sr/sync"
)

const (
	libZipFileName    = "lib.zip"
	assetsZipFileName = "assets.zip"
)

var (
	buildTime string
	gitHash   string
	version   = "0.1.20"
)

var (
	// ErrInvalidArguments is an exported error
	ErrInvalidArguments = errors.New("Invalid arguments")
	wamp                = wango.New()
	libFS               vfs.FileSystem
	assetsFS            vfs.FileSystem
	libZip              []byte
	fsLocker            sync.RWMutex
	errLibCreateError   = errors.New("Error saving uploaded file")
	port                = "1234"
)

func main() {
	if os.Getenv("BLANK_DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	}
	log.SetFormatter(&log.JSONFormatter{})
	if os.Getenv("GRAYLOG2_HOST") != "" {
		host := os.Getenv("GRAYLOG2_HOST")
		port := os.Getenv("GRAYLOG2_PORT")
		if port == "" {
			port = "12201"
		}
		source := os.Getenv("GRAYLOG2_SOURCE")
		if source == "" {
			source = "blank-sr"
		}
		hook := graylog.NewGraylogHook(host+":"+port, map[string]interface{}{"source-app": source})
		log.AddHook(hook)
	}

	showVer := flag.Bool("v", false, "show version")
	flag.Parse()
	if *showVer {
		printVersion()
		return
	}

	start()
}

func printVersion() {
	fmt.Printf("blank-sr: \tv%s \t build time: %s \t commit hash: %s \n", version, buildTime, gitHash)
}

func start() {
	config.Init("./config.json")
	sessionstore.Init()

	wamp.SetSessionOpenCallback(onSessionOpen)
	wamp.SetSessionCloseCallback(onSessionClose)

	s := new(websocket.Server)
	s.Handshake = func(c *websocket.Config, r *http.Request) error {
		return nil
	}
	s.Handler = func(ws *websocket.Conn) {
		wamp.WampHandler(ws, nil)
	}

	mux := http.NewServeMux()
	mux.Handle("/", s)
	mux.HandleFunc("/config", postConfigHandler)
	mux.HandleFunc("/lib/", libHandler)
	mux.HandleFunc("/assets/", assetsHandler)
	mux.HandleFunc("/public-key", publicKeyHandler)

	wamp.RegisterSubHandler("registry", registryHandler, nil, nil)
	wamp.RegisterSubHandler("config", configHandler, nil, nil)
	wamp.RegisterSubHandler("sessions", subSessionsHandler, nil, nil)
	wamp.RegisterSubHandler("events", nil, nil, nil)
	wamp.RegisterSubHandler("users", nil, nil, nil)

	wamp.RegisterRPCHandler("register", registerHandler)
	wamp.RegisterRPCHandler("publish", publishHandler)

	wamp.RegisterRPCHandler("session.new", newSessionHandler)
	wamp.RegisterRPCHandler("session.check", checkSessionByAPIKeyHandler)
	wamp.RegisterRPCHandler("session.delete", deleteSessionHandler)
	wamp.RegisterRPCHandler("session.subscribed", sessionSubscribedHandler)
	wamp.RegisterRPCHandler("session.unsubscribed", sessionUnsubscribedHandler)
	wamp.RegisterRPCHandler("session.delete-connection", sessionDeleteConnectionHandler)
	wamp.RegisterRPCHandler("session.user-update", sessionUserUpdateHandler)

	wamp.RegisterRPCHandler("sync.lock", syncLockHandler)
	wamp.RegisterRPCHandler("sync.unlock", syncUnlockHandler)
	wamp.RegisterRPCHandler("sync.once", syncOnceHandler)

	wamp.RegisterRPCHandler("localStorage.getItem", localStorageGetItemHandler)
	wamp.RegisterRPCHandler("localStorage.setItem", localStorageSetItemHandler)
	wamp.RegisterRPCHandler("localStorage.removeItem", localStorageRemoveItemHandler)
	wamp.RegisterRPCHandler("localStorage.clear", localStorageClearHandler)

	registry.OnCreate(func(_ registry.Service) {
		services := registry.GetAll()
		wamp.Publish("registry", services)
	})

	registry.OnUpdate(func(_ registry.Service) {
		services := registry.GetAll()
		wamp.Publish("registry", services)
	})

	registry.OnDelete(func(s registry.Service) {
		if s.Type == "taskQueue" { // router restarted?
			sessionstore.DeleteAllConnections()
		}
		services := registry.GetAll()
		wamp.Publish("registry", services)
	})

	sessionstore.OnSessionUpdate(func(s *sessionstore.Session) {
		wamp.Publish("sessions", map[string]interface{}{"event": "updated", "data": s})
	})

	sessionstore.OnSessionDelete(func(s *sessionstore.Session) {
		wamp.Publish("sessions", map[string]interface{}{"event": "deleted", "data": s})
	})

	config.OnUpdate(func(c map[string]config.Store) {
		log.Info("Config updated. Will publish to receivers")
		wamp.Publish("config", c)
	})

	makeLibFS()
	makeAssetsFS()

	if srPort := os.Getenv("BLANK_SERVICE_REGISTRY_PORT"); len(srPort) > 0 {
		port = srPort
	}

	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func onSessionClose(c *wango.Conn) {
	println("Disconnected client from SR", c.ID())
	registry.Unregister(c.ID())
	blankSync.UnlockForOwner(c.ID())
}

func onSessionOpen(c *wango.Conn) {
	println("New client", c.ID())
}

func publishDeleteSession(s *sessionstore.Session) {
	wamp.Publish("sessions", map[string]interface{}{"apiKey": s.APIKey, "deleted": true})
}

func makeLibFS() {
	lib, err := ioutil.ReadFile(libZipFileName)
	if err != nil {
		log.WithError(err).Warn("No lib.zip file found")
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(lib), int64(len(lib)))
	if err != nil {
		log.WithError(err).Error("Can't make zip.Reader from lib.zip file ")
		return
	}
	rc := &zip.ReadCloser{
		Reader: *zr,
	}
	fsLocker.Lock()
	libFS = zipfs.New(rc, "lib")
	libZip = lib
	fsLocker.Unlock()
}

func makeAssetsFS() {
	lib, err := ioutil.ReadFile(assetsZipFileName)
	if err != nil {
		log.WithError(err).Warn("No assets.zip file found")
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(lib), int64(len(lib)))
	if err != nil {
		log.WithError(err).Error("Can't make zip.Reader from assets.zip file ")
		return
	}
	rc := &zip.ReadCloser{
		Reader: *zr,
	}
	fsLocker.Lock()
	assetsFS = zipfs.New(rc, "lib")
	fsLocker.Unlock()
}

func postConfigHandler(rw http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("Only POST request is allowed"))
		return
	}
	decoder := json.NewDecoder(request.Body)
	var data map[string]config.Store

	defer func() {
		if r := recover(); r != nil {
			rw.WriteHeader(http.StatusBadRequest)
			switch r.(type) {
			case string:
				rw.Write([]byte(r.(string)))
			case error:
				rw.Write([]byte(r.(error).Error()))
			}
		}
	}()
	err := decoder.Decode(&data)

	if err != nil {
		panic(err)
	}

	rw.Write([]byte("OK"))
	config.ReloadConfig(data)
}

func libHandler(rw http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodPost:
		err := postLibHandler(libZipFileName, rw, request)
		if err == nil {
			makeLibFS()
		}
	case http.MethodGet:
		getLibHandler(rw, request)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("Only GET and POST request is allowed"))
	}
}

func getLibHandler(rw http.ResponseWriter, request *http.Request) {
	fsLocker.RLock()
	if libFS == nil {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("file not found"))
		fsLocker.RUnlock()
		return
	}
	var b []byte
	var err error
	if request.RequestURI == "/lib/" {
		b = append([]byte{}, libZip...)
		rw.Header().Set("Content-Disposition", `attachment; filename="lib.zip"`)
	} else {
		b, err = vfs.ReadFile(libFS, strings.TrimPrefix(request.RequestURI, "/lib"))
		rw.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(request.RequestURI)+`"`)
	}
	fsLocker.RUnlock()
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("file not found"))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(b)
}

func postLibHandler(fileName string, rw http.ResponseWriter, request *http.Request) error {
	buf := bytes.NewBuffer(nil)
	_, err := buf.ReadFrom(request.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("can't read file"))
		return errLibCreateError
	}

	out, err := os.Create(fileName)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("can't create file"))
		return errLibCreateError
	}
	defer out.Close()
	written, err := io.Copy(out, buf)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("can't write file"))
		return errLibCreateError
	}
	log.Infof("new %s file created. Written %v bytes", fileName, written)
	wamp.Publish("config", config.Get())
	return nil
}

func publicKeyHandler(rw http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("Only GET request is allowed"))
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(sessionstore.PublicKeyBytes())
}

func assetsHandler(rw http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodPost:
		err := postLibHandler(assetsZipFileName, rw, request)
		if err == nil {
			makeAssetsFS()
		}

	case http.MethodGet:
		getAssetsHandler(rw, request)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("Only GET and POST request is allowed"))
	}
}

func getAssetsHandler(rw http.ResponseWriter, request *http.Request) {
	fsLocker.RLock()
	if assetsFS == nil {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("file not found"))
		fsLocker.RUnlock()
		return
	}

	b, err := vfs.ReadFile(assetsFS, strings.TrimPrefix(request.RequestURI, "/assets"))
	fsLocker.RUnlock()
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(b)
}
