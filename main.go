package main

import (
	"database/sql"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/duarteocarmo/hyggenews/parser"
	"github.com/duarteocarmo/hyggenews/types"
)

// func NewFeedParser(config string) {
// 	f := types.FeedParser{}
// 	parser.Load(&f, config)
// 	parser.Parse(&f)
// }

type server struct {
	router *mux.Router
	parser types.FeedParser
	db     types.DB
}

func NewServer() {
	s := &server{
		router: mux.NewRouter(),
		parser: types.FeedParser{},
	}
	parser.Load(&s.parser, "config.json")
	s.db = s.parser.Config.Database
	// parser.Parse(&s.parser)
	s.routes()
	log.Fatal(http.ListenAndServe(":8080", s.router))
}

func (s *server) routes() {
	s.router.HandleFunc("/", s.handleIndex())
	s.router.HandleFunc("/{id:[a-zA-Z0-9]+}", s.handleArticle())
}

func getArticles(s *server) []types.Article {
	db, err := sql.Open("sqlite3", s.db.Conn)
	if err != nil {
		log.Println("Error opening database: ", err)
		return nil
	}
	defer db.Close() // Don't forget to close the database connection

	var articles []types.Article

	query := `
		SELECT id, title, link, date, content, source, TranslatedContent, TranslatedTitle 
		FROM articles
		WHERE date >= date('now') AND date < date('now', '+1 day')
		ORDER BY date DESC;
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Error querying database: ", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var a types.Article
		err := rows.Scan(&a.ID, &a.Title, &a.Link, &a.Date, &a.Content, &a.Source, &a.TranslatedContent, &a.TranslatedTitle)
		if err != nil {
			log.Println("Error scanning database result: ", err)
			continue
		}
		articles = append(articles, a)
	}

	if err = rows.Err(); err != nil {
		log.Println("Error with database rows: ", err)
		return nil
	}

	return articles
}
func getArticleByID(s *server, id string) (*types.Article, error) {
	db, err := sql.Open("sqlite3", s.db.Conn)
	if err != nil {
		log.Println("Error opening database: ", err)
		return nil, err
	}
	defer db.Close()

	var article types.Article

	query := `SELECT id, title, link, date, content, source, TranslatedContent, TranslatedTitle FROM articles WHERE id = ?`
	row := db.QueryRow(query, id)

	err = row.Scan(&article.ID, &article.Title, &article.Link, &article.Date, &article.Content, &article.Source, &article.TranslatedContent, &article.TranslatedTitle)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		log.Println("Error querying database by ID: ", err)
		return nil, err
	}

	return &article, nil
}

func (s *server) handleIndex() http.HandlerFunc {

	type Page struct {
		Articles []types.Article
		Today    string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		today := time.Now().Format("Monday, January 2, 2006")
		articles := getArticles(s)

		t, err := template.ParseFiles("index.html")
		if err != nil {
			log.Println(err)
		}
		t.Execute(w, Page{Articles: articles, Today: today})
	}
}
func (s *server) handleArticle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]

		article, err := getArticleByID(s, id)
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

		t, err := template.ParseFiles("article.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		t.Execute(w, article)
	}
}

func main() {
	// NewFeedParser("config.json")
	NewServer()
}
