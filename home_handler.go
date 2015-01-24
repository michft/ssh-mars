package main

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"html/template"
	"net"
	"net/http"
	"path"
	"time"
)

var (
	ErrNotSignedIn    = errors.New("not signed in")
	ErrInvalidSession = errors.New("invalid session token")
)

type UserSession struct {
	userId        int
	sessionId     string
	sessionSecret string
	csrfToken     string
}

type HomeHandler struct {
	db           *sql.DB
	sshAdvertise string
	assetsDir    string
	hostPubkey   ssh.PublicKey
}

type HomeContext struct {
	IntroPage          bool
	SigninPage         bool
	ThrowawayPage      bool
	FingerprintPage    bool
	SignedIn           bool
	UserId             int
	Fingerprint        string
	SigninToken        string
	CSRFToken          string
	SSHHost            string
	SSHPort            string
	SSHPortNonStandard bool
	HostFingerprint1   string
	HostFingerprint2   string
}

func (hd *HomeHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db

	t, err := template.ParseFiles(path.Join(hd.assetsDir, "index.html"))
	if err != nil {
		fmt.Println("parsing HTML template:", err)
		http.Error(resp, "Internal server error", http.StatusInternalServerError)
		return
	}

	session, err := sessionFromRequest(request, db)
	if err == ErrInvalidSession {
		fmt.Println(err)
		clearSessionCookie(resp)
	}
	signedIn := err == nil

	var pubkey []byte
	var fingerprint string
	var signinId, signinSecret, csrfToken string

	if signedIn {
		err = db.QueryRow("select pubkey from users where user_id = ?", session.userId).Scan(&pubkey)
		if err != nil {
			fmt.Println("retrieving pubkey:", err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
		fingerprint = pubkeyFingerprintMD5(pubkey)
		keepSessionAlive(resp, db, session)
	} else {
		signinId, err = randomToken(signinIdLength)
		if err != nil {
			fmt.Println(err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		signinSecret, err = randomToken(signinSecretLength)
		if err != nil {
			fmt.Println(err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		csrfToken, err = randomToken(csrfTokenLength)
		if err != nil {
			fmt.Println(err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}

		timestamp := time.Now().Unix()
		_, err = db.Exec("insert into signin_requests (created_at, signin_id, signin_secret, csrf_token) values (?, ?, ?, ?)", timestamp, signinId, signinSecret, csrfToken)
		if err != nil {
			fmt.Println(err)
			http.Error(resp, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	hostFingerprint := pubkeyFingerprintMD5(hd.hostPubkey.Marshal())
	var hostFingerprint1, hostFingerprint2 string
	if len(hostFingerprint) == 47 {
		hostFingerprint1 = hostFingerprint[0:23]
		hostFingerprint2 = hostFingerprint[24:47]
	}

	sshHost, sshPort, err := net.SplitHostPort(hd.sshAdvertise)
	if err != nil {
		fmt.Println("splitting SSH host and port:", err)
		http.Error(resp, "Internal server error", http.StatusInternalServerError)
		return
	}

	path := request.URL.Path
	context := HomeContext{
		IntroPage:          path == "/" && !signedIn,
		SigninPage:         path == "/signin",
		ThrowawayPage:      path == "/throwaway",
		FingerprintPage:    path == "/fingerprint",
		SignedIn:           signedIn,
		UserId:             session.userId,
		Fingerprint:        fingerprint,
		CSRFToken:          session.csrfToken,
		SSHHost:            sshHost,
		SSHPort:            sshPort,
		SSHPortNonStandard: sshPort != "22",
		HostFingerprint1:   hostFingerprint1,
		HostFingerprint2:   hostFingerprint2,
	}

	if !signedIn {
		context.SigninToken = fmt.Sprint(signinId, signinSecret)
		context.CSRFToken = csrfToken
	}

	t.Execute(resp, context)
}

func sessionFromRequest(request *http.Request, db *sql.DB) (UserSession, error) {
	session := UserSession{}

	cookie, err := request.Cookie("session")
	if err == http.ErrNoCookie {
		return session, ErrNotSignedIn
	} else if err != nil {
		return session, ErrInvalidSession
	}

	sessionToken := cookie.Value

	if len(sessionToken) != sessionIdLength+sessionSecretLength {
		return session, ErrInvalidSession
	}

	session.sessionId = sessionToken[:sessionIdLength]
	providedSessionSecret := sessionToken[sessionIdLength:]

	err = db.QueryRow("select user_id, session_secret, csrf_token from sessions where session_id = ?", session.sessionId).Scan(&session.userId, &session.sessionSecret, &session.csrfToken)
	if err != nil {
		return session, ErrInvalidSession
	}

	if subtle.ConstantTimeCompare([]byte(providedSessionSecret), []byte(session.sessionSecret)) != 1 {
		return session, ErrInvalidSession
	}

	return session, nil
}

func keepSessionAlive(resp http.ResponseWriter, db *sql.DB, session UserSession) {
	timestamp := time.Now().Unix()
	db.Exec("update sessions set last_active = ? where session_id = ?", timestamp, session.sessionId)
	setSessionCookie(resp, session.sessionId, session.sessionSecret)
}
