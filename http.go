package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	//go:embed views/head.go.html
	headFile string
	//go:embed views/index.go.html
	indexFile string
)

type DiscordPlaysHttp struct {
	httpSrv *http.Server
}

func (dpHttp *DiscordPlaysHttp) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dpHttp.httpSrv.Shutdown(ctx)
}

func (dpHttp *DiscordPlaysHttp) StartupHttp(port int, wg *sync.WaitGroup) {
	wg.Add(1)
	log.Printf("[Http::Bind] Starting HTTP server on %d\n", port)
	go dpHttp.startHttpServer(port, wg)
}

func (dpHttp *DiscordPlaysHttp) startHttpServer(port int, wg *sync.WaitGroup) {
	defer wg.Done()

	linkDiscord := os.Getenv("LINK_DISCORD")
	linkNotion := os.Getenv("LINK_NOTION")
	linkGithub := os.Getenv("LINK_GITHUB")

	router := mux.NewRouter()
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		dpHttp.generatePage(rw, "Discord Plays", indexFile, nil)
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

	dpHttp.httpSrv = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
	err := dpHttp.httpSrv.ListenAndServe()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Printf("[Http] The HTTP server shutdown successfully\n")
		} else {
			log.Printf("[Http] Error trying to host the HTTP server: %s\n", err.Error())
		}
	}
}

func (dpHttp *DiscordPlaysHttp) generatePage(rw http.ResponseWriter, title, templatePage string, data interface{}) {
	rw.Header().Add("Content-Type", "text/html")
	rw.Write([]byte("<!DOCTYPE html><html><head>"))
	fillPage(rw, "head", headFile, struct{ Title string }{Title: title})
	rw.Write([]byte("</head><body>"))
	fillPage(rw, "body", templatePage, data)
	rw.Write([]byte("</body></html>"))
}

func fillPage(w io.Writer, name string, tempStr string, data interface{}) error {
	tmpl, err := template.New(name).Parse(tempStr)
	if err != nil {
		log.Printf("[Http::GeneratePage] Parse: %v\n", err)
		return err
	}
	if data == nil {
		err = tmpl.Execute(w, struct{}{})
	} else {
		err = tmpl.Execute(w, data)
	}
	if err != nil {
		log.Printf("[Http::GeneratePage] Execute: %v\n", err)
	}
	return nil
}
