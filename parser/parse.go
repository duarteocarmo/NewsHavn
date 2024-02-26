package parser

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/duarteocarmo/newshavn/types"
	"github.com/mmcdole/gofeed"
	"jaytaylor.com/html2text"

	_ "github.com/mattn/go-sqlite3"
)

func Load(f *types.FeedParser, configFilePath string) *types.FeedParser {
	file, _ := os.Open(configFilePath)
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := types.Config{}
	err := decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	f.Config = config

	return f
}

func Parse(f *types.FeedParser) {
	type Result struct {
		articles []types.Article
		err      error
	}

	// Fetch all articles from all sources
	log.Println("Fetching articles from all sources")
	ch := make(chan Result, len(f.Config.Sources))
	start := time.Now()
	for _, source := range f.Config.Sources {
		go func() {
			a, err := parseSource(source)
			ch <- Result{a, err}
		}()
	}

	var articles []types.Article
	for i := 0; i < len(f.Config.Sources); i++ {
		result := <-ch
		if result.err != nil {
			log.Println("Error processing feed: ", result.err)
		}
		articles = append(articles, result.articles...)
	}
	log.Printf("Received %d articles in %s", len(articles), time.Since(start))

	// Remove duplicates
	seenids := make(map[string]struct{})
	result := []types.Article{}
	for _, article := range articles {
		if _, exists := seenids[article.ID]; !exists {
			seenids[article.ID] = struct{}{}
			result = append(result, article)
		} else {
			log.Println("Duplicate article: ", article.ID)
		}
	}
	articles = result

	// Remove if already in DB
	dbArticleIDs := getArticlesFromDB(f)
	filtered := []types.Article{}
	for _, article := range articles {
		if !slices.Contains(dbArticleIDs, article.ID) {
			filtered = append(filtered, article)
		}
	}
	articles = filtered

	if len(articles) == 0 {
		log.Println("No articles to process")
		return
	}

	// Translate all articles
	log.Println("Translating all articles")
	results := make(chan error, len(articles))
	start = time.Now()
	isTitle := false
	for i := range articles {
		article := &articles[i]
		go func(article *types.Article) {
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

	// Translate all titles
	time.Sleep(4 * time.Second)
	log.Println("Translating all articles")
	results = make(chan error, len(articles))
	start = time.Now()
	isTitle = true
	for i := range articles {
		article := &articles[i]
		go func(article *types.Article) {
			var err error
			article.TranslatedTitle, err = translate(article.Title, isTitle)
			results <- err
		}(article)
	}
	for i := 0; i < len(articles); i++ {
		e := <-results
		if e != nil {
			log.Println("Error translating item: ", err)
			err = e
		}
	}
	log.Printf("Translated %d article titles in %s", len(articles), time.Since(start))

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

func translate(content string, istitle bool) (string, error) {

	const (
		apiURL      = "https://api.together.xyz/v1/chat/completions"
		model       = "mistralai/Mistral-7B-Instruct-v0.2"
		temperature = 0
	)

	if content == "" {
		return "", fmt.Errorf("content is empty")
	}

	var query string
	var maxTokens int
	if istitle == true {
		query = fmt.Sprintf("You are a highly skilled and concise professional translator. When you receive a sentence in Danish, your task is to translate it into English. VERY IMPORTANT: Do not output any notes, explanations, alternatives or comments after or before the translation.\n\nDanish sentence: %s\n\nEnglish translation:", content)
		maxTokens = 50

	} else {
		query = fmt.Sprintf(`
You are a highly skilled professional translator. 

Here are your instructions:
- When you receive an article in Danish, your critical task is to translate it into English. 
- You do not output any html, but the actual text of the article. 
- You do not add any notes or explanations. 
- The article to translate will be inside the <article> tags. 
- Once prompted, just output the English translation.
- Do not output the title of the article, only the content.
- Make sure the translation is well formatted and easy to read (no useless line breaks, no extra spaces, etc.)

<article>

%s

</article>

Here is the best English translation of the article above:`, content)
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
	payload := strings.NewReader(jsonPayload)

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
	translation := result.Choices[0].Message.Content

	tRatio := float64(len(translation)) / float64(len(content))

	log.Println("=====================================")
	if tRatio < 0.5 {
		log.Println("Translation ratio is less than 0.5")
	}
	if tRatio > 2.0 && istitle == true {
		log.Println("Translation ratio is greater than 2.0 and is a title, limiting to first line")
		translation = strings.Split(translation, "\n")[0]
	}

	log.Println("Translation ratio: ", tRatio)
	log.Println("Translated item, isTitle: ", istitle)

	// log.Println("JSON payload: ", jsonPayload)
	// log.Println("Response: ", response)
	// log.Println("=====================================")

	return translation, nil
}

func getArticlesFromDB(f *types.FeedParser) []string {
	db, err := sql.Open("sqlite3", f.Config.Database.Conn)
	if err != nil {
		log.Println("Error opening database: ", err)
		return nil
	}

	defer db.Close()

	rows, err := db.Query("SELECT id FROM articles")
	if err != nil {
		log.Println("Error querying database: ", err)
		return nil
	}

	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			log.Println("Error scanning row: ", err)
			return nil
		}
		ids = append(ids, id)
	}

	return ids

}

func insertArticles(f *types.FeedParser, articles []types.Article) error {

	db, err := sql.Open("sqlite3", f.Config.Database.Conn)
	if err != nil {
		log.Println("Error opening database: ", err)
		return err
	}

	defer db.Close()

	for _, article := range articles {
		query := `
			INSERT OR IGNORE INTO articles 
			(id, title, link, date, content, source, TranslatedContent, TranslatedTitle, Category) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			article.Category,
		)

		if err != nil {
			log.Println("Error inserting article:", err)
		}
	}

	return nil
}

func parseSource(source types.Source) ([]types.Article, error) {
	const minContentLength = 100
	var err error
	var articles []types.Article
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

		text, err := html2text.FromString(content)
		if err != nil {
			log.Println("Error converting HTML to text: ", err)
			continue
		}

		articles = append(articles, types.Article{
			ID:       uniqueIDFromString(item.Link),
			Title:    item.Title,
			Link:     item.Link,
			Date:     item.PublishedParsed.UTC(),
			Content:  text,
			Source:   source.Name,
			Category: source.Category,
		})

	}

	return articles, nil
}

func getWebsiteContent(url string) (string, error) {
	text := fmt.Sprintf("Getting content from website: %s", url)
	return text, nil
}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	s := string(b)
	return s[1 : len(s)-1]
}

func uniqueIDFromString(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}
