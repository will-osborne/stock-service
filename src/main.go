package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	stockAPIAddr                  = "https://www.alphavantage.co"
	stcokQueryRequestPathTemplate = `%s/query?apikey=%s&function=TIME_SERIES_DAILY_ADJUSTED&symbol=%s`
	stockAPIKey                   string
	numDays                       int
	stockSymbol                   string
)

// stockQueryResponse represents a response from a request to alphavantage TIME_SERIES_DAILY_ADJUSTED
type stockQueryResponse struct {
	Series map[string]struct{
		Close float64 `json:"4. close,string"`
	} `json:"Time Series (Daily)"`
}

// getStockClosesResponse is the response object to return for an incoming request to get getStockCloses
type getStockClosesResponse struct {
	Stock string    `json:"stock"`
	Data  []float64 `json:"data"`
	Close float64   `json:"averageClose"`
}

func init() {
	log.Println("Startup..")
	var err error

	numDays, err = strconv.Atoi(requireEnv("NDAYS"))
	if err != nil {
		log.Fatalf("Error converting env var NDAYS to type int: %s", err)
	}
	stockSymbol = requireEnv("SYMBOL")
	stockAPIKey = requireSecretFile("/mnt/secrets/stockAPIKey")
}

// requireEnv returns the contents of an env var and will cause the program to exit if it does not exist
func requireEnv(key string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		log.Fatalf("Env '%s' is required but not set. Exiting..", key)
		return ""
	}
	return value
}
// requireSecretFile returns the contents of a file at a given path and will cause the program to exit if it fails
func requireSecretFile(filepath string) string {
	contents, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Error occured reading required secret from file (%s): %s", filepath, err)
		return ""
	}

	if len(contents) == 0 {
		log.Fatalf("Secrets file '%s' is required but has no content. Exiting..", filepath)
		return ""
	}
	return string(contents)
}

func main() {
	router := mux.NewRouter()
	router.Handle("/", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(getStockCloses))).
		Methods("OPTIONS", "GET")

	log.Println("Listening on :8000")
	err := http.ListenAndServe(":8000", router)
	log.Fatal(err)
}

// getStockCloses is an HTTP handler which responds with the previous N days of stock close values for stock X as well as an average close value.
func getStockCloses(w http.ResponseWriter, _ *http.Request) {
	// Formatted path of the endpoint from which we shall gain our stock info
	stockQueryReqPath := fmt.Sprintf(stcokQueryRequestPathTemplate, stockAPIAddr, stockAPIKey, stockSymbol)

	// Make requests to alphavantage API
	stockQueryResp, err := http.Get(stockQueryReqPath)
	if err != nil {
		http.Error(w, "There was an error processing your request", http.StatusInternalServerError)
		log.Printf("Error retreiving stock info from '%s': %s", stockAPIAddr, err)
		return
	}
	if stockQueryResp == nil {
		http.Error(w, "There was an error processing your request", http.StatusInternalServerError)
		log.Printf("Stock info response was nil")
		return
	}

	// Decode response from alphavantage
	var stockData stockQueryResponse
	err = json.NewDecoder(stockQueryResp.Body).Decode(&stockData)
	if err != nil {
		http.Error(w, "There was an error processing your request", http.StatusInternalServerError)
		log.Printf("Error decoding stock info response: %s", err)
		return
	}

	stockClosesResponse := getStockClosesResponse{
		Stock: stockSymbol,
	}

	dateFormat := "2006-01-02" // year-month-day
	now := time.Now()

	// For the last N days, not including today and ignoring weekends, add the close to the stockClosesResponse Data slice and track the cumulative value so it can be averaged.
	var cumulativeStockClose float64 = 0
	weekendCompensator := 0
	for i := 1; i <= numDays; i++ {
		targetDay := now.AddDate(0,0,-(i + weekendCompensator))

		// Act as if weekends don't exist and add an offset to use when getting target date
		switch targetDay.Weekday() {
		case time.Saturday:
			weekendCompensator ++
		case time.Sunday:
			weekendCompensator +=2
		}
		targetDay = now.AddDate(0,0,-(i + weekendCompensator))

		formattedDayToReturn := targetDay.Format(dateFormat)
		thisDayClose := stockData.Series[formattedDayToReturn].Close
		stockClosesResponse.Data = append(stockClosesResponse.Data, thisDayClose)
		cumulativeStockClose += thisDayClose
	}

	stockClosesResponse.Close = cumulativeStockClose/float64(numDays)

	// Encode and send response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(stockClosesResponse)
	if err != nil {
		http.Error(w, "There was an error processing your request", http.StatusInternalServerError)
		log.Printf("Error encoding response object: %s", err)
		return
	}
}