package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Machiel/slugify"
	"github.com/boltdb/bolt"
)

var (
	ErrLocationMustHaveOneBucket = errors.New("location must specify at least one bucket")
)

// put() takes a location and puts the value into it. The location is of the form 'bucket.key',
// 'bucket1.bucket2.key'. Top level keys such as 'key' are not allowed. We try and CreateBucketIfNotExists() for all
// buckets, so if any fail, then the whole thing fails. This might happen if you have a value at an intermediate bucket
// instead of a bucket.
func put(tx *bolt.Tx, location, key string, v interface{}) error {
	fmt.Printf("put(): entry\n")
	fmt.Printf("* location = %v\n", location)
	fmt.Printf("* key = %v\n", key)
	defer fmt.Printf("put(): exit\n")

	// split the 'bucket' on '.'
	// Example : https://play.golang.org/p/fcU47SFSL6
	buckets := strings.Split(location, ".")

	if len(buckets) < 1 {
		return ErrLocationMustHaveOneBucket
	}

	// get the first bucket
	fmt.Printf("Getting top-level bucket %s\n", buckets[0])
	b, errCreateTopLevel := tx.CreateBucketIfNotExists([]byte(buckets[0]))
	if errCreateTopLevel != nil {
		fmt.Printf("Error creating top level bucket %s : %v\n", buckets[0], errCreateTopLevel)
		return errCreateTopLevel
	}

	// now, only loop through if we have more than 2
	if len(buckets) > 1 {
		for _, name := range buckets[1:] {
			fmt.Printf("Getting sub-bucket bucket %s\n", name)
			var err error
			b, err = b.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				fmt.Printf("Error creating sub-bucket %s : %v\n", name, err)
				return err
			}
		}
	}

	// now put this value in this key
	bytes, errMarshal := json.Marshal(v)
	if errMarshal != nil {
		fmt.Printf("Error calling json.Marshal() : %v\n", errMarshal)
		return errMarshal
	}
	return b.Put([]byte(key), bytes)
}

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
		return put(tx, location, social.Id, s)
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
		return put(tx, location, "meta", u)
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
		return put(tx, location, slug, p)
	})

	return p, err
}
