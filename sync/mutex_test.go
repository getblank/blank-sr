package sync

import (
	"testing"
	"time"

	. "github.com/franela/goblin"
)

func TestMutex(t *testing.T) {
	g := Goblin(t)

	g.Describe("#Lock", func() {
		g.It("should create new locker in map and lock it", func() {
			owner := "24"
			lockID := "42"
			Lock(owner, lockID)
			g.Assert(len(lockers)).Equal(1)
			g.Assert(lockers[lockID] == nil).IsFalse()
		})
		g.It("should create new owner slice in map", func() {
			owner := "224"
			lockID := "242"
			Lock(owner, lockID)
			g.Assert(owners2Lockers[owner] == nil).IsFalse()
			g.Assert(len(owners2Lockers[owner])).Equal(1)
		})
		g.It("should increment lockers counter", func() {
			owner := "2224"
			lockID := "2242"
			Lock(owner, lockID)
			g.Assert(lockersCounters[lockID]).Equal(1)
		})
	})

	g.Describe("#Unlock", func() {
		g.It("should remove locker from map after unlock", func() {
			owner := "124"
			lockID := "142"
			Lock(owner, lockID)
			g.Assert(lockers[lockID] == nil).IsFalse()
			Unlock(owner, lockID)
			g.Assert(lockers[lockID] == nil).IsTrue()
		})
		g.It("should remove owner from map after unlock", func() {
			owner := "1124"
			lockID := "1142"
			Lock(owner, lockID)
			g.Assert(owners2Lockers[owner] == nil).IsFalse()
			Unlock(owner, lockID)
			g.Assert(owners2Lockers[owner] == nil).IsTrue()
		})
		g.It("should decrement lockers counter", func() {
			owner := "12224"
			lockID := "12242"
			Lock(owner, lockID)
			g.Assert(lockersCounters[lockID]).Equal(1)
			Unlock(owner, lockID)
			g.Assert(lockersCounters[lockID]).Equal(0)
		})
		g.It("should return if no locker found", func() {
			Unlock("someOuner", "someLock")
		})
	})

	g.Describe("#UnlockForOwner", func() {
		g.It("should unlock all lockers for ownerID and remove owner", func(done Done) {
			owner := "111111"
			otherOwner := "222222"
			lockID1 := "11"
			lockID2 := "12"
			lockID3 := "13"
			Lock(owner, lockID1)
			Lock(owner, lockID2)
			Lock(owner, lockID3)
			g.Assert(owners2Lockers[owner] == nil).IsFalse()
			g.Assert(len(owners2Lockers[owner])).Equal(3)
			go func() {
				var locked bool
				go func() {
					time.Sleep(time.Millisecond * 10)
					g.Assert(locked).IsFalse()
				}()
				Lock(otherOwner, lockID3)
				locked = true
				g.Assert(len(owners2Lockers[owner])).Equal(0)
				g.Assert(owners2Lockers[owner] == nil).IsTrue()
				done()
			}()
			time.Sleep(time.Millisecond * 50)
			UnlockForOwner(owner)
		})
	})
}
