package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/ssh"
	"net/http"
)

type HandlerWithDBConnection struct {
	db *sql.DB
}

func startHTTPServer(bind string, domain string, hostPubkey ssh.PublicKey, db *sql.DB) {
	r := mux.NewRouter()

	r.Handle("/signin/{token}", &SigninHandler{db: db}).Methods("GET")
	r.Handle("/signout", &SignoutHandler{db: db}).Methods("POST")
	r.Handle("/delete-account", &DeleteAccountHandler{db: db}).Methods("POST")

	homePaths := []string{"/", "/signin", "/throwaway", "/fingerprint"}
	for _, p := range homePaths {
		r.Handle(p, &HomeHandler{db: db, hostPubkey: hostPubkey, domain: domain}).Methods("GET")
	}

	r.Handle("/pins.csv", &PinsHandler{db: db}).Methods("GET")
	r.Handle("/pin", &UpdatePinHandler{db: db}).Methods("POST")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("public")))

	http.Handle("/", r)

	go func() {
		err := http.ListenAndServe(bind, nil)
		if err != nil {
			fmt.Println(err)
		}
	}()
}
