package main

import (
	"code.mrmelon54.xyz/sean/neutered-filesystem"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"strings"
)

func SetupDiscordPlaysProjects(dpHttp *DiscordPlaysHttp, router *mux.Router) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			rw.Header().Set("Location", fmt.Sprintf("%s://%s/bots/%s", dpHttp.protocol, dpHttp.rootDomain, *b.Code))
			rw.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	router.HandleFunc("/notion", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			rw.Header().Set("Location", *b.Notion)
			rw.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	router.HandleFunc("/github", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			rw.Header().Set("Location", *b.Github)
			rw.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	router.HandleFunc("/assets/logo.png", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			rw.Header().Set("Content-Type", "image/png")
			f, err := getAssetsFilesystem().Open(fmt.Sprintf("projects/%s.png", *b.Code))
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
	return getProjectItemFromName(dpHttp, a)
}

func getProjectItemFromName(dpHttp *DiscordPlaysHttp, name string) (*ProjectItem, bool) {
	dpHttp.rwSync.RLock()
	defer dpHttp.rwSync.RUnlock()
	b, ok := dpHttp.projectItems[name]
	return b, ok
}

func getFirstPartOfHost(a string) string {
	s := strings.Split(a, ".")
	if len(s) >= 3 {
		return s[len(s)-3]
	}
	return ""
}
