package main

import (
	"errors"
	"time"

	"github.com/Machiel/slugify"
	"github.com/boltdb/bolt"
	"github.com/chilts/rod"
)

var (
	ErrLocationMustHaveOneBucket = errors.New("location must specify at least one bucket")
)

func SocialIns(db *bolt.DB, social Social) (Social, error) {
	// generate some fields
	now := time.Now()

	// create the user we're inserting
	s := Social{
		Id:       social.Id,
		Name:     social.Name,
		Inserted: now,
		Updated:  now,
	}

	err := db.Update(func(tx *bolt.Tx) error {
		location := "social"
		return rod.PutJson(tx, location, social.Id, s)
	})

	return s, err
}

func UserIns(db *bolt.DB, user User) (User, error) {
	// generate some fields
	now := time.Now()

	// create the user we're inserting
	u := User{
		Name:     user.Name,
		Title:    user.Title,
		Email:    user.Email,
		Inserted: now,
		Updated:  now,
	}

	err := db.Update(func(tx *bolt.Tx) error {
		location := "user." + user.Name
		return rod.PutJson(tx, location, "meta", u)
	})

	return u, err
}

// ProjectIns takes a skeleton project, sets various fields on it and inserts it into the store. It fails if this
// project already exists (under this user).
//
// We only use the Title, Content, and UserName fields. The rest are generated.
func ProjectIns(db *bolt.DB, project Project) (Project, error) {
	// generate some fields
	slug := slugify.Slugify(project.Title)
	now := time.Now()

	//
	p := Project{
		Name:     slug,
		Title:    project.Title,
		Content:  project.Content,
		UserName: project.UserName,
		Inserted: now,
		Updated:  now,
	}

	err := db.Update(func(tx *bolt.Tx) error {
		location := "user." + p.UserName + ".project"
		return rod.PutJson(tx, location, slug, p)
	})

	return p, err
}
