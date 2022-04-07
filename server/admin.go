package server

import (
	"github.com/discord-plays/website/res"
	"github.com/gorilla/mux"
	"net/http"
)

func SetupDiscordPlaysAdmin(dpHttp *DiscordPlaysHttp, router *mux.Router) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		_, dpUser, _ := dpHttp.dpSess.CheckLogin(req)
		dpHttp.generatePage(rw, dpUser, "Discord Plays Admin", res.GetTemplateFileByName("admin.go.html"), nil)
	})
}
