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
	"github.com/gomiddleware/logger"
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

		newSocial, err := InsSocial(db, social)
		if err != nil {
			log.Printf("err inserting social: %v\n", err)
		}

		newUser, err := InsUser(db, user)
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
		// get this provider name from the URL
		userName := r.URL.Query().Get(":userName")
		projectName := r.URL.Query().Get(":projectName")

		fmt.Printf("userName=%s\n", userName)
		fmt.Printf("projectName=%s\n", projectName)

		// try and retrieve this project from the store
		p, err := GetProject(db, userName, projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if p.Name == "" {
			http.NotFound(w, r)
			return
		}

		// get a list of updates
		updates, err := SelUpdates(db, userName, projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// now check to see if a user is logged in
		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)

		fmt.Printf("Path=%s\n", r.URL.Path)
		data := struct {
			Title    string
			SubTitle string
			User     *User
			Project  Project
			Updates  []*Update
		}{
			p.Title,
			"by @" + p.UserName,
			user,
			p,
			updates,
		}
		render(w, "u-user-p-project.html", data)
	})

	// Publicly Viewable Projects
	p.Get("/u/{userName}/p/{projectName}", toSlash)

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

		err := InsProject(db, project)
		if err != nil {
			// ToDo: re-render the form with errors
			http.Redirect(w, r, "/p/new", http.StatusFound)
			return
		}

		// all good
		http.Redirect(w, r, "/p/"+project.Name+"/", http.StatusFound)
	})

	// Add an update to a project.
	p.Get("/p/{projectName}/update", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("GET /p/{projectName}/update : entry\n")
		defer log.Printf("GET /p/{projectName}/update : exit\n")

		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)
		if user == nil {
			log.Printf("/p/{projectName}/update : no user\n")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// get this project name from the URL
		projectName := r.URL.Query().Get(":projectName")
		log.Printf("/p/{projectName}/update : projectName=%s\n", projectName)

		// try and retrieve this project from the store
		p, err := GetProject(db, user.Name, projectName)
		if err != nil {
			log.Printf("/p/{projectName}/ : err GetProject : %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if p.Name == "" {
			log.Printf("/p/{projectName}/ : Not Found\n")
			http.NotFound(w, r)
			return
		}

		data := struct {
			Title    string
			SubTitle string
			User     *User
			Project  *Project
			Update   *Update
		}{
			p.Title,
			"",
			user,
			&p,
			&Update{},
		}
		render(w, "p-project-update.html", data)
	})

	// Add an update to a project.
	p.Post("/p/{projectName}/update", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("POST /p/{projectName}/update : entry\n")
		defer log.Printf("POST /p/{projectName}/update : exit\n")

		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)
		if user == nil {
			log.Printf("/p/{projectName}/update : no user\n")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// get this project name from the URL
		projectName := r.URL.Query().Get(":projectName")
		log.Printf("/p/{projectName}/update : projectName=%s\n", projectName)

		// try and retrieve this project from the store
		p, err := GetProject(db, user.Name, projectName)
		if err != nil {
			log.Printf("/p/{projectName}/ : err GetProject : %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if p.Name == "" {
			log.Printf("/p/{projectName}/ : Not Found\n")
			http.NotFound(w, r)
			return
		}

		// get the incoming form
		errParseForm := r.ParseForm()
		if errParseForm != nil {
			http.Error(w, errParseForm.Error(), http.StatusInternalServerError)
			return
		}

		update := Update{}
		errDecode := decoder.Decode(&update, r.PostForm)
		if errDecode != nil {
			http.Error(w, errDecode.Error(), http.StatusInternalServerError)
			return
		}

		if update.Validate() == false {
			data := struct {
				Title    string
				SubTitle string
				User     *User
				Project  *Project
				Update   *Update
			}{
				p.Title,
				"",
				user,
				&p,
				&update,
			}
			render(w, "p-project-update.html", data)
			return
		}

		errInsUpdate := InsUpdate(db, p, update)
		if errInsUpdate != nil {
			fmt.Printf("error inserting update = %#v\n", errInsUpdate)
			http.Redirect(w, r, "/p/"+projectName+"/update", http.StatusFound)
			return
		}

		http.Redirect(w, r, "/p/"+projectName+"/", http.StatusFound)
	})

	// Specific Project
	p.Get("/p/{projectName}/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("/p/{projectName}/ : entry\n")
		defer log.Printf("/p/{projectName}/ : exit\n")

		session, _ := sessionStore.Get(r, sessionName)
		user := getUserFromSession(session)
		if user == nil {
			log.Printf("/p/{projectName}/ : no user\n")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// get this provider name from the URL
		projectName := r.URL.Query().Get(":projectName")
		log.Printf("/p/{projectName}/ : projectName=%s\n", projectName)

		// try and retrieve this project from the store
		p, err := GetProject(db, user.Name, projectName)
		if err != nil {
			log.Printf("/p/{projectName}/ : err GetProject : %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if p.Name == "" {
			log.Printf("/p/{projectName}/ : Not Found\n")
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
			"by @" + p.UserName,
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
		projects, err := SelProjects(db, user.Name)
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

	// create the logger middleware
	log := logger.New()

	// server
	errServer := http.ListenAndServe(":"+port, log(p))
	check(errServer)
}
