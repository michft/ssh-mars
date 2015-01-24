package main

import (
	"crypto/subtle"
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	//"time"
)

type SigninNotifyHandler struct {
	db     *sql.DB
	broker *Broker
}

func (hd *SigninNotifyHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db
	broker := hd.broker

	flusher, ok := resp.(http.Flusher)
	if !ok {
		fmt.Printf("http streaming unsupported")
		http.Error(resp, "Internal server error", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(request)
	signinToken := vars["token"]

	if len(signinToken) != signinIdLength+signinSecretLength {
		fmt.Println("invalid length for signin token:", len(signinToken))
		http.Error(resp, "Invalid signin token", http.StatusBadRequest)
		return
	}

	signinId := signinToken[:signinIdLength]
	providedSigninSecret := signinToken[signinIdLength:]

	var signinSecret string
	var pubkey []byte
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

	resp.Header().Add("Content-Type", "text/event-stream; charset=utf-8")
	resp.Header().Add("Cache-Control", "no-cache")
	resp.Header().Add("Connection", "keep-alive")

	messageChan := make(chan string)
	broker.newClients <- messageChan

	defer func() {
		broker.closingClients <- messageChan
	}()

	closeNotify := resp.(http.CloseNotifier).CloseNotify()
	go func() {
		<-closeNotify
		broker.closingClients <- messageChan
	}()

	if len(pubkey) != 0 {
		fmt.Fprintf(resp, "event: authenticated\ndata:\n\n")
		flusher.Flush()
	}

	for {
		notifiedSigninToken := <-messageChan
		if notifiedSigninToken == signinToken {
			fmt.Fprintf(resp, "event: authenticated\ndata:\n\n")
			flusher.Flush()
		}
	}

	//ticker := time.NewTicker(1 * time.Second)
	//defer ticker.Stop()

	//for _ = range ticker.C {
	//  fmt.Fprintf(resp, "event: keepalive\n\n")
	//  flusher.Flush()
	//}
}
