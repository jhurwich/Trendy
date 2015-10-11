// Copyright 2015 Jordan Hurwich - no license granted

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Stock object for manages all data accesses for a specific stock symbol
type Stock struct {
	Symbol string
}

// Stock constructor
func NewStock(sym string) *Stock {
	return &Stock{
		Symbol: sym,
	}
}

func PollNewData(symbol string) (string, error) {
	stock := NewStock(symbol)

	request, err := NewMarkitChartAPIRequest(
		stock,
		time.Now().AddDate(0, 0, -3),
		time.Now())

	if err != nil {
		panic(err)
	}
	request.Request()

	return "", nil
}

/*  - - - - - Markit  - - - - - */

const markitChartAPIURL string = "http://dev.markitondemand.com/Api/v2/InteractiveChart/json"

// Constructor for MarkitChartAPIRequests
func NewMarkitChartAPIRequest(s *Stock, start time.Time, end time.Time) (*MarkitChartAPIRequest, error) {
	request := &MarkitChartAPIRequest{
		Stock:     s,
		StartDate: start.Format("2006-01-02T15:04:05-00"), // formatted like 2011-06-01T00:00:00-00
		EndDate:   end.Format("2006-01-02T15:04:05-00"),
		Url:       markitChartAPIURL,
	}

	// use object to build json parameters for url
	params := MarkitChartAPIRequestParams{
		Normalized:   false,
		NumberOfDays: 30, // TODO change!
		DataPeriod:   "Day",
		Elements: []Element{
			Element{
				Symbol: s.Symbol,
				Type:   "price",
				Params: []string{"ohlc"},
			},
			Element{
				Symbol: s.Symbol,
				Type:   "volume",
			},
		},
	}
	jsonStr, err := json.Marshal(params)
	request.Url = fmt.Sprintf("%s?parameters=%s", request.Url, jsonStr)
	fmt.Printf("request.Url:\n%s\n\n", request.Url)

	return request, err
}

func (request *MarkitChartAPIRequest) Request() (*MarkitChartAPIResponse, error) {
	response, err := http.Get(request.Url)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(response.Status)
	}

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("response Status:", response.Status)
	fmt.Println("response Headers:", response.Header)
	fmt.Println("response Body:", string(body))

	if err != nil {
		panic(err)
	}

	fmt.Printf("\n\nREQUEST:\n %+v \n", request)
	return nil, nil
}

// Markit API request format and supporting structs
type MarkitChartAPIRequest struct {
	Stock     *Stock
	StartDate string
	EndDate   string
	Url       string
}
type MarkitChartAPIRequestParams struct {
	Normalized   bool
	NumberOfDays int
	DataPeriod   string
	Elements     []Element
}
type Element struct {
	Symbol     string
	Type       string
	Params     []string   `json:",omitempty"`
	Currency   string     `json:",omitempty"`
	Timestamp  string     `json:"TimeStamp,omitempty"`
	Dataseries Dataseries `json:"DataSeries"`
}
type Dataseries struct {
	Open   Data `json:"open,omitempty"`
	High   Data `json:"high,omitempty"`
	Low    Data `json:"low,omitempty"`
	Close  Data `json:"close,omitempty"`
	Volume Data `json:"volume,omitempty"`
}
type Data struct {
	Min     float32   `json:"min"`
	Max     float32   `json:"max"`
	MaxDate time.Time `json:"maxDate"`
	MinDate time.Time `json:"minDate"`
	Values  []float32 `json:"values"`
}

// Markit API response format
type MarkitChartAPIResponse struct {
	Labels    MarkitChartAPIResponseLabels
	Positions []float32
	Dates     []time.Time
	Elements  []Element
}
type MarkitChartAPIResponseLabels struct {
	Dates      []string `json:"dates"`
	Pos        []string `json:"pos"`
	Priorities []string `json:"priorities"`
	Text       []string `json:"text"`
	UtcDates   []string `json:"utcdates"`
}
