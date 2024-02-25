package db

import (
	"database/sql"
	"log"

	"github.com/duarteocarmo/hyggenews/types"
)

func GetArticles(s *types.Server) []types.Article {
	db, err := sql.Open("sqlite3", s.Db.Conn)
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
func GetArticleByID(s *types.Server, id string) (*types.Article, error) {
	db, err := sql.Open("sqlite3", s.Db.Conn)
	if err != nil {
		log.Println("Error opening database: ", err)
		return nil, err
	}
	defer db.Close()

	var article types.Article

	query := `SELECT id, title, link, date, content, source, TranslatedContent, TranslatedTitle, Category FROM articles WHERE id = ?`
	row := db.QueryRow(query, id)

	err = row.Scan(&article.ID, &article.Title, &article.Link, &article.Date, &article.Content, &article.Source, &article.TranslatedContent, &article.TranslatedTitle, &article.Category)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		log.Println("Error querying database by ID: ", err)
		return nil, err
	}

	return &article, nil
}

func GetCategories(s *types.Server) ([]string, error) {
	db, err := sql.Open("sqlite3", s.Db.Conn)
	if err != nil {
		log.Println("Error opening database: ", err)
		return nil, err
	}
	defer db.Close()

	var categories []string

	query := `SELECT DISTINCT category FROM articles`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Error querying database: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		err := rows.Scan(&category)
		if err != nil {
			log.Println("Error scanning database result: ", err)
			continue
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		log.Println("Error with database rows: ", err)
		return nil, err
	}

	return categories, nil
}

func GetArticlesByCategory(s *types.Server, category string) ([]types.Article, error) {
	db, err := sql.Open("sqlite3", s.Db.Conn)
	if err != nil {
		log.Println("Error opening database: ", err)
		return nil, err
	}
	defer db.Close()

	var articles []types.Article

	query := `
		SELECT id, title, link, date, content, source, TranslatedContent, TranslatedTitle 
		FROM articles 
		WHERE category = ?
		AND date >= date('now') 
		AND date < date('now', '+1 day');
	`
	rows, err := db.Query(query, category)
	if err != nil {
		log.Println("Error querying database: ", err)
		return nil, err
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
		return nil, err
	}

	return articles, nil
}
