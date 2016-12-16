package main

import (
	"strings"
	"time"

	"github.com/Machiel/slugify"
)

type Social struct {
	Id       string // e.g. "twitter-123456"
	Name     string // e.g. "chilts" - the nickname they have in this system
	Inserted time.Time
	Updated  time.Time
}

type User struct {
	Name     string // e.g. "chilts" (ie. their Twitter handle)
	Title    string // e.g. "Andrew Chilton"
	Email    string // e.g. "andychilton@gmail.com"
	Inserted time.Time
	Updated  time.Time
}

type Project struct {
	Name     string            `schema:"-"`     // e.g. "week-project"
	Title    string            `schema:"Title"` // e.g. "The Week Project"
	Content  string            `schema:"Content"`
	UserName string            `schema:"-"` // e.g. "chilts" // ToDo: decide if we actually need this
	Inserted time.Time         `schema:"-"`
	Updated  time.Time         `schema:"-"`
	Error    map[string]string `json:"-"`
}

type Update struct {
	Status   string            `schema:"Status"`
	Progress int               `schema:"Progress"`
	Inserted time.Time         `schema:"-"`
	Updated  time.Time         `schema:"-"`
	Error    map[string]string `json:"-"`
}

// Validate firstly normalises the project, then validates it and returns either true (valid) or false (invalid). It sets any messages onto
// the Project.Error field.
func (p *Project) Validate() bool {
	now := time.Now().UTC()

	// normalise
	p.Name = slugify.Slugify(p.Title)
	p.Title = strings.TrimSpace(p.Title)
	p.Inserted = now
	p.Updated = now
	p.Error = make(map[string]string)

	// validate
	valid := true

	if len(p.Name) == 0 {
		p.Error["Name"] = "Name must be provided"
		valid = false
	}

	if len(p.Title) == 0 {
		p.Error["Title"] = "Title must be provided"
		valid = false
	}

	if len(p.UserName) == 0 {
		p.Error["UserName"] = "UserName must be provided"
		valid = false
	}

	return valid
}

func (u *Update) Validate() bool {
	now := time.Now().UTC()

	u.Inserted = now
	u.Updated = now
	u.Error = make(map[string]string)

	// ToDo: make sure Progress is in the range 0 to 100

	return true
}
