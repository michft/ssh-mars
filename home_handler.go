package main

import (
	"database/sql"
	"fmt"
	"golang.org/x/crypto/ssh"
	"html/template"
	"net/http"
	"time"
)

type HomeHandler struct {
	db         *sql.DB
	domain     string
	hostPubkey ssh.PublicKey
}

type HomeContext struct {
	IntroPage        bool
	SigninPage       bool
	ThrowawayPage    bool
	FingerprintPage  bool
	SignedIn         bool
	UserId           int
	Fingerprint      string
	CSRFToken        string
	Domain           string
	HostFingerprint1 string
	HostFingerprint2 string
}

func (hd *HomeHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db

	t, err := template.ParseFiles("public/index.html")
	if err != nil {
		fmt.Println("parsing HTML template:", err)
		http.Error(resp, "Internal server error", http.StatusInternalServerError)
		return
	}

	userId, sessionToken, csrfToken, signedIn, err := userIdFromSession(request, db)
	if err != nil {
		fmt.Println("reading session cookie:", err)
		clearSessionCookie(resp)
	}

	var pubkey []byte
	var fingerprint string

	if signedIn {
		err = db.QueryRow("select pubkey from users where user_id = ?", userId).Scan(&pubkey)
		if err != nil {
			fmt.Println("retrieving pubkey:", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
		fingerprint = pubkeyFingerprintMD5(pubkey)

		keepSessionAlive(resp, db, sessionToken)
	}

	hostFingerprint := pubkeyFingerprintMD5(hd.hostPubkey.Marshal())
	var hostFingerprint1, hostFingerprint2 string
	if len(hostFingerprint) == 47 {
		hostFingerprint1 = hostFingerprint[0:23]
		hostFingerprint2 = hostFingerprint[24:47]
	}

	path := request.URL.Path
	context := HomeContext{
		IntroPage:        path == "/" && !signedIn,
		SigninPage:       path == "/signin",
		ThrowawayPage:    path == "/throwaway",
		FingerprintPage:  path == "/fingerprint",
		SignedIn:         signedIn,
		UserId:           userId,
		Fingerprint:      fingerprint,
		CSRFToken:        csrfToken,
		Domain:           hd.domain,
		HostFingerprint1: hostFingerprint1,
		HostFingerprint2: hostFingerprint2,
	}

	t.Execute(resp, context)
}

func userIdFromSession(request *http.Request, db *sql.DB) (int, string, string, bool, error) {
	cookie, err := request.Cookie("session")
	if err == http.ErrNoCookie {
		return 0, "", "", false, nil
	} else if err != nil {
		return 0, "", "", false, err
	}

	sessionToken := cookie.Value

	var userId int
	var csrfToken string
	err = db.QueryRow("select user_id, csrf_token from sessions where session_token = ?", sessionToken).Scan(&userId, &csrfToken)
	if err != nil {
		return 0, "", "", false, err
	}

	return userId, sessionToken, csrfToken, true, nil
}

func keepSessionAlive(resp http.ResponseWriter, db *sql.DB, sessionToken string) {
	timestamp := time.Now().Unix()
	db.Exec("update sessions set last_active = ? where session_token = ?", timestamp, sessionToken)
	setSessionCookie(resp, sessionToken)
}
