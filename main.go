package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mmcdole/gofeed"
	"log"
	"os"
	"reflect"

	_ "github.com/mattn/go-sqlite3"
)

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
	config Config
}
type Article struct {
	ID                string
	Title             string
	Link              string
	Date              string
	Content           string
	Source            string
	TranslatedContent string
}

func (f *FeedParser) Load(configFilePath string) {
	file, _ := os.Open(configFilePath)
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := Config{}
	err := decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	f.config = config

}

func (f *FeedParser) Parse() {
	for _, source := range f.config.Sources {
		al, err := parseSource(source)
		if err != nil {
			log.Println("Error parsing source: ", source.Name)
			continue
		}

		log.Println("Parsed ", len(al), " articles from source: ", source.Name)

		if err = insertArticles(f, al); err != nil {
			log.Println("Error inserting articles: ", err)
		}

		log.Println("Inserted ", len(al), " articles from source: ", source.Name)
	}
}

func insertArticles(f *FeedParser, articles []Article) error {

	db, err := sql.Open("sqlite3", f.config.Database.Conn)
	if err != nil {
		log.Println("Error opening database: ", err)
		return err
	}

	defer db.Close()

	// db.Exec("CREATE TABLE IF NOT EXISTS articles (id TEXT PRIMARY KEY, title TEXT, link TEXT, date TEXT, content TEXT, source TEXT, translated_content TEXT)")

	for _, article := range articles {
		_, err = db.Exec("INSERT OR IGNORE INTO articles (id, title, link, date, content, source) VALUES (?, ?, ?, ?, ?, ?)", article.ID, article.Title, article.Link, article.Date, article.Content, article.Source)
		if err != nil {
			log.Println("Error inserting article: ", err)
		}
	}

	return nil
}

func uniqueIDFromString(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}

func parseSource(source Source) ([]Article, error) {
	const minContentLength = 100
	var err error
	var articles []Article
	var content string

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(source.Feed)
	if err != nil {
		log.Println("Error parsing feed: ", err)
		return nil, err
	}

	for _, item := range feed.Items {

		if source.Getwebsite {
			content, err = getWebsiteContent(item.Link)
			if err != nil {
				log.Println("Error getting website content: ", err)
				continue
			}
		} else {
			r := reflect.ValueOf(item)
			f := reflect.Indirect(r)
			content = f.FieldByName(source.Contentkey).String()
		}

		contentLength := len(content)

		if contentLength < minContentLength {
			continue
		}

		articles = append(articles, Article{
			ID:      uniqueIDFromString(item.Link),
			Title:   item.Title,
			Link:    item.Link,
			Date:    item.Published,
			Content: content,
			Source:  source.Name,
		})

	}

	return articles, nil
}

func getWebsiteContent(url string) (string, error) {
	text := fmt.Sprintf("Getting content from website: %s", url)
	return text, nil
}

func NewFeedParser(config string) {
	f := &FeedParser{}
	f.Load(config)
	f.Parse()
}

func main() {
	NewFeedParser("config.json")
}
