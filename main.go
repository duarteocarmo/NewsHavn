package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mmcdole/gofeed"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

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
	TranslatedTitle   string
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
	type Result struct {
		articles []Article
		err      error
	}

	// Fetch all articles from all sources
	log.Println("Fetching articles from all sources")
	ch := make(chan Result, len(f.config.Sources))
	start := time.Now()
	for _, source := range f.config.Sources {
		go func() {
			a, err := parseSource(source)
			ch <- Result{a, err}
		}()
	}

	var articles []Article
	for i := 0; i < len(f.config.Sources); i++ {
		result := <-ch
		if result.err != nil {
			log.Println("Error processing feed: ", result.err)
		}
		articles = append(articles, result.articles...)
	}
	log.Printf("Received %d articles in %s", len(articles), time.Since(start))

	// Translate all articles
	log.Println("Translating all articles")
	results := make(chan error, len(articles))
	start = time.Now()
	isTitle := false
	for i := range articles {
		article := &articles[i]
		go func(article *Article) {
			var err error
			article.TranslatedContent, err = translate(article.Content, isTitle)
			results <- err
		}(article)
	}
	var err error
	for i := 0; i < len(articles); i++ {
		e := <-results
		if e != nil {
			log.Println("Error translating item: ", err)
			err = e
		}
	}
	log.Printf("Translated %d articles in %s", len(articles), time.Since(start))

	// Insert all articles into the database
	log.Println("Inserting all articles into the database")
	start = time.Now()
	err = insertArticles(f, articles)
	if err != nil {
		log.Println("Error inserting articles into the database: ", err)
	}
	log.Printf("Inserted %d articles into the database in %s", len(articles), time.Since(start))

	log.Println("Done")

}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	s := string(b)
	return s[1 : len(s)-1]
}

func translate(content string, istitle bool) (string, error) {

	const (
		apiURL            = "https://api.endpoints.anyscale.com/v1/chat/completions"
		model             = "mistralai/Mixtral-8x7B-Instruct-v0.1"
		temperature       = 0
		// topP              = 0.7
		// topK              = 50
		// repetitionPenalty = 1
		// numCompletions    = 1
	)

	if content == "" {
		return "", fmt.Errorf("content is empty")
	}

	var query string
	var maxTokens int
	if istitle == true {
		query = fmt.Sprintf("Translate the following title into English, do not output any notes or explanations, just title when prompted for:\nOriginal Title: %s\nEnglish title:", content)
		maxTokens = 50

	} else {
		query = fmt.Sprintf("You are a highly skilled professional translator. When you receive an article in Danish, your critical task is to translate it into English. You do not output any html, but the actual text of the article. You do not add any notes or explanations. The article to translate will be inside the <article> tags. Once prompted, just output the English translation.\n\n\n<article>\n\n%s\n\n</article>\n\n\nEnglish translation:", content)
		maxTokens = 8400
	}

	query = jsonEscape(query)

	payloadTemplate := `{
		"model":"%s",
		"max_tokens":%d,
		"stop":["</s>","[/INST]"],
		"temperature":%d,
		"messages":[{"role":"user","content":"%s"}]
	}`

	jsonPayload := fmt.Sprintf(payloadTemplate, model, maxTokens, temperature, query)

	// print json payload
	payload := strings.NewReader(jsonPayload)
	log.Println("JSON payload: ", jsonPayload)

	req, err := http.NewRequest("POST", apiURL, payload)
	if err != nil {
		return "", err
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("API_KEY environment variable not set")
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("error making request: %v\n", err)
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error response from Together API: %d", res.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Println("Error decoding JSON response body: ", err)
		return "", err
	}

	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		log.Println("No content found in the message of the response body.")
		return "", fmt.Errorf("no content found in the message of the response body")
	}

	return result.Choices[0].Message.Content, nil
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
		query := `
			INSERT OR IGNORE INTO articles 
			(id, title, link, date, content, source, TranslatedContent, TranslatedTitle) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err = db.Exec(query,
			article.ID,
			article.Title,
			article.Link,
			article.Date,
			article.Content,
			article.Source,
			article.TranslatedContent,
			article.TranslatedTitle,
		)

		if err != nil {
			log.Println("Error inserting article:", err)
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
