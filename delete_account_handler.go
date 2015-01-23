package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
)

type DeleteAccountHandler HandlerWithDBConnection

func (hd *DeleteAccountHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db

	session, err := sessionFromRequest(request, db)
	if err == ErrInvalidSession {
		fmt.Println(err)
		clearSessionCookie(resp)
	}
	signedIn := err == nil

	if !signedIn {
		fmt.Println("deleting account: not signed in")
		http.Error(resp, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if subtle.ConstantTimeCompare([]byte(request.PostFormValue("csrf_token")), []byte(session.csrfToken)) != 1 {
		fmt.Println("invalid csrf token")
		http.Error(resp, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	// TODO: use idiomatic go transactions
	_, err = db.Exec("begin transaction; delete from users where user_id = ?; delete from sessions where user_id = ?; commit;", session.userId, session.userId)
	if err != nil {
		fmt.Println("deleting user:", err)
		http.Error(resp, "Internal server error", http.StatusInternalServerError)
		return
	}

	clearSessionCookie(resp)
	http.Redirect(resp, request, "/", http.StatusSeeOther)
}
