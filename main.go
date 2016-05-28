package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/zipfs"

	"github.com/getblank/blank-sr/config"
	"github.com/getblank/blank-sr/registry"
	"github.com/getblank/blank-sr/sessionstore"
	"github.com/getblank/wango"
)

const (
	libZip    = "lib.zip"
	assetsZip = "assets.zip"
)

var (
	ErrInvalidArguments = errors.New("Invalid arguments")
	wamp                = wango.New()
	libFS               vfs.FileSystem
	assetsFS            vfs.FileSystem
	fsLocker            sync.RWMutex
	errLibCreateError   = errors.New("Error saving uploaded file")
)

func main() {
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

	wamp.RegisterSubHandler("registry", registryHandler, nil, nil)
	wamp.RegisterSubHandler("config", configHandler, nil, nil)
	wamp.RegisterSubHandler("sessions", subSessionsHandler, nil, nil)
	wamp.RegisterSubHandler("events", nil, nil, nil)

	wamp.RegisterRPCHandler("register", registerHandler)
	wamp.RegisterRPCHandler("publish", publishHandler)

	wamp.RegisterRPCHandler("session.new", newSessionHandler)
	wamp.RegisterRPCHandler("session.check", checkSessionByApiKeyHandler)
	wamp.RegisterRPCHandler("session.delete", deleteSessionHandler)
	wamp.RegisterRPCHandler("session.subscribed", sessionSubscribedHandler)
	wamp.RegisterRPCHandler("session.unsubscribed", sessionUnsubscribedHandler)
	wamp.RegisterRPCHandler("session.delete-connection", sessionDeleteConnectionHandler)

	registry.OnCreate(func() {
		services := registry.GetAll()
		wamp.Publish("registry", services)
	})

	registry.OnUpdate(func() {
		services := registry.GetAll()
		wamp.Publish("registry", services)
	})

	registry.OnDelete(func() {
		services := registry.GetAll()
		wamp.Publish("registry", services)
	})

	sessionstore.OnSessionUpdate(func(s *sessionstore.Session) {
		wamp.Publish("sessions", map[string]interface{}{"event": "updated", "data": s})
	})

	sessionstore.OnSessionDelete(func(s *sessionstore.Session) {
		wamp.Publish("sessions", map[string]interface{}{"event": "deleted", "data": s.APIKey})
	})

	config.OnUpdate(func(c map[string]config.Store) {
		log.Info("Config updated. Will publish to receivers")
		wamp.Publish("config", c)
	})

	makeLibFS()
	makeAssetsFS()

	err := http.ListenAndServe(":1234", mux)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func onSessionClose(c *wango.Conn) {
	println("Disconnected", c.ID())
	registry.Unregister(c.ID())
}

func onSessionOpen(c *wango.Conn) {
	println("New client", c.ID())
}

func publishSession(s *sessionstore.Session) {
	wamp.Publish("sessions", s)
}

func publishDeleteSession(s *sessionstore.Session) {
	wamp.Publish("sessions", map[string]interface{}{"apiKey": s.APIKey, "deleted": true})
}

func makeLibFS() {
	lib, err := ioutil.ReadFile(libZip)
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
	fsLocker.Unlock()
}

func makeAssetsFS() {
	lib, err := ioutil.ReadFile(assetsZip)
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
		err := postLibHandler(libZip, rw, request)
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
	b, err := vfs.ReadFile(libFS, request.RequestURI)
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
	return nil
}

func assetsHandler(rw http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodPost:
		err := postLibHandler(assetsZip, rw, request)
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

	b, err := vfs.ReadFile(assetsFS, request.RequestURI)
	fsLocker.RUnlock()
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("file not found"))
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(b)
}
