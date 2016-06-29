package sync

import (
	"testing"
	"time"

	. "github.com/franela/goblin"
)

func TestOnce(t *testing.T) {
	g := Goblin(t)
	g.Describe("#Once", func() {
		g.It("should return nil only once for one id", func() {
			var id = "1"
			g.Assert(Once(id) == nil).IsTrue()
			g.Assert(Once(id) == nil).IsFalse()
			g.Assert(Once(id) == nil).IsFalse()
			g.Assert(Once(id) == nil).IsFalse()
		})
		g.It("should return nil again after ttl reached", func() {
			var id = "2"
			ttl = time.Millisecond * 500
			g.Assert(Once(id) == nil).IsTrue()
			g.Assert(Once(id) == nil).IsFalse()
			g.Assert(Once(id) == nil).IsFalse()

			time.Sleep(time.Second)
			g.Assert(Once(id) == nil).IsTrue()
		})
	})
}
