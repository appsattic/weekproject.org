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
	// ErrLocationMustHaveAtLeastOneBucket is returned if any location given hasn't got anything in it, ie. it is
	// empty.
	ErrLocationMustHaveAtLeastOneBucket = errors.New("location must specify at least one bucket")

	// ErrInvalidLocationBucket is returned if any location is blank, ie. you specified something like "user..field".
	ErrInvalidLocationBucket = errors.New("invalid location bucket")

	// ErrKeyNotProvided is returned if key was not specified, ie. it is empty.
	ErrKeyNotProvided = errors.New("key must be specified")
)

// Put will find your bucket location and put your value into the key specified. The location is specified as a
// hierarchy of bucket names such as "users", "users.chilts", or "users.chilts.posts" and will be split on the period
// for each bucket name.
//
// At every bucket specified in the location, CreateBucketIfNotExists() is called to make sure it exists. If any of these
// fail, the error is returned.
//
// Once the final bucket is found, the value is put using the key.
//
// Examples:
//
//    rod.Put(tx, "social", "twitter-123456", []byte("chilts"))
//    rod.Put(tx, "users.chilts", "email", []byte("andychilton@gmail.com"))
//    rod.Put(tx, "users.chilts.posts", "hello-world", []byte("Hello, World!"))
//
// The location must have at least one bucket ("" is not allowed), and the key must also be a non-empty string. The
// transaction must be a writeable one otherwise an error is returned.
func Put(tx *bolt.Tx, location, key string, value []byte) error {
	if location == "" {
		return ErrLocationMustHaveAtLeastOneBucket
	}
	if key == "" {
		return ErrKeyNotProvided
	}

	// split the 'bucket' on '.'
	buckets := strings.Split(location, ".")
	if buckets[0] == "" {
		return ErrInvalidLocationBucket
	}

	// get the first bucket
	b, errCreateTopLevel := tx.CreateBucketIfNotExists([]byte(buckets[0]))
	if errCreateTopLevel != nil {
		return errCreateTopLevel
	}

	// now, only loop through if we have more than 2
	if len(buckets) > 1 {
		for _, name := range buckets[1:] {
			if name == "" {
				return ErrInvalidLocationBucket
			}
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

// Get() will fetch the raw bytes from the BoltDB. If any bucket doesn't exist it will return nil. If the key doesn't
// exist it will also return nil.
//
// Error returned from this function are:
// * ErrLocationMustHaveAtLeastOneBucket if no location was specified
// * ErrKeyNotProvided if no key was specified
func Get(tx *bolt.Tx, location, key string) ([]byte, error) {
	b, err := GetBucket(tx, location)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}

	if key == "" {
		return nil, ErrKeyNotProvided
	}

	// get this key
	return b.Get([]byte(key)), nil
}

// GetJson() calls rod.Get() and then json.Unmarshal() with the result to deserialise the value into interface{}. If
// any bucket doesn't exist we just return nil with nothing placed into v. The same if the key doesn't exist.
func GetJson(tx *bolt.Tx, location, key string, v interface{}) error {
	// get this key
	raw, err := Get(tx, location, key)
	if err != nil {
		return err
	}
	if raw == nil {
		// no key exists
		return nil
	}

	// decode to the v interface{}
	return json.Unmarshal(raw, &v)
}

// GetBucket returns this nested bucket from the store.
func GetBucket(tx *bolt.Tx, location string) (*bolt.Bucket, error) {
	if location == "" {
		return nil, ErrLocationMustHaveAtLeastOneBucket
	}

	// split the 'bucket' on '.'
	buckets := strings.Split(location, ".")
	if buckets[0] == "" {
		return nil, ErrInvalidLocationBucket
	}

	// get the first bucket
	b := tx.Bucket([]byte(buckets[0]))
	if b == nil {
		return nil, nil
	}

	// loop through if we have more than 2
	if len(buckets) > 1 {
		for _, name := range buckets[1:] {
			if name == "" {
				return nil, ErrInvalidLocationBucket
			}
			b = b.Bucket([]byte(name))
			if b == nil {
				return nil, nil
			}
		}
	}

	return b, nil
}
