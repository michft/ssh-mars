package main

import (
	"fmt"
	"net/http"
)

type DeleteAccountHandler HandlerWithDBConnection

func (hd *DeleteAccountHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db

	userId, _, csrfToken, signedIn, err := userIdFromSession(request, db)
	if err != nil {
		fmt.Println("reading session cookie:", err)
		clearSessionCookie(resp)
	}

	if !signedIn {
		fmt.Println("deleting account: not signed in")
		http.Error(resp, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if request.PostFormValue("csrf_token") != csrfToken {
		fmt.Println("invalid csrf token")
		http.Error(resp, "Unauthorized", http.StatusUnauthorized)
		return
	}

	_, err = db.Exec("begin transaction; delete from users where user_id = ?; delete from sessions where user_id = ?; commit;", userId, userId)
	if err != nil {
		fmt.Println("deleting user:", err)
		http.Error(resp, "Internal server error", http.StatusInternalServerError)
		return
	}

	clearSessionCookie(resp)
	http.Redirect(resp, request, "/", http.StatusSeeOther)
}
