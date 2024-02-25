package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/duarteocarmo/hyggenews/db"
	"github.com/duarteocarmo/hyggenews/parser"
	"github.com/duarteocarmo/hyggenews/types"
)

func handlePage(s *types.Server, page string) http.HandlerFunc {

	type Page struct {
		Articles []types.Article
		Today    string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		today := time.Now().Format("Monday, January 2, 2006")
		articles := db.GetArticles(s)

		p := fmt.Sprintf("templates/%s.html", page)
		t, err := template.ParseFiles(p, "templates/partials/footer.html")
		if err != nil {
			log.Println(err)
		}
		t.Execute(w, Page{Articles: articles, Today: today})
	}
}

func handleArticle(s *types.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]

		article, err := db.GetArticleByID(s, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if article == nil {
			http.NotFound(w, r)
			return
		}

		plaintext := article.TranslatedContent
		lines := strings.Split(plaintext, "\n")
		htmlText := "<p>" + strings.Join(lines, "</p><p>") + "</p>"
		article.HTMLContent = template.HTML(htmlText)

		t, err := template.ParseFiles("templates/article.html", "templates/partials/footer.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		t.Execute(w, article)
	}
}

func handleCategory(s *types.Server) http.HandlerFunc {
	type CPage struct {
		Articles []types.Article
		Today    string
		Category string
	}

	today := time.Now().Format("Monday, January 2, 2006")
	page := "category"
	categories, err := db.GetCategories(s)
	if err != nil {
		log.Println(err)
		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		c := vars["category"]

		if !slices.Contains(categories, c) {
			http.NotFound(w, r)
			return
		}

		articles, err := db.GetArticlesByCategory(s, c)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		p := fmt.Sprintf("templates/%s.html", page)
		t, err := template.ParseFiles(p, "templates/partials/footer.html")
		if err != nil {
			log.Println(err)
		}
		t.Execute(w, CPage{Articles: articles, Today: today, Category: c})

	}
}

func NewServer() {
	s := &types.Server{
		Router: mux.NewRouter(),
		Parser: types.FeedParser{},
	}

	parser.Load(&s.Parser, "config/config.json")

	s.Db = s.Parser.Config.Database

	s.Router.HandleFunc("/", handlePage(s, "index"))
	s.Router.HandleFunc("/about", handlePage(s, "about"))
	s.Router.HandleFunc("/{id:[a-zA-Z0-9]+}", handleArticle(s))
	s.Router.HandleFunc("/category/{category}", handleCategory(s))
	s.Router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	// handle article category

	go func() {
		for {
			parser.Parse(&s.Parser)
			time.Sleep(1 * time.Hour)
		}
	}()
	log.Fatal(http.ListenAndServe(":8080", s.Router))
}

func main() {
	NewServer()
}
