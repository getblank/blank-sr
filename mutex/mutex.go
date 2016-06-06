package mutex

import "sync"

var (
	lockers         = map[string]*locker{}
	owners2Lockers  = map[string][]string{}
	lockersCounters = map[string]int{}
	mainLocker      sync.Mutex
)

type locker struct {
	ch chan struct{}
}

// Lock create new locker for provided id if it is not exists or takes existing, then locks it
func Lock(owner, id string) {
	mainLocker.Lock()
	m, ok := lockers[id]
	if !ok {
		m = new(locker)
		m.ch = make(chan struct{}, 1)
		lockers[id] = m
	}
	if _, ok := owners2Lockers[owner]; !ok {
		owners2Lockers[owner] = []string{}
	}
	owners2Lockers[owner] = append(owners2Lockers[owner], id)
	lockersCounters[id]++
	mainLocker.Unlock()

	m.lock()
}

// Unlock takes existing locker from map and unlocks it
func Unlock(owner, id string) {
	mainLocker.Lock()
	defer mainLocker.Unlock()
	m, ok := lockers[id]
	if !ok {
		return
	}
	m.unlock()
	lockersCounters[id]--
	if lockersCounters[id] == 0 {
		delete(lockers, id)
		delete(lockersCounters, id)
	}
	for i := len(owners2Lockers[owner]) - 1; i >= 0; i-- {
		_id := owners2Lockers[owner][i]
		if id == _id {
			owners2Lockers[owner] = append(owners2Lockers[owner][:i], owners2Lockers[owner][i+1:]...)
		}
		if len(owners2Lockers[owner]) == 0 {
			delete(owners2Lockers, owner)
		}
	}
}

// UnlockForOwner unlocks all lockers locked by owner
func UnlockForOwner(owner string) {
	mainLocker.Lock()
	defer mainLocker.Unlock()
	if locks, ok := owners2Lockers[owner]; ok {
		for _, id := range locks {
			lockers[id].unlock()
			lockersCounters[id]--
		}
		delete(owners2Lockers, owner)
	}
}

func (l *locker) lock() {
	l.ch <- struct{}{}
}

func (l *locker) unlock() {
	if len(l.ch) == 0 {
		panic("attempt to unlock no locked locker")
	}
	<-l.ch
}
