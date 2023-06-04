package server

import (
	nfHttp "code.mrmelon54.com/melon/neutered-filesystem/http"
	"fmt"
	"github.com/discord-plays/website/res"
	"github.com/discord-plays/website/structure"
	"net/http"

	"github.com/gorilla/mux"
)

func SetupDiscordPlaysRoot(dpHttp *DiscordPlaysHttp, router *mux.Router, linkDiscord, linkNotion, linkGithub string) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		_, dpUser, _ := dpHttp.dpSess.CheckLogin(req)
		dpHttp.rwSync.RLock()
		defer dpHttp.rwSync.RUnlock()
		dpHttp.generatePage(rw, dpUser, "Discord Plays", res.GetTemplateFileByName("index.go.html"), struct {
			Projects      []*structure.ProjectItem
			Protocol      string
			ProjectDomain string
		}{
			Projects:      dpHttp.projectData,
			Protocol:      dpHttp.Protocol,
			ProjectDomain: dpHttp.Domain.ProjectDomain,
		})
	})
	router.HandleFunc("/bots/{botName}", func(rw http.ResponseWriter, req *http.Request) {
		_, dpUser, _ := dpHttp.dpSess.CheckLogin(req)
		vars := mux.Vars(req)
		botName := vars["botName"]
		if b, ok := getProjectItemFromName(dpHttp, botName); ok {
			dpHttp.generatePage(rw, dpUser, "Discord Plays "+*b.Name, res.GetTemplateFileByName("project.go.html"), struct {
				Project    *structure.ProjectItem
				ProjectUrl string
			}{
				Project:    b,
				ProjectUrl: fmt.Sprintf("%s://%s%s", dpHttp.Protocol, *b.Code, dpHttp.Domain.ProjectDomain),
			})
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	router.HandleFunc("/about", func(rw http.ResponseWriter, req *http.Request) {
		_, dpUser, _ := dpHttp.dpSess.CheckLogin(req)
		dpHttp.generatePage(rw, dpUser, "About", res.GetTemplateFileByName("about.go.html"), nil)
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
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(nfHttp.New(http.FS(res.GetAssetsFilesystem())))))
}
