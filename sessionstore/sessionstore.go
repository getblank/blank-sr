package sessionstore

import (
	"encoding/json"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/getblank/blank-sr/bdb"
	"github.com/getblank/blank-sr/berror"
	"github.com/getblank/uuid"
)

var (
	bucket                = "__sessions"
	ttl                   = time.Hour * 24
	sessions              = map[string]*Session{}
	locker                sync.RWMutex
	sessionUpdateHandlers = []func(*Session){}
	sessionDeleteHandlers = []func(*Session){}
	db                    = bdb.DB{}
)

// Session represents user session in Blank
type Session struct {
	APIKey      string      `json:"apiKey"`
	UserID      string      `json:"userId"`
	Connections []*Conn     `json:"connections"`
	LastRequest time.Time   `json:"lastRequest"`
	User        interface{} `json:"user"`
	ttl         time.Duration
	sync.RWMutex
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
func New(userID string, user interface{}, tmp ...bool) *Session {
	s := &Session{
		uuid.NewV4(),
		userID,
		[]*Conn{},
		time.Now(),
		user,
		0,
		sync.RWMutex{},
	}
	if len(tmp) > 0 && tmp[0] {
		s.ttl = time.Minute
	}
	s.LastRequest = time.Now()
	locker.Lock()
	defer locker.Unlock()
	sessions[s.APIKey] = s
	sessionUpdated(s, true)
	return s
}

// GetAll returns all stored sessions
func GetAll() []*Session {
	result := make([]*Session, len(sessions))
	var i int
	locker.RLock()
	defer locker.RUnlock()
	for _, s := range sessions {
		result[i] = s
		i++
	}
	return result
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
		log.Error("Can't delete session", APIKey, err.Error())
	}
	locker.Lock()
	defer locker.Unlock()
	delete(sessions, APIKey)
}

// Delete removes
func DeleteAllForUser(userID string) {
	locker.RLock()
	defer locker.RUnlock()
	for _, s := range sessions {
		if s.UserID == userID {
			s.Delete()
		}
	}
}

func UpdateUser(userID string, user interface{}) {
	locker.RLock()
	defer locker.RUnlock()
	for _, s := range sessions {
		if s.UserID == userID {
			s.User = user
			sessionUpdated(s, true)
		}
	}
}

// AddSubscription adds subscription URI with provided params to user session
func (s *Session) AddSubscription(connID, uri string, extra interface{}) {
	s.Lock()
	defer s.Unlock()
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
	s.Lock()
	defer s.Unlock()
	for i, _c := range s.Connections {
		if _c.ConnID == connID {
			s.Connections = append(s.Connections[:i], s.Connections[i+1:]...)
			break
		}
	}
}

// DeleteSubscription deletes subscription from connection of user session
func (s *Session) DeleteSubscription(connID, uri string) {
	s.Lock()
	defer s.Unlock()
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
		log.Error("Can't delete session", s, err.Error())
	}
	locker.Lock()
	defer locker.Unlock()
	delete(sessions, s.APIKey)
	sessionDeleted(s)
}

// Save saves session in store and update LastRequest prop in it.
func (s *Session) Save(noLastRequestUpdate bool) {
	if !noLastRequestUpdate {
		s.LastRequest = time.Now()
	}
	s = copySession(s)
	err := db.Save(bucket, s.APIKey, s)
	if err != nil {
		log.Error("Can't save session", s, err.Error())
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

func OnSessionUpdate(handler func(*Session)) {
	sessionUpdateHandlers = append(sessionUpdateHandlers, handler)
	return
}

func OnSessionDelete(handler func(*Session)) {
	sessionDeleteHandlers = append(sessionDeleteHandlers, handler)
	return
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
	sessionUpdated(s)
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
				log.Error("Can't delete session", s, err.Error())
			}
			delete(sessions, s.APIKey)
		}
	}
}

func loadSessions() {
	_sessions, err := db.GetAll(bucket)
	if err != nil && err != berror.DbNotFound {
		log.Error("Can't read all sessions", err.Error())
	}
	now := time.Now()
	locker.Lock()
	defer locker.Unlock()
	for _, _s := range _sessions {
		var s Session
		err := json.Unmarshal(_s, &s)
		if err != nil {
			log.Error("Can't unmarshal session", _s, err.Error())
			continue
		}
		if now.Sub(s.LastRequest) > ttl {
			err := db.Delete(bucket, s.APIKey)
			if err != nil {
				log.Error("Can't delete session when Init()", s, err.Error())
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

func sessionUpdated(s *Session, userUpdated ...bool) {
	var b bool
	if userUpdated != nil {
		b = userUpdated[0]
	}
	s.Save(b)
	_s := copySession(s)
	if !b {
		_s.User = nil
	}

	for _, handler := range sessionUpdateHandlers {
		go handler(_s)
	}
}

func sessionDeleted(s *Session) {
	for _, handler := range sessionDeleteHandlers {
		go handler(s)
	}
}

func copySession(s *Session) *Session {
	_s := *s
	for i := range _s.Connections {
		c := *_s.Connections[i]
		subs := map[string]interface{}{}
		for k, v := range c.Subscriptions {
			subs[k] = v
		}
		c.Subscriptions = subs
		_s.Connections[i] = &c
	}
	return &_s
}
