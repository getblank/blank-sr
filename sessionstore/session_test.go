package sessionstore

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestSession(t *testing.T) {
	g := Goblin(t)
	g.Describe("Session Store", func() {
		g.Before(func() {
			db.DeleteBucket(bucket)
			Init()
		})
		g.Describe("#New", func() {
			g.It("User id must equals provided", func() {
				s := New(map[string]interface{}{"_id": "234"}, "")
				g.Assert(s.GetUserID()).Equal("234")
			})
			g.It("Must generate new APIKey", func() {
				s := New(map[string]interface{}{"_id": "userId"}, "")
				g.Assert(s.GetAPIKey() != "").IsTrue()
			})
			g.It("Must use provided APIKey", func() {
				s := New(map[string]interface{}{"_id": "userId"}, "42")
				g.Assert(s.GetAPIKey() == "42").IsTrue()
			})
			g.It("Should has 3 sessions", func() {
				g.Assert(len(sessions)).Equal(3)
			})
		})

		g.Describe("#GetByApiKey", func() {
			var user = map[string]interface{}{"_id": "345"}
			g.It("Should return session", func() {
				newS := New(user, "")
				s, err := GetByAPIKey(newS.GetAPIKey())
				g.Assert(err).Equal(nil)
				g.Assert(s.GetUserID()).Equal(user["_id"])
			})
		})

		g.Describe("#GetByUserId", func() {
			var user = map[string]interface{}{"_id": "456"}
			g.It("Should return session", func() {
				newS := New(user, "")
				s, err := GetByUserID(user["_id"])
				g.Assert(err).Equal(nil)
				g.Assert(s.GetAPIKey()).Equal(newS.GetAPIKey())
			})
		})

		g.Describe("#Delete", func() {
			var user = map[string]interface{}{"_id": "567"}
			g.It("Should delete session", func() {
				newS := New(user, "")
				totalSessions := len(sessions)
				newS.Delete()
				g.Assert(len(sessions)).Equal(totalSessions - 1)
				_, err := GetByUserID(user)
				g.Assert(err == nil).IsFalse()

				newS = New(user, "")
				totalSessions = len(sessions)
				Delete(newS.GetAPIKey())
				g.Assert(len(sessions)).Equal(totalSessions - 1)
			})
		})

		g.Describe("#AddSubscription", func() {
			var user = map[string]interface{}{"_id": "678"}
			g.It("Should add subscription uri with connID to session", func() {
				newS := New(user, "")
				g.Assert(len(newS.Connections)).Equal(0)
				var connID = "!!!"
				var uri = "com.sub"
				newS.AddSubscription(connID, uri, nil)
				g.Assert(len(newS.Connections)).Equal(1)
				g.Assert(len(newS.Connections[0].Subscriptions)).Equal(1)
			})
		})

		g.Describe("#DeleteSubscription", func() {
			var user = map[string]interface{}{"_id": "789"}
			g.It("Should delete subscription uri from session", func() {
				newS := New(user, "")
				g.Assert(len(newS.Connections)).Equal(0)
				var connID = "!!!"
				var uri = "com.sub"
				newS.AddSubscription(connID, uri, nil)
				g.Assert(len(newS.Connections)).Equal(1)
				newS.DeleteSubscription(connID, uri)
				g.Assert(len(newS.Connections[0].Subscriptions)).Equal(0)
			})
		})

		g.Describe("#DeleteConnection", func() {
			var user = map[string]interface{}{"_id": "890"}
			g.It("Should delete connection with connId from session", func() {
				newS := New(user, "")
				g.Assert(len(newS.Connections)).Equal(0)
				var connID = "!!!"
				var uri = "com.sub"
				newS.AddSubscription(connID, uri, nil)
				g.Assert(len(newS.Connections)).Equal(1)
				newS.DeleteConnection(connID)
				g.Assert(len(newS.Connections)).Equal(0)
			})
		})
	})
}
