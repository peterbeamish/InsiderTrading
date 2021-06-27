package scrapers

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"

	"github.com/gocolly/colly"
	"github.com/peterbeamish/InsiderTrading/pkg/model"
)

type SECScraper struct {
	collecter    *colly.Collector
	tickerLookup map[string]string
}

type cikTransactionRequest struct {
	CIK string
}

const SECBASEURL = "https://www.sec.gov"
const SECGETACTIVITYBYCIK = "/cgi-bin/own-disp?action=getissuer&CIK={{.CIK}}&type=&dateb=&owner=include&start=1"
const SECGETALLTICKERS = "/include/ticker.txt"

func NewSECScraper() (*SECScraper, error) {

	var scraper SECScraper
	scraper.tickerLookup = map[string]string{}

	// Load all the tickers CIKs
	cikScraper := colly.NewCollector()
	cikScraper.OnResponse(scraper.tikerScrape)
	cikScraper.Visit(SECBASEURL + SECGETALLTICKERS)

	scraper.collecter = colly.NewCollector()

	scraper.collecter.OnRequest(scraper.onRequest)

	scraper.collecter.OnHTML("table[id=\"transaction-report\"] tbody", scraper.cikTransactionScrape)

	return &scraper, nil

}

func (s *SECScraper) tikerScrape(resp *colly.Response) {
	if resp.StatusCode != http.StatusOK {
		return
	}
	r, _ := regexp.Compile("(.*)\t(.*)")

	matches := r.FindAllSubmatch(resp.Body, -2)

	for _, match := range matches {

		ticker := string(match[1])
		cik := string(match[2])
		s.tickerLookup[ticker] = cik
	}
}

func (s *SECScraper) cikTransactionScrape(e *colly.HTMLElement) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {

		var transaction model.InsiderTransaction

		row.ForEach("td", func(columnIndex int, colData *colly.HTMLElement) {

			switch columnIndex {
			case 0:
				transactionType := colData.Text
				if transactionType == "D" {
					transaction.TransactionType = model.InsiderTransaction_DISPOSITION
				}

			}

			fmt.Println(colData)
		})
		//rowText := string(row.Text)
		//fmt.Println(rowText)

	})
}

func (s *SECScraper) onRequest(r *colly.Request) {
	fmt.Println("Visiting", r.URL)
}

func (s *SECScraper) ScrapeByCIK(cik string) error {
	// We need to format the URL to request the page containing transactions
	var request cikTransactionRequest
	request.CIK = cik

	tmpl, err := template.New("CIKRequest").Parse(SECBASEURL + SECGETACTIVITYBYCIK)
	if err != nil {
		return err
	}

	reqBuf := bytes.NewBufferString("")
	err = tmpl.Execute(reqBuf, request)
	if err != nil {
		return err
	}

	return s.collecter.Visit(reqBuf.String())
}

func (s *SECScraper) ScapeByTicker(ticker string) error {
	cik, ok := s.tickerLookup[ticker]
	if !ok {
		return errors.New("Failed to resolve cik of provided ticker")
	}
	return s.ScrapeByCIK(cik)
}

func (s *SECScraper) DoScrape(url string) {
	fmt.Println("doing scrape: " + url)
	s.collecter.Visit(url)
}
