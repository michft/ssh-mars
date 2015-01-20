package main

import (
	"net/http"
)

type SignoutHandler HandlerWithDBConnection

func (hd *SignoutHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db

	clearSessionCookie(resp)

	cookie, err := request.Cookie("session")
	if err == nil {
		sessionToken := cookie.Value
		db.Exec("delete from sessions where session_token = ?", sessionToken)
	}

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
