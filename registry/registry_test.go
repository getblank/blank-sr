package registry

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestRegister(t *testing.T) {
	g := Goblin(t)

	g.Describe("Service Register", func() {
		g.Describe("#Register", func() {
			g.Before(func() {
				services = map[string][]Service{}
			})
			g.It("Should register service", func() {
				Register("type1", "addr1", "8888", "id1", "")
				g.Assert(len(services["type1"])).Equal(1)
			})

			g.It("Should register same type service", func() {
				Register("type1", "addr2", "9999", "id2", "")
				g.Assert(len(services["type1"])).Equal(2)
			})

			g.It("Should register another type service", func() {
				Register("type3", "addr3", "8881", "id3", "")
				g.Assert(len(services["type3"])).Equal(1)
			})
		})

		g.Describe("#Unregister", func() {
			g.Before(func() {
				services = map[string][]Service{}
				Register("type1", "addr1", "8888", "id1", "")
				Register("type1", "addr2", "9999", "id2", "")
				Register("type3", "addr3", "8881", "id3", "")
			})
			g.Describe("Should remove service", func() {
				g.It("and in registry only one service of type 'type1'", func() {
					lenBeforeRemoving := len(services["type1"])
					Unregister("id1")
					g.Assert(len(services["type1"])).Equal(lenBeforeRemoving - 1)
				})
				g.It("Last service must be with id 'id2'", func() {
					g.Assert(services["type1"][0].connID).Equal("id2")
				})
			})
			g.Describe("Should remove service of another type", func() {
				g.It("and registered zero service of type 'type2'", func() {
					Unregister("id2")
					g.Assert(len(services["type2"])).Equal(0)
				})
			})
		})

	})

}
