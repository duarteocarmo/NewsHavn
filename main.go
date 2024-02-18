package main

import (
	"github.com/duarteocarmo/hyggenews/parser"
	"github.com/duarteocarmo/hyggenews/types"
)

func NewFeedParser(config string) {
	f := types.FeedParser{}
	parser.Load(&f, config)
	parser.Parse(&f)
}

func main() {
	NewFeedParser("config.json")
}
