package scrapers

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/peterbeamish/InsiderTrading/pkg/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SECScraper struct {
	collecter     *colly.Collector
	tickerLookup  map[string]string
	ticker        string
	reportChannel chan *model.ScrapedInsiderReport
}

type cikTransactionRequest struct {
	CIK string
}

const SECBASEURL = "https://www.sec.gov"
const SECGETACTIVITYBYCIK = "/cgi-bin/own-disp?action=getissuer&CIK={{.CIK}}&type=&dateb=&owner=include&start=1"
const SECGETALLTICKERS = "/include/ticker.txt"

func NewSECScraper() (*SECScraper, error) {

	var scraper SECScraper
	// Cache all the known tickers, and their CIKs
	scraper.initializeTickerMap()

	scraper.collecter = colly.NewCollector()

	// Log when requests are made
	scraper.collecter.OnRequest(scraper.onRequest)

	// Callback when a table is found
	scraper.collecter.OnHTML("table[id=\"transaction-report\"] tbody", scraper.cikTransactionScrape)
	scraper.collecter.Async = true
	scraper.collecter.AllowURLRevisit = true

	return &scraper, nil

}

var tickerLookupSingleton map[string]string

func (s *SECScraper) initializeTickerMap() {

	if tickerLookupSingleton == nil {
		s.tickerLookup = map[string]string{}
		// Load all the tickers CIKs
		cikScraper := colly.NewCollector()
		cikScraper.OnResponse(s.tikerScrape)
		cikScraper.Visit(SECBASEURL + SECGETALLTICKERS)
		tickerLookupSingleton = s.tickerLookup
	} else {
		s.tickerLookup = tickerLookupSingleton
	}
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

	var report model.ScrapedInsiderReport
	report.ExtractionTime = timestamppb.Now()
	report.Ticker = s.ticker

	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {

		var transaction model.InsiderTransaction
		var validRow bool = false
		row.ForEach("td", func(columnIndex int, colData *colly.HTMLElement) {

			validRow = true
			switch columnIndex {
			// Transaction Type
			case 0:
				transactionType := colData.Text
				switch transactionType {
				case "D":
					transaction.TransactionType = model.InsiderTransaction_DISPOSITION
				case "S":
					transaction.TransactionType = model.InsiderTransaction_SELL
				case "P":
					transaction.TransactionType = model.InsiderTransaction_BUY
				case "A":
					transaction.TransactionType = model.InsiderTransaction_AWARD
				}

			// Transaction Date
			case 1:
				date := colData.Text
				time, err := time.Parse("2006-01-02", date)
				if err != nil {
					return
				}
				transaction.TransactionTime = timestamppb.New(time)

			// Person
			case 3:
				transaction.InsiderName = colData.Text

			// Number of shares transacted
			case 7:
				strNumberOfShares := strings.Trim(colData.Text, " ")

				transaction.NumberOfSharesTransacted, _ = strconv.ParseFloat(strNumberOfShares, 64)
			// Number of shares owned
			case 8:
				strNumberOfShares := strings.Trim(colData.Text, " ")
				transaction.NumberOfSharesOwned, _ = strconv.ParseFloat(strNumberOfShares, 64)
			}

		})
		if validRow && transaction.TransactionType == model.InsiderTransaction_DISPOSITION {
			report.Transactions = append(report.Transactions, &transaction)
		}

	})

	// Report has been produced, let's send this
	s.reportChannel <- &report
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

func (s *SECScraper) ScapeByTicker(ticker string, reportChannel chan *model.ScrapedInsiderReport) error {
	cik, ok := s.tickerLookup[ticker]
	if !ok {
		return errors.New("Failed to resolve cik of provided ticker")
	}
	s.ticker = ticker
	s.reportChannel = reportChannel
	return s.ScrapeByCIK(cik)
}

func (s *SECScraper) DoScrape(url string) {
	fmt.Println("doing scrape: " + url)
	s.collecter.Visit(url)
}
