package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/twitter"
)

var sessionStore = sessions.NewCookieStore(
	[]byte(os.Getenv("SESSION_AUTH_KEY_V2")),
	[]byte(os.Getenv("SESSION_ENC_KEY_V2")),
	[]byte(os.Getenv("SESSION_AUTH_KEY_V1")),
	[]byte(os.Getenv("SESSION_ENC_KEY_V1")),
)

var tmpl *template.Template
var sessionName = "session"

func render(w http.ResponseWriter, templateName string, data interface{}) {
	err := tmpl.ExecuteTemplate(w, templateName, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getStringFromSession(session *sessions.Session, name string) string {
	id, ok := session.Values[name].(string)
	if !ok {
		id = ""
	}
	return id
}

func init() {
	gothic.Store = sessionStore

	// don't need `.Delims("[[", "]]")` since we're not using Vue.js here
	tmpl1, err := template.New("").ParseGlob("./templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl = tmpl1
}

func main() {
	baseUrl := os.Getenv("BASE_URL")
	port := os.Getenv("PORT")

	// twitter
	twitterConsumerKey := os.Getenv("TWITTER_CONSUMER_KEY")
	twitterSecretKey := os.Getenv("TWITTER_SECRET_KEY")
	twitter := twitter.NewAuthenticate(twitterConsumerKey, twitterSecretKey, baseUrl+"/auth/twitter/callback")

	// goth
	goth.UseProviders(twitter)

	// router
	p := pat.New()

	p.PathPrefix("/s/").Handler(http.FileServer(http.Dir("static")))

	p.Get("/auth/{provider}/callback", func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, sessionName)

		// get this provider name from the URL
		provider := r.URL.Query().Get(":provider")
		// fmt.Printf("provider=%s\n", provider)

		authUser, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// fmt.Printf("auth=%#v\n", authUser)

		// set this info in the session
		session.Values["id"] = authUser.UserID
		session.Values["username"] = authUser.NickName
		session.Values["email"] = authUser.Email

		// read this user out of the store

		// firstly, see if this user is in the store and insert if not

		// read this user out of the store

		// ToDo: save this user in the store
		user := User{
			Id:       provider + "-" + authUser.UserID,
			Username: provider + "-" + authUser.UserID,
			Email:    authUser.Email,
		}
		fmt.Printf("user=%#v\n", user)

		// save all sessions
		sessions.Save(r, w)

		// redirect to somewhere else
		http.Redirect(w, r, "/p/", http.StatusFound)
	})

	// begin auth
	p.Get("/auth/{provider}", gothic.BeginAuthHandler)

	// logout
	p.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, sessionName)

		// scrub user
		delete(session.Values, "id")
		delete(session.Values, "username")
		delete(session.Values, "email")
		session.Save(r, w)

		// redirect to somewhere else
		http.Redirect(w, r, "/", http.StatusFound)
	})

	// Projects
	p.Get("/p/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/p/" {
			http.NotFound(w, r)
			return
		}

		// firstly, check if this user is logged in
		session, _ := sessionStore.Get(r, sessionName)

		// get some things from the session
		id := getStringFromSession(session, "id")
		username := getStringFromSession(session, "username")
		email := getStringFromSession(session, "email")

		// check the id only
		if id == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// get a list of projects

		data := struct {
			Title    string
			Id       string
			Username string
			Email    string
			Projects []Project
		}{
			"Your Projects",
			id,
			username,
			email,
			make([]Project, 0),
		}

		render(w, "p.html", data)
	})

	// home
	p.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		session, _ := sessionStore.Get(r, sessionName)

		id := getStringFromSession(session, "id")
		username := getStringFromSession(session, "username")
		email := getStringFromSession(session, "email")

		data := struct {
			Title    string
			Id       string
			Username string
			Email    string
		}{
			"The Week Project",
			id,
			username,
			email,
		}

		render(w, "index.html", data)
	})

	// server
	err := http.ListenAndServe(":"+port, p)
	log.Fatal(err)
}
