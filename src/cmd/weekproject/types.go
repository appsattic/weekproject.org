package main

type User struct {
	Id    string // e.g. "twitter-123456"
	Name  string // e.g. "chilts" (ie. their Twitter handle)
	Title string // e.g. "Andrew Chilton"
	Email string // e.g. "andychilton@gmail.com"
}

type Project struct {
	Name     string // e.g. "week-project"
	Title    string // e.g. "The Week Project"
	Content  string
	UserName string // e.g. "chilts"
}
