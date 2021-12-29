package main

import (
	"net/http"

	"github.com/gorilla/mux"
	neutered_filesystem "tea.melonie54.xyz/sean/neutered-filesystem"
)

func SetupDiscordPlaysRoot(dpHttp *DiscordPlaysHttp, router *mux.Router, linkDiscord, linkNotion, linkGithub string) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		dpHttp.rwSync.RLock()
		defer dpHttp.rwSync.RUnlock()
		dpHttp.generatePage(rw, "Discord Plays", getTemplateFileByName("index.go.html"), struct {
			Projects      []*ProjectItem
			Protocol      string
			ProjectDomain string
		}{
			Projects:      dpHttp.projectData,
			Protocol:      dpHttp.protocol,
			ProjectDomain: dpHttp.projectDomain,
		})
	})
	router.HandleFunc("/about", func(rw http.ResponseWriter, req *http.Request) {
		dpHttp.generatePage(rw, "About", getTemplateFileByName("about.go.html"), nil)
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
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(neutered_filesystem.New(http.FS(getAssetsFilesystem())))))
}
