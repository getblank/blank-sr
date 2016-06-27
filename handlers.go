package main

import (
	"errors"
	"strings"

	"github.com/getblank/blank-sr/config"
	"github.com/getblank/blank-sr/localstorage"
	"github.com/getblank/blank-sr/mutex"
	"github.com/getblank/blank-sr/registry"
	"github.com/getblank/blank-sr/sessionstore"
	"github.com/getblank/wango"
	"github.com/labstack/gommon/log"
)

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
	remoteAddr := strings.Split(c.RemoteAddr(), ":")[0]
	if remoteAddr == "[" {
		remoteAddr = "127.0.0.1"
	}
	switch typ {
	case registry.TypeFileStore:
		remoteAddr = "http://" + remoteAddr
	default:
		remoteAddr = "ws://" + remoteAddr
	}
	var port string
	if _port, ok := mes["port"]; ok {
		port, ok = _port.(string)
	}
	var commonJS string
	if _commonJS, ok := mes["commonJS"]; ok {
		commonJS, ok = _commonJS.(string)
	}
	registry.Register(typ, remoteAddr, port, c.ID(), commonJS)

	return nil, nil
}

// args: uri string, event interface{}, subscribers array of connIDs
// This data will be transferred sent as event on "events" topic
func publishHandler(c *wango.Conn, _uri string, args ...interface{}) (interface{}, error) {
	uri, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	_, ok = args[2].([]interface{})
	if !ok {
		return nil, ErrInvalidArguments
	}
	message := map[string]interface{}{
		"event":       args[1],
		"subscribers": args[2],
		"uri":         uri,
	}
	wamp.Publish("events", message)

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
	var user interface{}
	if len(args) > 1 {
		user = args[1]
	}

	return sessionstore.New(userId, user).APIKey, nil
}

func checkSessionByAPIKeyHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if args == nil {
		return nil, ErrInvalidArguments
	}
	apiKey, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}

	s, err := sessionstore.GetByAPIKey(apiKey)
	if err != nil {
		return nil, err
	}
	return s.GetUserID(), nil
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
	log.Debug("Session delete request", args)
	if args == nil {
		return nil, ErrInvalidArguments
	}
	apiKey, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}

	s, err := sessionstore.GetByAPIKey(apiKey)
	if err != nil {
		return nil, err
	}
	s.Delete()

	return nil, nil
}

// args must have 4 members
// apiKey, connID, uri string, extra interface{}
func sessionSubscribedHandler(c *wango.Conn, _uri string, args ...interface{}) (interface{}, error) {
	log.Debug("Session subscribe request", args)
	if len(args) < 4 {
		return nil, ErrInvalidArguments
	}
	apiKey, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	s, err := sessionstore.GetByAPIKey(apiKey)
	if err != nil {
		return nil, err
	}
	connID, ok := args[1].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	uri, ok := args[2].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	extra := args[3]
	s.AddSubscription(connID, uri, extra)

	publishSession(s)

	return nil, nil
}

// args must have 3 members
// apiKey, connID, uri string
func sessionUnsubscribedHandler(c *wango.Conn, _uri string, args ...interface{}) (interface{}, error) {
	log.Debug("Session unsubscribe request", args)
	if len(args) < 3 {
		return nil, ErrInvalidArguments
	}
	apiKey, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	s, err := sessionstore.GetByAPIKey(apiKey)
	if err != nil {
		return nil, err
	}
	connID, ok := args[1].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	uri, ok := args[2].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	s.DeleteSubscription(connID, uri)
	publishSession(s)

	return nil, nil
}

// args must have 2 members
// apiKey, connID string
func sessionDeleteConnectionHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	log.Debug("Session delete connection request", args)
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	apiKey, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	s, err := sessionstore.GetByAPIKey(apiKey)
	if err != nil {
		return nil, err
	}
	connID, ok := args[1].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	s.DeleteConnection(connID)
	publishSession(s)

	return nil, nil
}

// args must have 1 or 2 members
// userID string, user interface{} (optional)
func sessionUserUpdateHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	log.Debug("Session update request", args)
	if len(args) == 0 {
		return nil, ErrInvalidArguments
	}
	userID, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	if len(args) == 1 {
		sessionstore.DeleteAllForUser(userID)
		return nil, nil
	}
	sessionstore.UpdateUser(userID, args[1])

	return nil, nil
}

func subSessionsHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	all := sessionstore.GetAll()
	return map[string]interface{}{"event": "init", "data": all}, nil
}

func mutexLockHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, ErrInvalidArguments
	}
	id, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	mutex.Lock(c.ID(), id)
	return nil, nil
}

func mutexUnlockHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, ErrInvalidArguments
	}
	id, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	mutex.Unlock(c.ID(), id)
	return nil, nil
}

func localStorageGetItemHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, ErrInvalidArguments
	}
	id, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}

	return localstorage.GetItem(id), nil
}

func localStorageSetItemHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	id, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	item, ok := args[1].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}

	return localstorage.SetItem(id, item), nil
}

func localStorageRemoveItemHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, ErrInvalidArguments
	}
	id, ok := args[0].(string)
	if !ok {
		return nil, ErrInvalidArguments
	}
	localstorage.RemoveItem(id)
	return nil, nil
}

func localStorageClearHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
	localstorage.Clear()
	return nil, nil
}
