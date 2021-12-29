package main

import (
	"context"
	_ "embed"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/jszwec/csvutil"
)

type DiscordPlaysHttp struct {
	httpSrv       *http.Server
	projectData   []*ProjectItem
	projectItems  map[string]*ProjectItem
	projectHeader []string
	rwSync        *sync.RWMutex
	rootDomain    string
	adminDomain   string
	protocol      string
	projectDomain string
}

type ProjectItem struct {
	Code        string
	Name        string
	SubText     string
	Description string
	Invite      string
	ImageAlt    string
	Notion      string
	Github      string
}

func (dpHttp *DiscordPlaysHttp) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dpHttp.httpSrv.Shutdown(ctx)
}

func (dpHttp *DiscordPlaysHttp) StartupHttp(port int, wg *sync.WaitGroup) {
	dpHttp.rwSync = &sync.RWMutex{}
	dpHttp.projectHeader = make([]string, 0)
	dpHttp.projectData = make([]*ProjectItem, 0)
	dpHttp.projectItems = make(map[string]*ProjectItem)

	h, err := csvutil.Header(ProjectItem{}, "csv")
	if err != nil {
		log.Fatal(err)
	}
	dpHttp.projectHeader = h

	dpHttp.loadProjectCsv()

	wg.Add(1)
	log.Printf("[Http::Bind] Starting HTTP server on %d\n", port)
	go dpHttp.startHttpServer(port, wg)
}

func (dpHttp *DiscordPlaysHttp) loadProjectCsv() {
	f, err := os.OpenFile("projects.csv", os.O_RDONLY, 0666)
	if err != nil {
		log.Printf("Failed to open 'projects.csv': %s\n", err.Error())
		return
	}
	r := csv.NewReader(f)
	d, err := csvutil.NewDecoder(r, dpHttp.projectHeader...)
	if err != nil {
		log.Printf("Failed to create csv decoder: %s\n", err.Error())
		return
	}

	dpHttp.rwSync.Lock()
	defer dpHttp.rwSync.Unlock()

	projects := make([]*ProjectItem, 0)
	projectMap := make(map[string]*ProjectItem)
	isFirst := true
	for {
		var p ProjectItem
		if err := d.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Failed to decode csv item: %s\n", err.Error())
			continue
		}
		if isFirst {
			isFirst = false
			continue
		}

		// Trim spaces lol
		p.Code = strings.TrimSpace(p.Code)
		p.Name = strings.TrimSpace(p.Name)
		p.SubText = strings.TrimSpace(p.SubText)
		p.Description = strings.TrimSpace(p.Description)
		p.Invite = strings.TrimSpace(p.Invite)
		p.ImageAlt = strings.TrimSpace(p.ImageAlt)
		p.Notion = strings.TrimSpace(p.Notion)
		p.Github = strings.TrimSpace(p.Github)

		projects = append(projects, &p)
		projectMap[p.Code] = &p
	}

	dpHttp.projectData = projects
	dpHttp.projectItems = projectMap
}

func (dpHttp *DiscordPlaysHttp) startHttpServer(port int, wg *sync.WaitGroup) {
	defer wg.Done()

	dpHttp.protocol = os.Getenv("PROTOCOL")
	dpHttp.rootDomain = os.Getenv("ROOT_DOMAIN")
	dpHttp.adminDomain = os.Getenv("ADMIN_DOMAIN")
	dpHttp.projectDomain = os.Getenv("PROJECT_DOMAIN")

	linkDiscord := os.Getenv("LINK_DISCORD")
	linkNotion := os.Getenv("LINK_NOTION")
	linkGithub := os.Getenv("LINK_GITHUB")

	router := mux.NewRouter()
	SetupDiscordPlaysRoot(dpHttp, router.Host(dpHttp.rootDomain).Subrouter(), linkDiscord, linkNotion, linkGithub)
	SetupDiscordPlaysAdmin(dpHttp, router.Host(dpHttp.adminDomain).Subrouter())
	SetupDiscordPlaysProjects(dpHttp, router)

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
	funcMap := template.FuncMap{
		"mod": func(i, j int) int {
			return i % j
		},
	}

	dpHttp.rwSync.RLock()
	defer dpHttp.rwSync.RUnlock()

	rw.Header().Add("Content-Type", "text/html")
	rw.Write([]byte("<!DOCTYPE html><html><head>"))
	fillPage(rw, "head", getTemplateFileByName("head.go.html"), struct{ Title string }{Title: title})
	rw.Write([]byte("</head><body class=\"bg-dark\">"))
	fillPage(rw, "nav", getTemplateFileByName("nav.go.html"), struct {
		RootDomain template.HTMLAttr
		Projects   []*ProjectItem
	}{
		RootDomain: template.HTMLAttr(fmt.Sprintf("%s://%s", dpHttp.protocol, dpHttp.rootDomain)),
		Projects:   dpHttp.projectData,
	})
	fillPageWithFuncMap(rw, "body", templatePage, funcMap, data)
	rw.Write([]byte("</body></html>"))
}

func fillPage(w io.Writer, name string, tempStr string, data interface{}) {
	tmpl, err := template.New(name).Parse(tempStr)
	if err != nil {
		log.Printf("[Http::GeneratePage] Parse: %v\n", err)
		return
	}
	if data == nil {
		err = tmpl.Execute(w, struct{}{})
	} else {
		err = tmpl.Execute(w, data)
	}
	if err != nil {
		log.Printf("[Http::GeneratePage] Execute: %v\n", err)
	}
}

func fillPageWithFuncMap(w io.Writer, name string, tempStr string, funcMap template.FuncMap, data interface{}) {
	tmpl := template.New(name)
	tmpl.Funcs(funcMap)
	_, err := tmpl.Parse(tempStr)
	if err != nil {
		log.Printf("[Http::GeneratePage Parse: %v\n", err)
		return
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("[Http::GeneratePage Parse: %v\n", err)
	}
}
