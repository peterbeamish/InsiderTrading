package main

import (
	"fmt"
	"sync"

	"github.com/peterbeamish/InsiderTrading/pkg/model"
	"github.com/peterbeamish/InsiderTrading/pkg/scraping"
)

func main() {

	fmt.Println("Initializing Scraper")

	scrapeReports := make(chan *model.ScrapedInsiderReport, 5)

	manager, err := scraping.NewScrapeManager(scrapeReports)
	if err != nil {
		fmt.Errorf("Failed to initialize scrape manager")
		return
	}

	go func() {
		for {
			select {
			case report, ok := <-scrapeReports:
				if !ok {
					fmt.Println("Channel closing")
					goto exitloop
				}
				fmt.Printf("Report for ticker %s recieved\n", report.Ticker)

			}
		}
	exitloop:
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	manager.AddTicker(&wg, "aapl")
	manager.AddTicker(&wg, "tsla")
	manager.AddTicker(&wg, "ge")

	wg.Wait()
}
