package main

import (
	"net/http"
)

func HowHandler(resp http.ResponseWriter, request *http.Request) {
	http.ServeFile(resp, request, "public/how.html")
}
