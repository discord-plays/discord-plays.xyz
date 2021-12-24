package main

import (
	"github.com/gorilla/mux"
	"net/http"
)

func SetupDiscordPlaysAdmin(dpHttp *DiscordPlaysHttp, router *mux.Router) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		dpHttp.generatePage(rw, "Discord Plays Admin", getTemplateFileByName("admin.go.html"), nil)
	})
}
