package main

import (
	"fmt"
	"sync"

	"github.com/peterbeamish/InsiderTrading/pkg/scraping"
)

func main() {

	fmt.Println("Initializing Scraper")

	scraper, err := scraping.NewSECScraper()
	if err != nil {
		fmt.Errorf("Failed to initialize scraper")
		return
	}

	//scraper.ScrapeByCIK("0000320193")
	scraper.ScapeByTicker("aapl")

	var wg sync.WaitGroup
	wg.Add(1)

	wg.Wait()
}
