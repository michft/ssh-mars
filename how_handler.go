package main

import (
	"net/http"
	"path"
)

type HowHandler struct {
	assetsDir string
}

func (hd *HowHandler) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	http.ServeFile(resp, request, path.Join(hd.assetsDir, "how.html"))
}
