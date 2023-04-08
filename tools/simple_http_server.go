package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var count = 0

func handleHttp(h http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	b, _ := io.ReadAll(req.Body)
	fmt.Println(string(b))
	response := strings.Repeat("a", 1000)
	fmt.Fprintf(h, response)
}

func main() {
	http.HandleFunc("/", handleHttp)
	http.ListenAndServe("127.0.0.1:8989", nil)
}
