package main

type User struct {
	Id       string // e.g. "twitter-123456"
	Username string // e.g. "chilts"
	Email    string // e.g. "andychilton@gmail.com"
}

type Project struct {
	Id       string
	Username string
	Title    string
	Content  string
}
