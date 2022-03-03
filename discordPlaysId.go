package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	LoginFrameStart = "<!DOCTYPE html><html><head><script>window.opener.postMessage({user:"
	LoginFrameEnd   = "},\"%s://%s\");window.close();</script></head></html>"
	CheckFrameStart = "<!DOCTYPE html><html><head><script>window.onload=function(){window.parent.postMessage({user:"
	CheckFrameEnd   = "},\"%s://%s\");window.addEventListener(\"message\",function(evt){if (evt.origin.endsWith(\"%s\")) {if(evt.data.logout==\"bye\"){console.log(\"logging out\");}}});}</script></head></html>"
)

type discordMeBody struct {
	Id            string    `json:"id"`
	Username      string    `json:"username"`
	Discriminator string    `json:"discriminator"`
	Avatar        string    `json:"avatar"`
	LoggedInUntil time.Time `json:"-"`
}

type discordPlaysUserBody struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Admin    bool   `json:"admin"`
}

func SetupDiscordPlaysId(dpHttp *DiscordPlaysHttp, router *mux.Router) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, fmt.Sprintf("%s://%s", dpHttp.protocol, dpHttp.rootDomain), http.StatusTemporaryRedirect)
	})
	router.HandleFunc("/login", func(rw http.ResponseWriter, req *http.Request) {
		redirectDomain := req.URL.Query().Get("redirect")
		sess, _, ok := dpHttp.dpSess.CheckLogin(req)
		sess.Values["RedirectDomain"] = redirectDomain
		_ = sess.Save(req, rw)
		if ok {
			if redirectDomain == "" {
				redirectDomain = dpHttp.rootDomain
			}
			http.Redirect(rw, req, fmt.Sprintf("%s://%s", dpHttp.protocol, redirectDomain), http.StatusTemporaryRedirect)
			return
		}

		state := dpHttp.dpSess.GetStateToken(sess)
		_ = sess.Save(req, rw)
		http.Redirect(rw, req, dpHttp.oAuthConf.AuthCodeURL(state), http.StatusTemporaryRedirect)
	})
	router.HandleFunc("/check", func(rw http.ResponseWriter, req *http.Request) {
		parentDomain := req.URL.Query().Get("parent")
		_, meBody, ok := dpHttp.dpSess.CheckLogin(req)
		if ok {
			dpBody := convertToDpBody(dpHttp, meBody)
			j, err := json.Marshal(dpBody)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			if parentDomain != dpHttp.rootDomain && !strings.HasSuffix(parentDomain, dpHttp.projectDomain) {
				parentDomain = dpHttp.rootDomain
			}

			_, _ = rw.Write([]byte(CheckFrameStart))
			_, _ = rw.Write(j)
			_, _ = rw.Write([]byte(fmt.Sprintf(CheckFrameEnd, dpHttp.protocol, parentDomain, dpHttp.projectDomain)))
			return
		}
		_, _ = rw.Write([]byte{})
	})
	router.HandleFunc("/auth/callback", func(rw http.ResponseWriter, req *http.Request) {
		sess, _, ok := dpHttp.dpSess.CheckLogin(req)
		if !ok {
			if req.FormValue("state") != dpHttp.dpSess.GetStateToken(sess) {
				rw.WriteHeader(http.StatusBadRequest)
				_, _ = rw.Write([]byte("State does not match."))
				return
			}
			// Step 3: We exchange the code we got for an access token
			// Then we can use the access token to do actions, limited to scopes we requested
			token, err := dpHttp.oAuthConf.Exchange(context.Background(), req.FormValue("code"))

			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			// Step 4: Use the access token, here we use it to get the logged in user's info.
			res, err := dpHttp.oAuthConf.Client(context.Background(), token).Get("https://discord.com/api/users/@me")

			if err != nil || res.StatusCode != 200 {
				rw.WriteHeader(http.StatusInternalServerError)
				if err != nil {
					_, _ = rw.Write([]byte(err.Error()))
				} else {
					_, _ = rw.Write([]byte(res.Status))
				}
				return
			}

			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(res.Body)

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			meBody := &discordMeBody{}
			err = json.Unmarshal(body, meBody)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}
			meBody.LoggedInUntil = time.Now().Add(2 * time.Hour)
			s := new(bytes.Buffer)
			g := gob.NewEncoder(s)
			err = g.Encode(meBody)
			if err != nil {
				sess.Values["dpUser"] = []byte{}
			} else {
				sess.Values["dpUser"] = s.Bytes()
			}
			_ = sess.Save(req, rw)

			dpBody := convertToDpBody(dpHttp, meBody)
			j, err := json.Marshal(dpBody)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			var redirectDomain = ""
			if redirectableDomain, ok := sess.Values["RedirectDomain"].(string); ok {
				redirectDomain = redirectableDomain
			}

			log.Printf("%s :: %s\n", redirectDomain, dpHttp.projectDomain)
			log.Printf("%s :: %s\n", redirectDomain, dpHttp.rootDomain)
			if redirectDomain != dpHttp.rootDomain && !strings.HasSuffix(redirectDomain, dpHttp.projectDomain) {
				redirectDomain = dpHttp.rootDomain
			}

			_, _ = rw.Write([]byte(LoginFrameStart))
			_, _ = rw.Write(j)
			_, _ = rw.Write([]byte(fmt.Sprintf(LoginFrameEnd, dpHttp.protocol, redirectDomain)))
		}
	})
}

func convertToDpBody(dpHttp *DiscordPlaysHttp, meBody *discordMeBody) *discordPlaysUserBody {
	if meBody == nil {
		return nil
	}
	hash := md5.Sum([]byte(meBody.Id))
	return &discordPlaysUserBody{
		Id:       hex.EncodeToString(hash[:]),
		Username: fmt.Sprintf("%s#%s", meBody.Username, meBody.Discriminator),
		Avatar:   fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=256", meBody.Id, meBody.Avatar),
		Admin:    isAdminUser(dpHttp.dpAdmins, meBody.Id),
	}
}

func isAdminUser(a []string, b string) bool {
	for _, i := range a {
		if b == i {
			return true
		}
	}
	return false
}
