package main

import (
	"fmt"
	"sync"

	"github.com/peterbeamish/InsiderTrading/pkg/scraping"
)

func main() {

	fmt.Println("Initializing Scraper")

	manager, err := scraping.NewScrapeManager()
	if err != nil {
		fmt.Errorf("Failed to initialize scrape manager")
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	manager.AddTicker(&wg, "aapl")

	wg.Wait()
}
