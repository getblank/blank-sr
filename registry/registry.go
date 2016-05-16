package registry

import (
	"errors"
	"sync"

	"github.com/getblank/wango"
)

var (
	services       = map[string][]Service{}
	locker         sync.RWMutex
	createHandlers = []func(){}
	updateHandlers = []func(){}
	deleteHandlers = []func(){}
)

const (
	TypeWorker = "worker"
	TypePBX    = "PBX"
	Type       = "TaskQueue"
)

type Service struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	connID  string
}

type RegisterMessage struct {
	Type string `json:"type"`
}

func GetAll() map[string][]Service {
	locker.RLock()
	defer locker.RUnlock()
	all := map[string][]Service{}
	for typ, _services := range services {
		all[typ] = []Service{}
		for _, srv := range _services {
			all[typ] = append(all[typ], srv)
		}
	}
	return all
}

func OnCreate(fn func()) {
	createHandlers = append(createHandlers, fn)
}

func OnUpdate(fn func()) {
	updateHandlers = append(updateHandlers, fn)
}

func OnDelete(fn func()) {
	deleteHandlers = append(deleteHandlers, fn)
}

func RegisterHandler(c *wango.Conn, uri string, args ...interface{}) (interface{}, error) {
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
	remoteAddr := c.RemoteAddr()
	s := Service{
		Type:    typ,
		Address: remoteAddr,
		connID:  c.ID(),
	}
	register(s)

	for _, h := range createHandlers {
		h()
	}

	return nil, nil
}

func Unregister(id string) {
	unregister(id)
}

func register(service Service) {
	locker.Lock()
	defer locker.Unlock()

	if services[service.Type] == nil {
		services[service.Type] = []Service{}
	}
	services[service.Type] = append(services[service.Type], service)
}

func unregister(id string) {
	locker.Lock()
	defer locker.Unlock()
	for typ, ss := range services {
		for i, _ss := range ss {
			if _ss.connID == id {
				services[typ] = append(ss[:i], ss[i+1:]...)
				for _, h := range deleteHandlers {
					go h()
				}
				return
			}
		}
	}
}
