package main

import (
	"crypto/subtle"
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

	userId, sessionId, sessionSecret, csrfToken, signedIn, err := userIdFromSession(request, db)
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

		keepSessionAlive(resp, db, sessionId, sessionSecret)
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

func userIdFromSession(request *http.Request, db *sql.DB) (int, string, string, string, bool, error) {
	cookie, err := request.Cookie("session")
	if err == http.ErrNoCookie {
		return 0, "", "", "", false, nil
	} else if err != nil {
		return 0, "", "", "", false, err
	}

	sessionToken := cookie.Value

	if len(sessionToken) != sessionIdLength+sessionSecretLength {
		return 0, "", "", "", false, fmt.Errorf("invalid session token")
	}

	sessionId := sessionToken[:sessionIdLength]
	providedSessionSecret := sessionToken[sessionIdLength:]

	var userId int
	var sessionSecret, csrfToken string
	err = db.QueryRow("select user_id, session_secret, csrf_token from sessions where session_id = ?", sessionId).Scan(&userId, &sessionSecret, &csrfToken)
	if err != nil {
		return 0, "", "", "", false, err
	}

	if subtle.ConstantTimeCompare([]byte(providedSessionSecret), []byte(sessionSecret)) != 1 {
		return 0, "", "", "", false, fmt.Errorf("invalid session token")
	}

	return userId, sessionId, sessionSecret, csrfToken, true, nil
}

func keepSessionAlive(resp http.ResponseWriter, db *sql.DB, id, secret string) {
	timestamp := time.Now().Unix()
	db.Exec("update sessions set last_active = ? where session_id = ?", timestamp, id)
	setSessionCookie(resp, id, secret)
}
