package main

import (
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/net/websocket"

	"github.com/getblank/blank-sr/config"
	"github.com/getblank/blank-sr/registry"
	"github.com/getblank/blank-sr/sessionstore"
	"github.com/getblank/wango"
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
	if len(args) == 0 {
		return nil, errors.New("No register message")
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
	registry.Register(typ, c.RemoteAddr(), c.ID())

	return nil, nil
}
