package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	_ "embed"

	"github.com/go-rod/rod"
)

var (
	//go:embed extractor.js
	jsCode string
)

type ExtractedElement []struct {
	ImageURL string `json:"imageUrl"`
	VideoURL string `json:"videoUrl"`
	Time     string `json:"time"`
	Desc     string `json:"desc"`
}

func rumbleGet(url string) (ExtractedElement, error) {
	browser := rod.New().MustConnect()
	defer browser.MustClose()
	page := browser.MustPage(url).MustWaitStable()
	res := page.MustEval(jsCode)
	var elements ExtractedElement
	err := res.Unmarshal(&elements)
	if err != nil {
		log.Fatalf("Failed to parse elements: %v", err)
	}
	return elements, nil
}

func main() {
	var inputURL string
	flag.StringVar(&inputURL, "url", "", "input url")
	flag.Parse()
	elements, err := rumbleGet(inputURL)
	if err != nil {
		log.Printf("Failed to get rumble: %v\n", err)
	}
	output, err := json.MarshalIndent(elements, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal json: %v\n", err)
	}
	fmt.Println(string(output))
}
