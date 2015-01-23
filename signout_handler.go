package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
)

type SignoutHandler HandlerWithDBConnection

func (hd *SignoutHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db

	session, err := sessionFromRequest(request, db)
	if err == ErrInvalidSession {
		fmt.Println(err)
		clearSessionCookie(resp)
	}
	signedIn := err == nil

	if !signedIn {
		fmt.Println("not signed in")
		http.Error(resp, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if subtle.ConstantTimeCompare([]byte(request.PostFormValue("csrf_token")), []byte(session.csrfToken)) != 1 {
		fmt.Println("invalid csrf token")
		http.Error(resp, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	clearSessionCookie(resp)

	db.Exec("delete from sessions where session_id = ?", session.sessionId)

	http.Redirect(resp, request, "/", http.StatusSeeOther)
}

func clearSessionCookie(resp http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "session",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(resp, cookie)
}
