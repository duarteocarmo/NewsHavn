package types

import (
	"github.com/gorilla/mux"
	"html/template"
	"time"
)

type Server struct {
	Router *mux.Router
	Parser FeedParser
	Db     DB
}

type FeedParser struct {
	Config Config
}

type Config struct {
	Sources  []Source
	Database DB
}

type DB struct {
	Conn string
}

type Source struct {
	Name       string
	Feed       string
	Getwebsite bool
	Contentkey string
}

type Article struct {
	ID                string
	Title             string
	Link              string
	Date              time.Time
	Content           string
	Source            string
	TranslatedContent string
	TranslatedTitle   string
	HTMLContent       template.HTML
}
