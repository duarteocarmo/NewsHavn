package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/mmcdole/gofeed"
)

type Source struct {
	Name       string
	Feed       string
	Getwebsite bool
	Contentkey string
}
type Config struct {
	Sources []Source
}
type FeedParser struct {
	config Config
}
type Article struct {
	ID      string
	Title   string
	Link    string
	Date    string
	Content string
	Source  string
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

		for _, a := range al {
			log.Printf("ID: %s, Title: %s, Source: %s", a.ID, a.Title, a.Source)
		}
	}
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

	log.Println("Parsing feed for source: ", source.Name)
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
