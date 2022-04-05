package main

import (
	"fmt"
	"net/http"

	neuteredFilesystem "code.mrmelon54.xyz/sean/neutered-filesystem"
	"github.com/gorilla/mux"
)

func SetupDiscordPlaysRoot(dpHttp *DiscordPlaysHttp, router *mux.Router, linkDiscord, linkNotion, linkGithub string) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		_, dpUser, _ := dpHttp.dpSess.CheckLogin(req)
		dpHttp.rwSync.RLock()
		defer dpHttp.rwSync.RUnlock()
		dpHttp.generatePage(rw, dpUser, "Discord Plays", getTemplateFileByName("index.go.html"), struct {
			Projects      []*ProjectItem
			Protocol      string
			ProjectDomain string
		}{
			Projects:      dpHttp.projectData,
			Protocol:      dpHttp.protocol,
			ProjectDomain: dpHttp.projectDomain,
		})
	})
	router.HandleFunc("/bots/{botName}", func(rw http.ResponseWriter, req *http.Request) {
		_, dpUser, _ := dpHttp.dpSess.CheckLogin(req)
		vars := mux.Vars(req)
		botName := vars["botName"]
		if b, ok := getProjectItemFromName(dpHttp, botName); ok {
			dpHttp.generatePage(rw, dpUser, "Discord Plays "+*b.Name, getTemplateFileByName("project.go.html"), struct {
				Project    *ProjectItem
				ProjectUrl string
			}{
				Project:    b,
				ProjectUrl: fmt.Sprintf("%s://%s%s", dpHttp.protocol, *b.Code, dpHttp.projectDomain),
			})
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	router.HandleFunc("/about", func(rw http.ResponseWriter, req *http.Request) {
		_, dpUser, _ := dpHttp.dpSess.CheckLogin(req)
		dpHttp.generatePage(rw, dpUser, "About", getTemplateFileByName("about.go.html"), nil)
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
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(neuteredFilesystem.New(http.FS(getAssetsFilesystem())))))
}
