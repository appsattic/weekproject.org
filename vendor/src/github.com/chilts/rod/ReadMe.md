# rod #

Rod is a simple way to put and get values to/from a [BoltDB](https://github.com/boltdb/bolt) store. It can deal with
deep-hierarchies easily and is therefore a rod straight to the value you want.

## Example ##

```go
user := User{
    Name: "chilts",
    Email: "andychilton@gmail.com",
    Logins: 1,
    Inserted: time.Now(),
}

db.Update(func(tx *bolt.TX) error {
    return rod.PutJson(tx, "users.chilts", "chilts", user)
})

```

## Author ##

By [Andrew Chilton](https://chilts.org/), [@twitter](https://twitter.com/andychilton).

For [AppsAttic](https://appsattic.com/), [@AppsAttic](https://twitter.com/AppsAttic).

## License ##

MIT.

(Ends)
