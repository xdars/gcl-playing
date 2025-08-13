package database

import (
	"fmt"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type User struct {
	Token string
	RToken string
}

func GetUser(email string) (*User, error) {
	db, err := sql.Open("sqlite3", "../data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("select token, refresh_token from users where email = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var token string
	var rtoken string
	err = stmt.QueryRow(email).Scan(&token, &rtoken)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if len(token) == 0 {
		log.Println("token is empty. = used does not exist")
	}
	user := &User{token, rtoken}
	return user, nil

}