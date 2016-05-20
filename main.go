package main

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"golang.org/x/net/websocket"

	"github.com/getblank/blank-sr/config"
	"github.com/getblank/blank-sr/registry"
	"github.com/getblank/blank-sr/sessionstore"
	"github.com/getblank/wango"
)

var (
	ErrInvalidArguments = errors.New("Invalid arguments")
	wamp                = wango.New()
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
	http.Handle("/", s)
	http.HandleFunc("/config", postConfigHandler)

	wamp.RegisterSubHandler("registry", registryHandler, nil)
	wamp.RegisterSubHandler("config", configHandler, nil)
	wamp.RegisterSubHandler("sessions", nil, nil)
	wamp.RegisterSubHandler("events", nil, nil)

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

	config.OnUpdate(func(c map[string]config.Store) {
		log.Info("Config updated. Will publish to receivers")
		wamp.Publish("config", c)
	})

	err := http.ListenAndServe(":1234", nil)
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

	// fmt.Println(t)
	rw.Write([]byte("OK"))
	config.ReloadConfig(data)
}
