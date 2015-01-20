package main

import (
	"crypto/subtle"
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

	if len(signinToken) != signinIdLength+signinSecretLength {
		fmt.Println("invalid length for signin token:", len(signinToken))
		http.Error(resp, "Invalid signin token", http.StatusBadRequest)
		return
	}

	signinId := signinToken[:signinIdLength]
	providedSigninSecret := signinToken[signinIdLength:]

	var pubkey []byte
	var signinSecret string
	err := db.QueryRow("select signin_secret, pubkey from signin_requests where signin_id = ?", signinId).Scan(&signinSecret, &pubkey)

	if err == sql.ErrNoRows {
		fmt.Printf("no signin request for token: %q\n", signinToken)
		http.Error(resp, "Invalid signin token", http.StatusBadRequest)
		return
	} else if err != nil {
		fmt.Println("retrieving signin token:", err)
		http.Error(resp, "Invalid signin token", http.StatusBadRequest)
		return
	}

	if subtle.ConstantTimeCompare([]byte(providedSigninSecret), []byte(signinSecret)) != 1 {
		fmt.Printf("incorrect signin token: %q\n", signinToken)
		http.Error(resp, "Invalid signin token", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("delete from signin_requests where signin_id = ?", signinId)
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

	sessionId, err := randomToken(sessionIdLength)
	if err != nil {
		fmt.Println("generating random token:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}

	sessionSecret, err := randomToken(sessionSecretLength)
	if err != nil {
		fmt.Println("generating random token:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}

	csrfToken, err := randomToken(csrfTokenLength)
	if err != nil {
		fmt.Println("generating random token:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}
	_, err = db.Exec("insert into sessions (user_id, last_active, session_id, session_secret, csrf_token) values (?, ?, ?, ?, ?)", userId, timestamp, sessionId, sessionSecret, csrfToken)
	if err != nil {
		fmt.Println("creating user session:", err)
		http.Error(resp, "There was an error signing you in", http.StatusInternalServerError)
		return
	}

	setSessionCookie(resp, sessionId, sessionSecret)

	targetPath := "/"
	http.Redirect(resp, request, targetPath, http.StatusSeeOther)
}

func setSessionCookie(resp http.ResponseWriter, id, secret string) {
	timestamp := time.Now().Unix()
	cookie := &http.Cookie{
		Name:     "session",
		Value:    fmt.Sprint(id, secret),
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
