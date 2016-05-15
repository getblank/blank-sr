package main

import (
	_ "github.com/getblank/blank-sr/config"
	"github.com/getblank/blank-sr/wango"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	"golang.org/x/net/websocket"
)

func main() {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		websocket.Handler(func(ws *websocket.Conn) {
			wampServer.WampHandler(ws, nil)
		})
		return nil
	})
	wamp := wango.New()
	e.Run(standard.New(":1323"))
}
