package main

import (
	"bytes"
	"encoding/gob"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"net/http"
	"os"
	"time"
)

const cookieName = "DpSession"

type DiscordPlaysSessions struct {
	store *sessions.CookieStore
}

func NewDiscordPlaysSessions() *DiscordPlaysSessions {
	cookieSessions := sessions.NewCookieStore([]byte(os.Getenv("SESSION_ENCRYPTION")))
	cookieSessions.Options.SameSite = http.SameSiteLaxMode
	cookieSessions.Options.Domain = os.Getenv("COOKIE_DOMAIN")
	return &DiscordPlaysSessions{store: cookieSessions}
}

func (dpSess *DiscordPlaysSessions) CheckLogin(req *http.Request) (*sessions.Session, *discordMeBody, bool) {
	sess, _ := dpSess.store.Get(req, cookieName)
	sess.Options.SameSite = http.SameSiteLaxMode
	if dpUserBytes, ok := sess.Values["dpUser"].([]byte); ok {
		s := bytes.NewBuffer(dpUserBytes)
		g := gob.NewDecoder(s)
		dpUser := &discordMeBody{}
		g.Decode(dpUser)

		if time.Now().Before(dpUser.LoggedInUntil) {
			return sess, dpUser, true
		}
	}
	return sess, nil, false
}

func (dpSess *DiscordPlaysSessions) GetStateToken(sess *sessions.Session) string {
	if stateToken, ok := sess.Values["stateToken"].(string); ok {
		return stateToken
	}
	return dpSess.GenerateStateToken(sess)
}

func (dpSess *DiscordPlaysSessions) GenerateStateToken(sess *sessions.Session) string {
	u := uuid.NewString()
	sess.Values["stateToken"] = u
	return u
}
