package server

import (
	"code.mrmelon54.xyz/sean/neutered-filesystem"
	"fmt"
	"github.com/discord-plays/website/res"
	"github.com/discord-plays/website/structure"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"strings"
)

func SetupDiscordPlaysProjects(dpHttp *DiscordPlaysHttp, router *mux.Router) {
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		if b, ok := getProjectItem(dpHttp, req); ok {
			rw.Header().Set("Location", fmt.Sprintf("%s://%s/bots/%s", dpHttp.Protocol, dpHttp.Domain.RootDomain, *b.Code))
			rw.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			router.NotFoundHandler.ServeHTTP(rw, req)
		}
	})
	redirectToProjectAddress(dpHttp, router, "/invite", func(item *structure.ProjectItem) string {
		return *item.Invite
	})
	redirectToProjectAddress(dpHttp, router, "/notion", func(item *structure.ProjectItem) string {
		return *item.Notion
	})
	redirectToProjectAddress(dpHttp, router, "/github", func(item *structure.ProjectItem) string {
		return *item.Github
	})
	imageForProjectAddress(dpHttp, router, "logo")
	imageForProjectAddress(dpHttp, router, "banner")
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(neutered_filesystem.New(http.FS(res.GetAssetsFilesystem())))))
}

func getProjectItem(dpHttp *DiscordPlaysHttp, req *http.Request) (*structure.ProjectItem, bool) {
	a := getFirstPartOfHost(req.Host)
	return getProjectItemFromName(dpHttp, a)
}

func getProjectItemFromName(dpHttp *DiscordPlaysHttp, name string) (*structure.ProjectItem, bool) {
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

func redirectToProjectAddress(dpHttp *DiscordPlaysHttp, router *mux.Router, prefix string, cb func(*structure.ProjectItem) string) {
	router.HandleFunc(prefix, func(rw http.ResponseWriter, req *http.Request) {
		useProjectItem(dpHttp, req, func(item *structure.ProjectItem) {
			rw.Header().Set("Location", cb(item))
			rw.WriteHeader(http.StatusTemporaryRedirect)
		}, func() {
			router.NotFoundHandler.ServeHTTP(rw, req)
		})
	})
}

func imageForProjectAddress(dpHttp *DiscordPlaysHttp, router *mux.Router, name string) {
	router.HandleFunc("/assets/"+name+".png", func(rw http.ResponseWriter, req *http.Request) {
		useProjectItem(dpHttp, req, func(item *structure.ProjectItem) {
			rw.Header().Set("Content-Type", "image/png")
			f, err := res.GetAssetsFilesystem().Open(fmt.Sprintf("projects/%s/%s.png", *item.Code, name))
			if err != nil {
				rw.WriteHeader(500)
			} else {
				_, _ = io.Copy(rw, f)
			}
		}, func() {
			router.NotFoundHandler.ServeHTTP(rw, req)
		})
	})
}

func useProjectItem(dpHttp *DiscordPlaysHttp, req *http.Request, good func(item *structure.ProjectItem), bad func()) {
	if b, ok := getProjectItem(dpHttp, req); ok {
		good(b)
	} else {
		bad()
	}
}
