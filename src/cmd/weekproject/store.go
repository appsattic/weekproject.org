package main

import (
	"errors"
	"time"

	"github.com/boltdb/bolt"
	"github.com/chilts/rod"
)

var (
	ErrLocationMustHaveOneBucket = errors.New("location must specify at least one bucket")
)

var format = "2006-01-02T15:04:05Z"

func SocialIns(db *bolt.DB, social Social) (Social, error) {
	// generate some fields
	now := time.Now().UTC()

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
	now := time.Now().UTC()

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

// InsProject takes a project and it into the store. It doesn't set or manipulate any fields on the project prior to
// insert. It fails if this project already exists (under this user).
//
// We only use the Title, Content, and UserName fields. The rest are generated.
func InsProject(db *bolt.DB, p Project) error {
	return db.Update(func(tx *bolt.Tx) error {
		location := "user." + p.UserName + ".project." + p.Name
		return rod.PutJson(tx, location, "meta", p)
	})
}

// GetProject
func GetProject(db *bolt.DB, userName, projectName string) (Project, error) {
	p := Project{}

	err := db.View(func(tx *bolt.Tx) error {
		return rod.GetJson(tx, "user."+userName+".project."+projectName, "meta", &p)
	})

	return p, err
}

// ProjectSel returns a splice of projects for this userName.
func SelProject(db *bolt.DB, userName string) ([]*Project, error) {
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

		// loop through all project buckets
		c := b.Cursor()
		for name, _ := c.First(); name != nil; name, _ = c.Next() {
			// get this project
			p := Project{}
			err := rod.GetJson(tx, "user."+userName+".project."+string(name), "meta", &p)
			if err != nil {
				return nil
			}
			projects = append(projects, &p)
		}

		return nil
	})

	return projects, err
}

// InsUpdate takes an update and a project and puts it into the store. It doesn't set or manipulate any fields on the
// project prior to insert. It uses an id based on the u.Inserted time.
func InsUpdate(db *bolt.DB, p Project, u Update) error {
	id := u.Inserted.Format(format)

	return db.Update(func(tx *bolt.Tx) error {
		location := "user." + p.UserName + ".project." + p.Name + ".update"
		return rod.PutJson(tx, location, id, u)
	})
}
