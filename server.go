package main

// import (
// 	"encoding/json"
// 	"log"
// 	"net/http"
//
// 	"github.com/gorilla/mux"
// 	"github.com/mmcdole/gofeed"
// )
//
// type server struct {
// 	router *mux.Router
// 	// db     *someDatabase
// 	// email  EmailSender
// }
//
// func NewServer() {
// 	s := &server{mux.NewRouter()}
// 	s.routes()
// 	log.Fatalln(http.ListenAndServe(":8080", s.router))
// }
//
// func (s *server) routes() {
// 	s.router.HandleFunc("/", s.handleIndex())
// }
//
// func (s *server) handleIndex() http.HandlerFunc {
// 	type response struct {
// 		Greeting string `json:"greeting"`
// 	}
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		greet := response{"Hello world"}
// 		js, err := json.Marshal(greet)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}
//
// 		w.Header().Set("Content-Type", "application/json")
// 		w.Write(js)
//
// 	}
// }
//
// func main() {
// 	// NewServer()
// 	NewFeedParser()
// }
