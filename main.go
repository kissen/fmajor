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
	router.HandleFunc("/login", GetLogin).Methods("GET")
	router.HandleFunc("/login", PostLogin).Methods("POST")
	router.HandleFunc("/logout", PostLogout).Methods("POST")
	router.HandleFunc("/favicon.ico", GetFavicon).Methods("GET")
	router.HandleFunc("/files/{file_id:.+}/{file_name:.+}", GetFile).Methods("GET")
	router.HandleFunc("/files/{file_id:.+}/{file_name:.+}", HeadFile).Methods("HEAD")
	router.HandleFunc("/f/{short_id:.+}", GetShort).Methods("GET")
	router.HandleFunc("/thumbnails/{file_id:.+}/thumbnail.jpg", GetThumbnail).Methods("GET")
	router.HandleFunc("/thumbnails/{file_id:.+}/thumbnail.jpg", GetThumbnail).Methods("HEAD")
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
