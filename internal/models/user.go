package models

type User struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	Password string `json:"-"` // не возвращаем в JSON
}
