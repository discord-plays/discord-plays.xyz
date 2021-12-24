package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"strings"
	neutered_filesystem "tea.melonie54.xyz/sean/neutered-filesystem"
)

func SetupDiscordPlaysProjects(dpHttp *DiscordPlaysHttp, router *mux.Router) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			dpHttp.generatePage(rw, "Discord Plays", getTemplateFileByName("project.go.html"), struct {
				Project    *ProjectItem
				ProjectUrl string
			}{
				Project:    b,
				ProjectUrl: fmt.Sprintf("%s://%s%s", dpHttp.protocol, b.Code, dpHttp.projectDomain),
			})
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	router.HandleFunc("/notion", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			rw.Header().Set("Location", b.Notion)
			rw.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	router.HandleFunc("/github", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			rw.Header().Set("Location", b.Github)
			rw.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	router.HandleFunc("/assets/logo.png", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			rw.Header().Set("Content-Type", "image/png")
			f, err := getAssetsFilesystem().Open(fmt.Sprintf("projects/%s.png", b.Code))
			if err != nil {
				rw.WriteHeader(500)
			} else {
				io.Copy(rw, f)
			}
		}
	})
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(neutered_filesystem.New(http.FS(getAssetsFilesystem())))))
}

func getProjectItem(dpHttp *DiscordPlaysHttp, req *http.Request) (*ProjectItem, bool) {
	a := getFirstPartOfHost(req.Host)
	dpHttp.rwSync.RLock()
	defer dpHttp.rwSync.RUnlock()
	b, ok := dpHttp.projectItems[a]
	return b, ok
}

func getFirstPartOfHost(a string) string {
	s := strings.Split(a, ".")
	if len(s) >= 3 {
		return s[len(s)-3]
	}
	return ""
}
