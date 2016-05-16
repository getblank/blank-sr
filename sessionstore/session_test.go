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
		})
		g.Describe("#New", func() {
			g.It("User id must equals provided", func() {
				s := New("234")
				g.Assert(s.GetUserID()).Equal("234")
			})
			g.It("Must generate new APIKey", func() {
				s := New("userId")
				g.Assert(s.GetAPIKey() != "").IsTrue()
			})
			g.It("Should have 2 sessions", func() {
				g.Assert(len(sessions)).Equal(2)
			})
		})

		g.Describe("#GetByApiKey", func() {
			var userID = "345"
			g.It("Should return session", func() {
				newS := New(userID)
				s, err := GetByApiKey(newS.GetAPIKey())
				g.Assert(err).Equal(nil)
				g.Assert(s.GetUserID()).Equal(userID)
			})
		})

		g.Describe("#GetByUserId", func() {
			var userID = "456"
			g.It("Should return session", func() {
				newS := New(userID)
				s, err := GetByUserID(userID)
				g.Assert(err).Equal(nil)
				g.Assert(s.GetAPIKey()).Equal(newS.GetAPIKey())
			})
		})

		g.Describe("#Delete", func() {
			var userID = "567"
			g.It("Should delete session", func() {
				newS := New(userID)
				totalSessions := len(sessions)
				newS.Delete()
				g.Assert(len(sessions)).Equal(totalSessions - 1)
				_, err := GetByUserID(userID)
				g.Assert(err == nil).IsFalse()

				newS = New(userID)
				totalSessions = len(sessions)
				Delete(newS.GetAPIKey())
				g.Assert(len(sessions)).Equal(totalSessions - 1)
			})
		})
	})
}
