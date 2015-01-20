package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type SigninHandler HandlerWithDBConnection

func (hd *SigninHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db
	vars := mux.Vars(request)
	signinToken := vars["token"]

	var pubkey []byte
	// TODO: check signin not exipred
	// TODO: check token length
	err := db.QueryRow("select pubkey from signin_requests where signin_token = ?", signinToken).Scan(&pubkey)

	if err == sql.ErrNoRows {
		fmt.Printf("no signin request for token: %q\n", signinToken)
		http.Error(resp, "Invalid signin token", http.StatusBadRequest)
		return
	} else if err != nil {
		fmt.Println("retrieving signin token:", err)
		http.Error(resp, "Invalid signin token", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("delete from signin_requests where signin_token = ?", signinToken)
	if err != nil {
		fmt.Println("deleting signin token:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}

	var userId int
	err = db.QueryRow("select user_id from users where pubkey = ?", pubkey).Scan(&userId)
	if err == sql.ErrNoRows {
		userId, err = createUser(pubkey, db)
		if err != nil {
			fmt.Println("creating new user:", err)
			http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		fmt.Println("retrieving user ID:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().Unix()

	sessionToken, err := randomToken(50)
	if err != nil {
		fmt.Println("generating random token:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}

	csrfToken, err := randomToken(50)
	if err != nil {
		fmt.Println("generating random token:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}
	_, err = db.Exec("insert into sessions (user_id, last_active, session_token, csrf_token) values (?, ?, ?, ?)", userId, timestamp, sessionToken, csrfToken)
	if err != nil {
		fmt.Println("creating user session:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}

	setSessionCookie(resp, sessionToken)

	targetPath := "/"
	http.Redirect(resp, request, targetPath, http.StatusSeeOther)
}

func setSessionCookie(resp http.ResponseWriter, sessionToken string) {
	timestamp := time.Now().Unix()
	cookie := &http.Cookie{
		Name:     "session",
		Value:    sessionToken,
		Path:     "/",
		Expires:  time.Unix(timestamp+(3600*24*20), 0),
		HttpOnly: true,
	}
	http.SetCookie(resp, cookie)
}

func createUser(pubkey []byte, db *sql.DB) (int, error) {
	res, err := db.Exec("insert into users (pubkey) values (?)", pubkey)
	if err != nil {
		return 0, err
	}

	userId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(userId), nil
}
