package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	router := mux.NewRouter()
	router.HandleFunc("/", GetIndex).Methods("GET")
	router.HandleFunc("/static/{resourceid:.+}", GetStatic).Methods("GET")
	router.HandleFunc("/files/{fileid:.+}", GetFile).Methods("GET")
	router.HandleFunc("/submit", PostSubmit).Methods("POST")

	router.NotFoundHandler = Error(http.StatusNotFound, "")
	router.MethodNotAllowedHandler = Error(http.StatusMethodNotAllowed, "")

	if err := http.ListenAndServe(GetConfig().ListenAddress, router); err != nil {
		log.Fatal(err)
	}
}
