package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type DiscordPlaysHttp struct {
	db            *gorm.DB
	httpSrv       *http.Server
	projectData   []*ProjectItem
	projectItems  map[string]*ProjectItem
	projectHeader []string
	rwSync        *sync.RWMutex
	protocol      string
	rootDomain    string
	idDomain      string
	adminDomain   string
	projectDomain string
	oAuthConf     *oauth2.Config
	dpSess        *DiscordPlaysSessions
	dpAdmins      []string
}

func (dpHttp *DiscordPlaysHttp) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dpHttp.httpSrv.Shutdown(ctx)
}

func (dpHttp *DiscordPlaysHttp) StartupHttp(port int, wg *sync.WaitGroup) {
	dpHttp.rwSync = &sync.RWMutex{}
	dpHttp.rwSync.Lock()
	dpHttp.projectData = make([]*ProjectItem, 0)
	dpHttp.projectItems = make(map[string]*ProjectItem)
	dpHttp.rwSync.Unlock()

	dpHttp.loadProjectsFromDB()

	wg.Add(1)
	log.Printf("[Http::Bind] Starting HTTP server on %d\n", port)
	go dpHttp.startHttpServer(port, wg)
}

func (dpHttp *DiscordPlaysHttp) loadProjectsFromDB() {
	dpHttp.rwSync.Lock()
	defer dpHttp.rwSync.Unlock()

	var projects []*ProjectItem
	dpHttp.db.Model(&ProjectItem{}).Find(&projects)

	projectMap := make(map[string]*ProjectItem)
	for _, p := range projects {
		emptyStringIfNull(p.Code)
		emptyStringIfNull(p.Name)
		emptyStringIfNull(p.SubText)
		emptyStringIfNull(p.Description)
		emptyStringIfNull(p.Invite)
		emptyStringIfNull(p.ImageAlt)
		emptyStringIfNull(p.Notion)
		emptyStringIfNull(p.Github)

		// Trim spaces lol
		*p.Code = strings.TrimSpace(*p.Code)
		*p.Name = strings.TrimSpace(*p.Name)
		*p.SubText = strings.TrimSpace(*p.SubText)
		*p.Description = strings.TrimSpace(*p.Description)
		*p.Invite = strings.TrimSpace(*p.Invite)
		*p.ImageAlt = strings.TrimSpace(*p.ImageAlt)
		*p.Notion = strings.TrimSpace(*p.Notion)
		*p.Github = strings.TrimSpace(*p.Github)

		projectMap[*p.Code] = p
	}

	dpHttp.projectData = projects
	dpHttp.projectItems = projectMap
}

func (dpHttp *DiscordPlaysHttp) startHttpServer(port int, wg *sync.WaitGroup) {
	defer wg.Done()

	dpHttp.protocol = os.Getenv("PROTOCOL")
	dpHttp.rootDomain = os.Getenv("ROOT_DOMAIN")
	dpHttp.idDomain = os.Getenv("ID_DOMAIN")
	dpHttp.adminDomain = os.Getenv("ADMIN_DOMAIN")
	dpHttp.projectDomain = os.Getenv("PROJECT_DOMAIN")

	linkDiscord := os.Getenv("LINK_DISCORD")
	linkNotion := os.Getenv("LINK_NOTION")
	linkGithub := os.Getenv("LINK_GITHUB")

	discordClient := os.Getenv("DISCORD_CLIENT")
	discordSecret := os.Getenv("DISCORD_SECRET")

	dpHttp.dpAdmins = strings.Split(os.Getenv("DP_ADMINS"), ",")

	dpHttp.oAuthConf = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s://%s/auth/callback", dpHttp.protocol, dpHttp.idDomain),
		ClientID:     discordClient,
		ClientSecret: discordSecret,
		Scopes:       []string{discord.ScopeIdentify},
		Endpoint:     discord.Endpoint,
	}
	dpHttp.dpSess = NewDiscordPlaysSessions()

	router := mux.NewRouter()
	SetupDiscordPlaysRoot(dpHttp, router.Host(dpHttp.rootDomain).Subrouter(), linkDiscord, linkNotion, linkGithub)
	SetupDiscordPlaysId(dpHttp, router.Host(dpHttp.idDomain).Subrouter())
	SetupDiscordPlaysAdmin(dpHttp, router.Host(dpHttp.adminDomain).Subrouter())
	SetupDiscordPlaysProjects(dpHttp, router)
	router.HandleFunc("/login", func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, fmt.Sprintf("%s://%s/login?redirect=%s", dpHttp.protocol, dpHttp.idDomain, req.Host), http.StatusTemporaryRedirect)
	})
	router.HandleFunc("/logout", func(rw http.ResponseWriter, req *http.Request) {
		sess, _, _ := dpHttp.dpSess.CheckLogin(req)
		delete(sess.Values, "dpUser")
		sess.Save(req, rw)
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

func (dpHttp *DiscordPlaysHttp) generatePage(rw http.ResponseWriter, dpUser *discordMeBody, title, templatePage string, data interface{}) {
	funcMap := template.FuncMap{
		"mod": func(i, j int) int {
			return i % j
		},
	}

	dpHttp.rwSync.RLock()
	defer dpHttp.rwSync.RUnlock()

	dpMeUser := convertToDpBody(dpHttp, dpUser)

	rw.Header().Add("Content-Type", "text/html")
	rw.Write([]byte("<!DOCTYPE html><html><head>"))
	fillPage(rw, "head", getTemplateFileByName("head.go.html"), struct{ Title string }{Title: title})
	rw.Write([]byte("</head><body class=\"bg-dark\">"))
	fillPage(rw, "nav", getTemplateFileByName("nav.go.html"), struct {
		RootDomain       template.HTMLAttr
		IdDomain         template.HTMLAttr
		DiscordPlaysUser *discordPlaysUserBody
		Projects         []*ProjectItem
	}{
		RootDomain:       template.HTMLAttr(fmt.Sprintf("%s://%s", dpHttp.protocol, dpHttp.rootDomain)),
		IdDomain:         template.HTMLAttr(fmt.Sprintf("%s://%s", dpHttp.protocol, dpHttp.idDomain)),
		DiscordPlaysUser: dpMeUser,
		Projects:         dpHttp.projectData,
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
