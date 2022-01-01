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
	"io/ioutil"
	"net/http"
	"time"
)

const (
	LoginFrameStart = "<script>window.opener.postMessage({user:"
	LoginFrameEnd   = "},\"%s://%s\");window.close();</script>"
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
		sess, _, ok := dpHttp.dpSess.CheckLogin(req)
		if ok {
			http.Redirect(rw, req, fmt.Sprintf("%s://%s", dpHttp.protocol, dpHttp.rootDomain), http.StatusTemporaryRedirect)
			return
		}

		state := dpHttp.dpSess.GetStateToken(sess)
		sess.Save(req, rw)
		http.Redirect(rw, req, dpHttp.oAuthConf.AuthCodeURL(state), http.StatusTemporaryRedirect)
	})
	router.HandleFunc("/auth/callback", func(rw http.ResponseWriter, req *http.Request) {
		sess, _, ok := dpHttp.dpSess.CheckLogin(req)
		if !ok {
			if req.FormValue("state") != dpHttp.dpSess.GetStateToken(sess) {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("State does not match."))
				return
			}
			// Step 3: We exchange the code we got for an access token
			// Then we can use the access token to do actions, limited to scopes we requested
			token, err := dpHttp.oAuthConf.Exchange(context.Background(), req.FormValue("code"))

			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(err.Error()))
				return
			}

			// Step 4: Use the access token, here we use it to get the logged in user's info.
			res, err := dpHttp.oAuthConf.Client(context.Background(), token).Get("https://discord.com/api/users/@me")

			if err != nil || res.StatusCode != 200 {
				rw.WriteHeader(http.StatusInternalServerError)
				if err != nil {
					rw.Write([]byte(err.Error()))
				} else {
					rw.Write([]byte(res.Status))
				}
				return
			}

			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(err.Error()))
				return
			}

			meBody := &discordMeBody{}
			err = json.Unmarshal(body, meBody)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(err.Error()))
				return
			}
			meBody.LoggedInUntil = time.Now().Add(2 * time.Hour)
			s := new(bytes.Buffer)
			g := gob.NewEncoder(s)
			g.Encode(meBody)
			sess.Values["dpUser"] = s.Bytes()
			sess.Save(req, rw)

			dpBody := convertToDpBody(dpHttp, meBody)
			j, err := json.Marshal(dpBody)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(err.Error()))
				return
			}

			var redirectDomain = ""
			if redirectableDomain, ok := sess.Values["RedirectDomain"].(string); ok {
				redirectDomain = redirectableDomain
			}

			rw.Write([]byte(LoginFrameStart))
			rw.Write(j)
			rw.Write([]byte(fmt.Sprintf(LoginFrameEnd, dpHttp.protocol, redirectDomain)))
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
