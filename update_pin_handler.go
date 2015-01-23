package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type UpdatePinHandler HandlerWithDBConnection

func (hd *UpdatePinHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db

	session, err := sessionFromRequest(request, db)
	if err == ErrInvalidSession {
		fmt.Println(err)
		clearSessionCookie(resp)
	}
	signedIn := err == nil

	if !signedIn {
		fmt.Println("updating pins: not signed in")
		http.Error(resp, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if subtle.ConstantTimeCompare([]byte(request.PostFormValue("csrf_token")), []byte(session.csrfToken)) != 1 {
		fmt.Println("invalid csrf token")
		http.Error(resp, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	keepSessionAlive(resp, db, session)

	latStr := request.PostFormValue("lat")
	lonStr := request.PostFormValue("lon")
	timestamp := time.Now().Unix()

	if latStr == "" && lonStr == "" {
		_, err = db.Exec("update users set lat = null, lon = null, pin_updated_at = null where user_id = ?", session.userId)
		if err != nil {
			fmt.Println("updating pin position:", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		var lat, lon float64
		lat, err = strconv.ParseFloat(latStr, 64)
		if err == nil {
			lon, err = strconv.ParseFloat(lonStr, 64)
		}
		if err != nil {
			fmt.Println("parsing lat/lon values:", err)
			http.Error(resp, "Bad request", http.StatusBadRequest)
			return
		}

		_, err = db.Exec("update users set lat = ?, lon = ?, pin_updated_at = ? where user_id = ?", lat, lon, timestamp, session.userId)
		if err != nil {
			fmt.Println("updating pin position:", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}
