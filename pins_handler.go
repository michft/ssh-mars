package main

import (
	"fmt"
	"net/http"
)

type PinsHandler HandlerWithDBConnection

func (hd *PinsHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	db := hd.db

	resp.Header().Add("Content-Type", "text/csv; charset=utf-8")
	resp.Header().Add("Cache-Control", "public")

	rows, err := db.Query("select pubkey, lat, lon from users where lat is not null and lon is not null")

	defer rows.Close()
	for rows.Next() {
		var lat, lon float64
		var pubkey []byte
		err = rows.Scan(&pubkey, &lat, &lon)

		fingerprint := pubkeyFingerprintMD5(pubkey)
		fmt.Fprintf(resp, "%s,%.6f,%.6f\n", fingerprint, lat, lon)
	}
	err = rows.Err()

	if err != nil {
		fmt.Println("listing pins:", err)
		http.Error(resp, "Internal server error", http.StatusInternalServerError)
		return
	}
}
