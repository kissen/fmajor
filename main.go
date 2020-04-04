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
	router.HandleFunc("/favicon.ico", GetFavicon).Methods("GET")
	router.HandleFunc("/files/{file_id:.+}", GetFile).Methods("GET")
	router.HandleFunc("/static/{resource_id:.+}", GetStatic).Methods("GET")
	router.HandleFunc("/submit", PostSubmit).Methods("POST")
	router.HandleFunc("/delete", PostDelete).Methods("POST")

	router.NotFoundHandler = Error(http.StatusNotFound, "")
	router.MethodNotAllowedHandler = Error(http.StatusMethodNotAllowed, "")

	addr := GetConfig().ListenAddress
	log.Printf(`listening on addr="%v"`, addr)

	server := http.Server{
		Addr:    addr,
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
