package main

import (
	"encoding/json"
	"errors"
	"time"

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

// ProjectIns takes a project and it into the store. It doesn't set or manipulate any fields on the project prior to
// insert. It fails if this project already exists (under this user).
//
// We only use the Title, Content, and UserName fields. The rest are generated.
func ProjectIns(db *bolt.DB, p Project) (Project, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		location := "user." + p.UserName + ".project"
		return rod.PutJson(tx, location, p.Name, p)
	})

	return p, err
}

// ProjectGet
func ProjectGet(db *bolt.DB, userName, projectName string) (Project, error) {
	p := Project{}

	err := db.View(func(tx *bolt.Tx) error {
		return rod.GetJson(tx, "user."+userName+".project", projectName, &p)
	})

	return p, err
}

// ProjectSel returns a splice of projects for this userName.
func ProjectSel(db *bolt.DB, userName string) ([]*Project, error) {
	projects := make([]*Project, 0)

	err := db.View(func(tx *bolt.Tx) error {
		// range over this user's projects
		b, err := rod.GetBucket(tx, "user."+userName+".project")
		if err != nil {
			return err
		}
		if b == nil {
			return nil
		}

		// loop through all posts
		c := b.Cursor()
		for name, raw := c.First(); name != nil; name, raw = c.Next() {
			// decode this post
			p := Project{}
			err := json.Unmarshal(raw, &p)
			if err != nil {
				return nil
			}
			projects = append(projects, &p)
		}

		return nil
	})

	return projects, err
}
