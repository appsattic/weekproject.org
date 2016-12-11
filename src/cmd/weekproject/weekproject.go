package main

import (
	"encoding/gob"
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

func getUserFromSession(session *sessions.Session) *User {
	user, ok := session.Values["user"].(*User)
	if !ok {
		return nil
	}
	return user
}

func init() {
	// tell gothic where our session store is
	gothic.Store = sessionStore

	// don't need `.Delims("[[", "]]")` since we're not using Vue.js here
	tmpl1, err := template.New("").ParseGlob("./templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl = tmpl1

	// Register the user with `gob` so we can serialise it.
	gob.Register(&User{})
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

		fmt.Printf("auth=%#v\n", authUser)

		// set this info in the session
		session.Values["id"] = authUser.UserID
		session.Values["name"] = authUser.NickName
		session.Values["title"] = authUser.Name
		session.Values["email"] = authUser.Email

		// read this user out of the store

		// firstly, see if this user is in the store and insert if not

		// read this user out of the store

		// ToDo: save this user in the store
		user := User{
			Id:    provider + "-" + authUser.UserID,
			Name:  authUser.NickName,
			Title: authUser.Name,
			Email: authUser.Email,
		}
		fmt.Printf("user=%#v\n", user)

		session.Values["user"] = &user

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
		delete(session.Values, "name")
		delete(session.Values, "title")
		delete(session.Values, "email")
		delete(session.Values, "user")
		session.Save(r, w)

		// redirect to somewhere else
		http.Redirect(w, r, "/", http.StatusFound)
	})

	// Publicly Viewable Projects
	p.Get("/u/:userName/p/:projectName/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title    string
			UserName string
		}{
			"Project by User",
			"",
		}
		render(w, "user-u-project-p.html", data)
	})

	// Projects
	p.Get("/p/new", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/p/new" {
			http.NotFound(w, r)
			return
		}

		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)
		if user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// render the new form
		data := struct {
			Title    string
			User     *User
			UserName string
		}{
			"New Project",
			user,
			"",
		}
		render(w, "project-new.html", data)
	})

	// Specific Project
	p.Get("/p/:projectName/", func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)
		if user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		data := struct {
			Title    string
			User     *User
			UserName string
		}{
			"Project by User",
			user,
			"",
		}
		render(w, "project-info.html", data)
	})

	// Projects
	p.Get("/p/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/p/" {
			http.NotFound(w, r)
			return
		}

		// firstly, check if this user is logged in
		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)
		if user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// get some things from the session
		id := getStringFromSession(session, "id")
		name := getStringFromSession(session, "name")
		email := getStringFromSession(session, "email")

		// check the id only
		if id == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// get a list of projects

		data := struct {
			Title    string
			User     *User
			Id       string
			UserName string
			Email    string
			Projects []Project
		}{
			"Your Projects",
			user,
			id,
			name,
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
		user := getUserFromSession(session)

		id := getStringFromSession(session, "id")
		name := getStringFromSession(session, "name")
		email := getStringFromSession(session, "email")

		data := struct {
			Title    string
			Id       string
			UserName string
			Email    string
			User     *User
		}{
			"The Week Project",
			id,
			name,
			email,
			user,
		}

		render(w, "index.html", data)
	})

	// server
	err := http.ListenAndServe(":"+port, p)
	log.Fatal(err)
}
