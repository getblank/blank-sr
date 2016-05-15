package registry

import (
	"errors"
	"sync"

	"github.com/getblank/wango"
)

var (
	services       = Services{services: map[string][]Service{}}
	createHandlers = []func(){}
	updateHandlers = []func(){}
	deleteHandlers = []func(){}
)

const (
	TypeWorker = "worker"
	TypePBX    = "PBX"
	Type       = "TaskQueue"
)

type Services struct {
	services map[string][]Service
	locker   sync.RWMutex
}

type Service struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	connID  string
}

type RegisterMessage struct {
	Type string `json:"type"`
}

func GetAll() map[string][]Service {
	services.locker.RLock()
	defer services.locker.RUnlock()
	all := map[string][]Service{}
	for typ, _services := range services.services {
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

func (s Services) register(service Service) {
	s.locker.Lock()
	defer s.locker.Unlock()

	if s.services[service.Type] == nil {
		s.services[service.Type] = []Service{}
	}
	s.services[service.Type] = append(s.services[service.Type], service)
}

func (s Services) unregister(id string) {
	s.locker.Lock()
	defer s.locker.Unlock()
	for typ, ss := range s.services {
		for i, _ss := range ss {
			if _ss.connID == id {
				ss = append(ss[:i], ss[i+1:]...)
				s.services[typ] = ss
				for _, h := range deleteHandlers {
					h()
				}
				return
			}
		}
	}

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
	services.register(s)

	for _, h := range createHandlers {
		h()
	}

	return nil, nil
}

func Unregister(id string) {
	services.unregister(id)
}
