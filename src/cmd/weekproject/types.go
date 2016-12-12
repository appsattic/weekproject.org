package main

import "time"

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
	Name     string // e.g. "week-project"
	Title    string // e.g. "The Week Project"
	Content  string
	UserName string // e.g. "chilts"
	Inserted time.Time
	Updated  time.Time
}
