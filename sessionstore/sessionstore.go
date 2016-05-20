package sessionstore

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/getblank/blank-sr/bdb"
	"github.com/getblank/blank-sr/berror"
	"github.com/getblank/uuid"
	"github.com/ivahaev/go-logger"
)

var (
	bucket                = "__sessions"
	ttl                   = time.Hour * 24
	sessions              = map[string]*Session{}
	locker                sync.RWMutex
	sessionUpdateHandlers = []func(*Session){}
	db                    = bdb.DB{}
)

// Session represents user session in Blank
type Session struct {
	APIKey            string    `json:"apiKey"`
	UserID            string    `json:"userId"`
	Connections       []*Conn   `json:"connections"`
	LastRequest       time.Time `json:"lastRequest"`
	connectionsLocker sync.RWMutex
	ttl               time.Duration
}

// Conn represents WAMP connection in session
type Conn struct {
	ConnID        string                 `json:"connId"`
	Subscriptions map[string]interface{} `json:"subscriptions"`
}

// Init is the entrypoint of sessionstore
func Init() {
	loadSessions()
	go ttlWatcher()
}

// New created new user session. Optional bool param for creating session with 1 minute ttl
func New(userID string, tmp ...bool) *Session {
	s := &Session{
		uuid.NewV4(),
		userID,
		[]*Conn{},
		time.Now(),
		sync.RWMutex{},
		0,
	}
	if len(tmp) > 0 && tmp[0] {
		s.ttl = time.Minute
	}
	locker.Lock()
	defer locker.Unlock()
	sessions[s.APIKey] = s
	go sessionUpdated(s)
	return s
}

// GetByApiKey returns point to Session or error if it is not exists.
func GetByApiKey(APIKey string) (s *Session, err error) {
	return getByApiKey(APIKey)
}

// GetByApiKey returns point to Session or error if it is not exists.
func GetByUserID(id string) (s *Session, err error) {
	return getByUserId(id)
}

// Delete removes
func Delete(APIKey string) {
	err := db.Delete(bucket, APIKey)
	if err != nil {
		logger.Error("Can't delete session", APIKey, err.Error())
	}
	locker.Lock()
	defer locker.Unlock()
	delete(sessions, APIKey)
}

// AddSubscription adds subscription URI with provided params to user session
func (s *Session) AddSubscription(connID, uri string, extra interface{}) {
	s.connectionsLocker.Lock()
	defer s.connectionsLocker.Unlock()
	var c *Conn
	for _, _c := range s.Connections {
		if _c.ConnID == connID {
			c = _c
			break
		}
	}
	if c == nil {
		c = new(Conn)
		c.ConnID = connID
		c.Subscriptions = map[string]interface{}{}
		s.Connections = append(s.Connections, c)
	}
	c.Subscriptions[uri] = extra
}

// DeleteConnection deletes WAMP connection from user session
func (s *Session) DeleteConnection(connID string) {
	s.connectionsLocker.Lock()
	defer s.connectionsLocker.Unlock()
	for i, _c := range s.Connections {
		if _c.ConnID == connID {
			s.Connections = append(s.Connections[:i], s.Connections[i+1:]...)
			break
		}
	}
}

// DeleteSubscription deletes subscription from connection of user session
func (s *Session) DeleteSubscription(connID, uri string) {
	s.connectionsLocker.Lock()
	defer s.connectionsLocker.Unlock()
	var c *Conn
	for _, _c := range s.Connections {
		if _c.ConnID == connID {
			c = _c
			break
		}
	}
	if c == nil {
		return
	}
	delete(c.Subscriptions, uri)
}

// Delete removes Session from store
func (s *Session) Delete() {
	err := db.Delete(bucket, s.APIKey)
	if err != nil {
		logger.Error("Can't delete session", s, err.Error())
	}
	locker.Lock()
	defer locker.Unlock()
	delete(sessions, s.APIKey)
}

// Save saves session in store and update LastRequest prop in it.
func (s *Session) Save() {
	locker.Lock()
	defer locker.Unlock()
	s.LastRequest = time.Now()
	err := db.Save(bucket, s.APIKey, s)
	if err != nil {
		logger.Error("Can't save session", s, err.Error())
	}
}

// GetUserID returns userID stored in session
func (s *Session) GetUserID() string {
	return s.UserID
}

// GetUserID returns apiKey of session
func (s *Session) GetAPIKey() string {
	return s.APIKey
}

func getByApiKey(APIKey string) (s *Session, err error) {
	locker.Lock()
	defer locker.Unlock()
	s, ok := sessions[APIKey]
	if !ok {
		return s, berror.DbNotFound
	}
	if s.ttl > 0 {
		s.ttl = 0
		s.APIKey = uuid.NewV4()
		sessions[s.APIKey] = s
		delete(sessions, APIKey)
	}
	go sessionUpdated(s)
	return s, err
}

func getByUserId(id string) (s *Session, err error) {
	locker.RLock()
	defer locker.RUnlock()
	for _, v := range sessions {
		if v.UserID == id {
			return v, nil
		}
	}
	return nil, berror.DbNotFound
}

func clearRottenSessions() {
	locker.Lock()
	defer locker.Unlock()
	now := time.Now()
	for _, s := range sessions {
		if now.Sub(s.LastRequest) > ttl || (s.ttl > 0 && now.Sub(s.LastRequest) > s.ttl) {
			err := db.Delete(bucket, s.APIKey)
			if err != nil {
				logger.Error("Can't delete session", s, err.Error())
			}
			delete(sessions, s.APIKey)
		}
	}
}

func loadSessions() {
	_sessions, err := db.GetAll(bucket)
	if err != nil && err != berror.DbNotFound {
		logger.Error("Can't read all sessions", err.Error())
	}
	now := time.Now()
	locker.Lock()
	defer locker.Unlock()
	for _, _s := range _sessions {
		var s Session
		err := json.Unmarshal(_s, &s)
		if err != nil {
			logger.Error("Can't unmarshal session", _s, err.Error())
			continue
		}
		if now.Sub(s.LastRequest) > ttl {
			err := db.Delete(bucket, s.APIKey)
			if err != nil {
				logger.Error("Can't delete session when Init()", s, err.Error())
			}
			continue
		}
		sessions[s.APIKey] = &s
	}
}

func ttlWatcher() {
	c := time.Tick(time.Minute)
	for {
		<-c
		clearRottenSessions()
	}
}

func OnSessionUpdate(handler func(*Session)) {
	sessionUpdateHandlers = append(sessionUpdateHandlers, handler)
	return
}

func sessionUpdated(s *Session) {
	s.Save()
	for _, handler := range sessionUpdateHandlers {
		go handler(s)
	}
}
