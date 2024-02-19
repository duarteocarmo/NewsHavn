package types

import "html/template"

type Source struct {
	Name       string
	Feed       string
	Getwebsite bool
	Contentkey string
}

type DB struct {
	Conn string
}

type Config struct {
	Sources  []Source
	Database DB
}

type FeedParser struct {
	Config Config
}

type Article struct {
	ID                string
	Title             string
	Link              string
	Date              string
	Content           string
	Source            string
	TranslatedContent string
	TranslatedTitle   string
	HTMLContent 	  template.HTML
}
