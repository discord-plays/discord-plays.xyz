package main

import (
	"github.com/gorilla/mux"
	"net/http"
)

func SetupDiscordPlaysRoot(dpHttp *DiscordPlaysHttp, router *mux.Router, linkDiscord, linkNotion, linkGithub string) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		dpHttp.generatePage(rw, "Discord Plays", getTemplateFileByName("index.go.html"), nil)
	})
	router.HandleFunc("/discord", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Location", linkDiscord)
		rw.WriteHeader(http.StatusTemporaryRedirect)
	})
	router.HandleFunc("/notion", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Location", linkNotion)
		rw.WriteHeader(http.StatusTemporaryRedirect)
	})
	router.HandleFunc("/github", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Location", linkGithub)
		rw.WriteHeader(http.StatusTemporaryRedirect)
	})
}
