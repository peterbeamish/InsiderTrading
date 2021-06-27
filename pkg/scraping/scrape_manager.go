package scraping

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/peterbeamish/InsiderTrading/pkg/model"
	"github.com/peterbeamish/InsiderTrading/pkg/scraping/scrapers"
)

//go:generate protoc -I. --go_out=. --go_opt=paths=source_relative scrape_data.proto

type ScrapeManager struct {
	scrapeInterval time.Duration
	tickersManaged map[string]context.CancelFunc
	reportChannel  chan *model.ScrapedInsiderReport
}

func NewScrapeManager(reportChannel chan *model.ScrapedInsiderReport) (*ScrapeManager, error) {
	var manager ScrapeManager

	manager.scrapeInterval = 5 * time.Second
	manager.tickersManaged = make(map[string]context.CancelFunc)
	manager.reportChannel = reportChannel

	return &manager, nil
}

// AddTicker Adds a new ticker to the scrape schedule
func (m *ScrapeManager) AddTicker(wg *sync.WaitGroup, ticker string) error {
	if _, ok := m.tickersManaged[ticker]; ok {
		return errors.New("Ticker already managed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.tickersManaged[ticker] = cancel
	// Start thread
	go m.runScraping(ctx, wg, ticker)

	return nil
}

// This will continually monitor the souce of the insider information.
func (m *ScrapeManager) runScraping(ctx context.Context, wg *sync.WaitGroup, ticker string) {
	defer wg.Done()

	// Initialize the SEC scraper
	scraper, err := scrapers.NewSECScraper()
	if err != nil {
		fmt.Errorf("Failed to initialize scraper")
		return
	}

	scrapeTicker := time.NewTicker(m.scrapeInterval)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Scraping cancelled for ticker: ", ticker)
			return
		case t := <-scrapeTicker.C:
			fmt.Printf("%d Scrape Initiated: %s \n", t.Unix(), ticker)
			err = scraper.ScapeByTicker(ticker, m.reportChannel)
			if err != nil {
				fmt.Errorf("Error scraping ticker: %s, "+err.Error(), ticker)
				return
			}
			//scrapeTicker.Stop()

		}

	}
}

// CancelScraping cancels scraping and terminates thread for ticker provided
func (m *ScrapeManager) CancelScraping(ticker string) error {
	if cancelFunc, ok := m.tickersManaged[ticker]; ok {
		cancelFunc()
		delete(m.tickersManaged, ticker)
		return nil
	} else {
		return errors.New("Ticker not managed")
	}
}

// CancelAll cancels scraping and terminates thread for all tickers
func (m *ScrapeManager) CancelAll() error {
	for _, cancelFunc := range m.tickersManaged {
		cancelFunc()
	}
	m.tickersManaged = make(map[string]context.CancelFunc, 0)
	return nil
}
