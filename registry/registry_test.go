package registry

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestRegister(t *testing.T) {
	g := Goblin(t)

	g.Describe("Service Register", func() {
		g.Describe("#Register", func() {
			g.It("Should register service", func() {
				s := Service{"type1", "addr1", "id1"}
				services.register(s)
				g.Assert(len(services.services["type1"])).Equal(1)
			})

			g.It("Should register same type service", func() {
				s := Service{"type1", "addr2", "id2"}
				services.register(s)
				if len(services.services["type1"]) != 2 {
					t.Fatal("Service was not registered")
				}
			})

			g.It("Should register another type service", func() {
				s := Service{"type3", "addr3", "id3"}
				services.register(s)
				g.Assert(len(services.services["type3"])).Equal(1)
			})
		})

		g.Describe("#Unregister", func() {
			g.Describe("Should remove service", func() {
				g.It("Should registered one service of type 'type1'", func() {
					Unregister("id1")
					g.Assert(len(services.services["type1"])).Equal(1)
				})
				g.It("Last service must be with id 'id2'", func() {
					g.Assert(services.services["type1"][0].connID).Equal("id2")
				})
			})
			g.Describe("Should remove service of another type", func() {
				g.It("Should registered zero service of type 'type2'", func() {
					Unregister("id2")
					g.Assert(len(services.services["type2"])).Equal(0)
				})
			})
		})

	})

}
