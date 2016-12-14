package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/pat"
	"github.com/gorilla/schema"
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

var decoder = schema.NewDecoder()

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func render(w http.ResponseWriter, tmplName string, data interface{}) {
	log.Printf("render(): entry, name=%s", tmplName)
	defer log.Printf("render(): exit")

	buf := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(buf, tmplName, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
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

	// open the store
	db, errBoltOpen := bolt.Open("weekproject.db", 0666, &bolt.Options{Timeout: 1 * time.Second})
	check(errBoltOpen)
	defer db.Close()

	// Goth example setup : https://publish.li/goth-example-TQEVYjoH

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

		authUser, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("provider=%s\n", provider)
		fmt.Printf("authUser.Provider=%s\n", authUser.Provider)
		fmt.Printf("auth=%#v\n", authUser)

		// set this info in the session
		session.Values["id"] = authUser.UserID
		session.Values["name"] = authUser.NickName
		session.Values["title"] = authUser.Name
		session.Values["email"] = authUser.Email

		// read this user out of the store

		// firstly, see if this user is in the store and insert if not

		// read this user out of the store

		// save this social and user to the store
		social := Social{
			Id:   authUser.Provider + "-" + authUser.UserID,
			Name: authUser.NickName,
		}
		user := User{
			Name:  authUser.NickName,
			Title: authUser.Name,
			Email: authUser.Email,
		}

		newSocial, err := SocialIns(db, social)
		if err != nil {
			log.Printf("err inserting social: %v\n", err)
		}

		newUser, err := UserIns(db, user)
		if err != nil {
			log.Printf("err inserting user: %v\n", err)
		}

		fmt.Printf("social=%#v\n", social)
		fmt.Printf("user=%#v\n", user)
		fmt.Printf("new social=%#v\n", newSocial)
		fmt.Printf("new user=%#v\n", newUser)

		session.Values["user"] = &newUser

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
	p.Get("/u/{userName}/p/{projectName}/", func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)

		// get this provider name from the URL
		userName := r.URL.Query().Get(":userName")
		projectName := r.URL.Query().Get(":projectName")

		fmt.Printf("userName=%s\n", userName)
		fmt.Printf("projectName=%s\n", projectName)

		// try and retrieve this project from the store
		p, err := ProjectGet(db, userName, projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if p.Name == "" {
			http.NotFound(w, r)
			return
		}

		fmt.Printf("Path=%s\n", r.URL.Path)
		data := struct {
			Title    string
			SubTitle string
			User     *User
			Project  Project
		}{
			p.Title,
			"By " + p.UserName,
			user,
			p,
		}
		render(w, "u-user-p-project.html", data)
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
			SubTitle string
			User     *User
		}{
			"New Project",
			"",
			user,
		}
		render(w, "p-new.html", data)
	})

	p.Post("/p/new", func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)
		if user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		errParseForm := r.ParseForm()
		if errParseForm != nil {
			http.Error(w, errParseForm.Error(), http.StatusInternalServerError)
			return
		}

		project := Project{}
		errDecode := decoder.Decode(&project, r.PostForm)
		if errDecode != nil {
			http.Error(w, errDecode.Error(), http.StatusInternalServerError)
			return
		}
		project.UserName = user.Name

		if project.Validate() == false {
			fmt.Printf("validation errors = %#v\n", project.Error)
		}

		newProject, err := ProjectIns(db, project)
		if err != nil {
			// ToDo: re-render the form with errors
			http.Redirect(w, r, "/p/new", http.StatusFound)
			return
		}

		// all good
		http.Redirect(w, r, "/p/"+newProject.Name+"/", http.StatusFound)
	})

	// Specific Project
	p.Get("/p/{projectName}/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("/p/{projectName}/ : entry\n")
		defer fmt.Printf("/p/{projectName}/ : exit\n")

		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)
		if user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// get this provider name from the URL
		projectName := r.URL.Query().Get(":projectName")

		// try and retrieve this project from the store
		p, err := ProjectGet(db, user.Name, projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if p.Name == "" {
			http.NotFound(w, r)
			return
		}

		data := struct {
			Title    string
			SubTitle string
			User     *User
			Project  Project
		}{
			p.Title,
			"by " + p.UserName,
			user,
			p,
		}
		render(w, "p-project.html", data)
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

		// get a list of projects
		// ToDo: ... !
		projects, err := ProjectSel(db, user.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("projects=%#v\n", projects)

		data := struct {
			Title    string
			SubTitle string
			User     *User
			Projects []*Project
		}{
			"Your Projects",
			"",
			user,
			projects,
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

		data := struct {
			Title    string
			SubTitle string
			User     *User
		}{
			"The Week Project",
			"",
			user,
		}

		render(w, "index.html", data)
	})

	// server
	errServer := http.ListenAndServe(":"+port, p)
	check(errServer)
}
