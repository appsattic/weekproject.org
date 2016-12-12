package rod

// Rod is a simple way to put/get values to/from a BoltDB (https://github.com/boltdb/bolt) store. It can deal with
// deep-bucket hierarchies easily and is therefore a rod straight to the value you want.
//
// Whilst this package won't solve all of your problems or use-cases, it does make a few things simple and is used
// successfully in https://publish.li/ and https://weekproject.com/ and various other applications.

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/boltdb/bolt"
)

var (
	ErrLocationMustHaveAtLeastOneBucket = errors.New("location must specify at least one bucket")
)

// Put will find your bucket location and put your value into the key specified. The location is specified as a
// hierarchy of bucket names such as "users", "users.chilts", or "users.chilts.posts" and will be split on the period
// for each bucket name.
//
// At every bucket specified in the location, CreateBucketIfNotExists() is called to make sure it exists. If any of these
// fail, an error is returned.
//
// Once the final bucket is found, the value is put using the key.
//
// Example:
//
//    rod.Put(tx, "social", "twitter-123456", []byte("chilts"))
//    rod.Put(tx, "users.chilts", "email", []byte("andychilton@gmail.com"))
//    rod.Put(tx, "users.chilts.posts", "hello-world", []byte("Hello, World!"))
//
// The location must have at least one bucket ("" is not allowed), and the key must also be a non-empty string. The
// transaction must be a writeable one otherwise an error is returned.
func Put(tx *bolt.Tx, location, key string, value []byte) error {
	// split the 'bucket' on '.'
	buckets := strings.Split(location, ".")

	if len(buckets) < 1 {
		return ErrLocationMustHaveAtLeastOneBucket
	}

	// get the first bucket
	b, errCreateTopLevel := tx.CreateBucketIfNotExists([]byte(buckets[0]))
	if errCreateTopLevel != nil {
		return errCreateTopLevel
	}

	// now, only loop through if we have more than 2
	if len(buckets) > 1 {
		for _, name := range buckets[1:] {
			var err error
			b, err = b.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return err
			}
		}
	}

	return b.Put([]byte(key), value)
}

// PutJson() calls json.Marshal() to serialise the value into []byte and calls rod.Put() with the result.
func PutJson(tx *bolt.Tx, location, key string, v interface{}) error {
	// now put this value in this key
	value, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return Put(tx, location, key, value)
}
