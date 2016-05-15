package main

import (
	"net/http"

	"github.com/getblank/blank-sr/config"
	"github.com/getblank/blank-sr/registry"
	"github.com/getblank/wango"
	"golang.org/x/net/websocket"
)

func main() {
	config.Init("./config.json")

	wamp := wango.New()
	wamp.SetSessionCloseCallback(onSessionClose)

	s := new(websocket.Server)
	s.Handshake = func(c *websocket.Config, r *http.Request) error {
		return nil
	}
	s.Handler = func(ws *websocket.Conn) {
		println("Connect")
		wamp.WampHandler(ws, nil)
	}
	http.Handle("/", s)

	go func() {
		wango.Connect("ws://localhost:1234", "http://localhost:1234")
	}()

	wamp.RegisterSubHandler("registry", registryHandler, nil)
	wamp.RegisterSubHandler("config", configHandler, nil)
	wamp.RegisterRPCHandler("register", registry.RegisterHandler)

	registry.OnUpdate(func() {
		services := registry.GetAll()
		wamp.Publish("registry", services)
	})

	err := http.ListenAndServe(":1234", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func onSessionClose(c *wango.Conn) {
	registry.Unregister(c.ID())
}

func registryHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	services := registry.GetAll()
	return services, nil
}

func configHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	conf := config.GetAllStoreObjectsFromDb()
	return conf, nil
}
