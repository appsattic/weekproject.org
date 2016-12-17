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

	if len(p.Name) == 0 {
		p.Error["Name"] = "Name must be provided"
	}

	if len(p.Title) == 0 {
		p.Error["Title"] = "Title must be provided"
	}

	if len(p.UserName) == 0 {
		p.Error["UserName"] = "UserName must be provided"
	}

	return len(p.Error) == 0
}

func (u *Update) Validate() bool {
	// normalise
	now := time.Now().UTC()
	u.Inserted = now
	u.Updated = now
	u.Error = make(map[string]string)

	if len(u.Status) > 1000 {
		u.Error["Status"] = "Status should be less than 1,000 chars"
	}

	if u.Progress < 0 && u.Progress > 100 {
		u.Error["Progress"] = "Progress should be between 0 and 100 inclusive"
	}

	return len(u.Error) == 0
}
