package main

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/websocket"

	"github.com/getblank/blank-sr/config"
	"github.com/getblank/blank-sr/registry"
	"github.com/getblank/blank-sr/sessionstore"
	"github.com/getblank/wango"
)

var (
	ErrInvalidArguments = errors.New("Invalid arguments")
)

func main() {
	config.Init("./config.json")
	sessionstore.Init()

	wamp := wango.New()
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

	wamp.RegisterSubHandler("registry", registryHandler, nil)
	wamp.RegisterSubHandler("config", configHandler, nil)

	wamp.RegisterRPCHandler("register", registerHandler)

	wamp.RegisterRPCHandler("session.new", newSessionHandler)
	wamp.RegisterRPCHandler("session.get", getSessionByApiKeyHandler)
	wamp.RegisterRPCHandler("session.delete", deleteSessionHandler)

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

func registryHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	services := registry.GetAll()
	return services, nil
}

func configHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	conf := config.GetAllStoreObjectsFromDb()
	return conf, nil
}

func registerHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if args == nil {
		return nil, ErrInvalidArguments
	}

	mes, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, errors.New("Invalid register message")
	}

	_type, ok := mes["type"]
	if !ok {
		return nil, errors.New("Invalid register message. No type")
	}
	typ, ok := _type.(string)
	if !ok || typ == "" {
		return nil, errors.New("Invalid register message. No type")
	}
	remoteAddr := "ws://" + strings.Split(c.RemoteAddr(), ":")[0]
	registry.Register(typ, remoteAddr, c.ID())

	return nil, nil
}

func newSessionHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if args == nil {
		return nil, ErrInvalidArguments
	}
	userId, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}

	return sessionstore.New(userId), nil
}

func getSessionByApiKeyHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if args == nil {
		return nil, ErrInvalidArguments
	}
	apiKey, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}

	return sessionstore.GetByApiKey(apiKey)
}

func getSessionByUserIDHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if args == nil {
		return nil, ErrInvalidArguments
	}
	userID, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}

	return sessionstore.GetByUserID(userID)
}

func deleteSessionHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if args == nil {
		return nil, ErrInvalidArguments
	}
	apiKey, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}

	s, err := sessionstore.GetByApiKey(apiKey)
	if err != nil {
		return nil, err
	}
	s.Delete()

	return nil, nil
}
