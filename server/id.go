package server

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/discord-plays/website/structure"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
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

func SetupDiscordPlaysId(dpHttp *DiscordPlaysHttp, router *mux.Router) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, fmt.Sprintf("%s://%s", dpHttp.Protocol, dpHttp.Domain.RootDomain), http.StatusTemporaryRedirect)
	})
	router.HandleFunc("/login", func(rw http.ResponseWriter, req *http.Request) {
		redirectDomain := req.URL.Query().Get("redirect")
		sess, _, ok := dpHttp.dpSess.CheckLogin(req)
		sess.Values["RedirectDomain"] = redirectDomain
		_ = sess.Save(req, rw)
		if ok {
			if redirectDomain == "" {
				redirectDomain = dpHttp.Domain.RootDomain
			}
			http.Redirect(rw, req, fmt.Sprintf("%s://%s", dpHttp.Protocol, redirectDomain), http.StatusTemporaryRedirect)
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
			dpBody := dpHttp.convertToDpBody(meBody)
			j, err := json.Marshal(dpBody)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			if parentDomain != dpHttp.Domain.RootDomain && !strings.HasSuffix(parentDomain, dpHttp.Domain.ProjectDomain) {
				parentDomain = dpHttp.Domain.RootDomain
			}

			_, _ = rw.Write([]byte(CheckFrameStart))
			_, _ = rw.Write(j)
			_, _ = rw.Write([]byte(fmt.Sprintf(CheckFrameEnd, dpHttp.Protocol, parentDomain, dpHttp.Domain.ProjectDomain)))
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

			meBody := &structure.DiscordMeBody{}
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

			dpBody := dpHttp.convertToDpBody(meBody)
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

			if redirectDomain != dpHttp.Domain.RootDomain && !strings.HasSuffix(redirectDomain, dpHttp.Domain.ProjectDomain) {
				redirectDomain = dpHttp.Domain.RootDomain
			}

			_, _ = rw.Write([]byte(LoginFrameStart))
			_, _ = rw.Write(j)
			_, _ = rw.Write([]byte(fmt.Sprintf(LoginFrameEnd, dpHttp.Protocol, redirectDomain)))
		}
	})
}
