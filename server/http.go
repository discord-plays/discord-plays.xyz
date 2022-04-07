package server

import (
	"context"
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"fmt"
	"github.com/discord-plays/website/res"
	"github.com/discord-plays/website/structure"
	"github.com/discord-plays/website/utils"
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
	projectData   []*structure.ProjectItem
	projectItems  map[string]*structure.ProjectItem
	projectHeader []string
	rwSync        *sync.RWMutex
	Protocol      string
	Domain        *structure.Domains
	oAuthConf     *oauth2.Config
	dpSess        *DiscordPlaysSessions
	dpAdmins      []string
}

func New(db *gorm.DB) *DiscordPlaysHttp {
	return &DiscordPlaysHttp{
		db:           db,
		projectData:  make([]*structure.ProjectItem, 0),
		projectItems: make(map[string]*structure.ProjectItem),
		rwSync:       &sync.RWMutex{},
	}
}

func (dpHttp *DiscordPlaysHttp) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return dpHttp.httpSrv.Shutdown(ctx)
}

func (dpHttp *DiscordPlaysHttp) StartupHttp(port int, wg *sync.WaitGroup) {
	dpHttp.loadProjectsFromDB()

	wg.Add(1)
	log.Printf("[Http::Bind] Starting HTTP server on %d\n", port)
	go dpHttp.startHttpServer(port, wg)
}

func (dpHttp *DiscordPlaysHttp) loadProjectsFromDB() {
	dpHttp.rwSync.Lock()
	defer dpHttp.rwSync.Unlock()

	var projects []*structure.ProjectItem
	dpHttp.db.Model(&structure.ProjectItem{}).Find(&projects)

	projectMap := make(map[string]*structure.ProjectItem)
	for _, p := range projects {
		utils.EmptyStringIfNil(p.Code)
		utils.EmptyStringIfNil(p.Name)
		utils.EmptyStringIfNil(p.SubText)
		utils.EmptyStringIfNil(p.Description)
		utils.EmptyStringIfNil(p.Invite)
		utils.EmptyStringIfNil(p.ImageAlt)
		utils.EmptyStringIfNil(p.Notion)
		utils.EmptyStringIfNil(p.Github)

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

	dpHttp.Protocol = os.Getenv("PROTOCOL")
	dpHttp.Domain = &structure.Domains{
		RootDomain:    os.Getenv("ROOT_DOMAIN"),
		IdDomain:      os.Getenv("ID_DOMAIN"),
		AdminDomain:   os.Getenv("ADMIN_DOMAIN"),
		ProjectDomain: os.Getenv("PROJECT_DOMAIN"),
	}

	linkDiscord := os.Getenv("LINK_DISCORD")
	linkNotion := os.Getenv("LINK_NOTION")
	linkGithub := os.Getenv("LINK_GITHUB")

	discordClient := os.Getenv("DISCORD_CLIENT")
	discordSecret := os.Getenv("DISCORD_SECRET")

	dpHttp.dpAdmins = strings.Split(os.Getenv("DP_ADMINS"), ",")

	dpHttp.oAuthConf = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s://%s/auth/callback", dpHttp.Protocol, dpHttp.Domain.IdDomain),
		ClientID:     discordClient,
		ClientSecret: discordSecret,
		Scopes:       []string{discord.ScopeIdentify},
		Endpoint:     discord.Endpoint,
	}
	dpHttp.dpSess = NewDiscordPlaysSessions()

	router := mux.NewRouter()
	SetupDiscordPlaysRoot(dpHttp, router.Host(dpHttp.Domain.RootDomain).Subrouter(), linkDiscord, linkNotion, linkGithub)
	SetupDiscordPlaysId(dpHttp, router.Host(dpHttp.Domain.IdDomain).Subrouter())
	SetupDiscordPlaysAdmin(dpHttp, router.Host(dpHttp.Domain.AdminDomain).Subrouter())
	SetupDiscordPlaysProjects(dpHttp, router)
	router.HandleFunc("/login", func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, fmt.Sprintf("%s://%s/login?redirect=%s", dpHttp.Protocol, dpHttp.Domain.IdDomain, req.Host), http.StatusTemporaryRedirect)
	})
	router.HandleFunc("/logout", func(rw http.ResponseWriter, req *http.Request) {
		sess, _, _ := dpHttp.dpSess.CheckLogin(req)
		delete(sess.Values, "dpUser")
		_ = sess.Save(req, rw)
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

func (dpHttp *DiscordPlaysHttp) convertToDpBody(meBody *structure.DiscordMeBody) *structure.DiscordPlaysUserBody {
	if meBody == nil {
		return nil
	}
	hash := md5.Sum([]byte(meBody.Id))
	return &structure.DiscordPlaysUserBody{
		Id:       hex.EncodeToString(hash[:]),
		Username: fmt.Sprintf("%s#%s", meBody.Username, meBody.Discriminator),
		Avatar:   fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=256", meBody.Id, meBody.Avatar),
		Admin:    dpHttp.isAdminUser(meBody.Id),
	}
}

func (dpHttp *DiscordPlaysHttp) isAdminUser(a string) bool {
	for _, i := range dpHttp.dpAdmins {
		if i == a {
			return true
		}
	}
	return false
}

func (dpHttp *DiscordPlaysHttp) generatePage(rw http.ResponseWriter, dpUser *structure.DiscordMeBody, title, templatePage string, data interface{}) {
	funcMap := template.FuncMap{
		"mod": func(i, j int) int {
			return i % j
		},
	}

	dpHttp.rwSync.RLock()
	defer dpHttp.rwSync.RUnlock()

	dpMeUser := dpHttp.convertToDpBody(dpUser)

	rw.Header().Add("Content-Type", "text/html")
	_, _ = rw.Write([]byte("<!DOCTYPE html><html><head>"))
	fillPage(rw, "head", res.GetTemplateFileByName("head.go.html"), struct{ Title string }{Title: title})
	_, _ = rw.Write([]byte("</head><body class=\"bg-dark\">"))
	fillPage(rw, "nav", res.GetTemplateFileByName("nav.go.html"), struct {
		RootDomain       template.HTMLAttr
		IdDomain         template.HTMLAttr
		DiscordPlaysUser *structure.DiscordPlaysUserBody
		Projects         []*structure.ProjectItem
	}{
		RootDomain:       template.HTMLAttr(fmt.Sprintf("%s://%s", dpHttp.Protocol, dpHttp.Domain.RootDomain)),
		IdDomain:         template.HTMLAttr(fmt.Sprintf("%s://%s", dpHttp.Protocol, dpHttp.Domain.IdDomain)),
		DiscordPlaysUser: dpMeUser,
		Projects:         dpHttp.projectData,
	})
	fillPageWithFuncMap(rw, "body", templatePage, funcMap, data)
	_, _ = rw.Write([]byte("</body></html>"))
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
