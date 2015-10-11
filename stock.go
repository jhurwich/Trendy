// Copyright 2015 Jordan Hurwich - no license granted

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
		log.Fatal(err)
	}
	response, err := request.Request()
	if err != nil {
		fmt.Printf("PANIC\n%s\n\n", err)
		log.Fatal(err)
	}

	fmt.Printf("RESPONSE:\n%+v\n\n", response)
	return "", nil
}

/*  - - - - - Markit  - - - - - */

const markitChartAPIURL string = "http://dev.markitondemand.com/Api/v2/InteractiveChart/json"

// Constructor for MarkitChartAPIRequests
func NewMarkitChartAPIRequest(s *Stock, start time.Time, end time.Time) (*MarkitChartAPIRequest, error) {
	request := &MarkitChartAPIRequest{
		Stock:     s,
		StartDate: start.Format(ISOFormat), // formatted like 2011-06-01T00:00:00-00
		EndDate:   end.Format(ISOFormat),
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
	r, err := http.Get(request.Url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}

	response := new(MarkitChartAPIResponse)
	err = json.NewDecoder(r.Body).Decode(response)
	if err != nil {
		log.Fatal(err)
	}

	return response, nil
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
	Params     []string    `json:",omitempty"`
	Currency   string      `json:",omitempty"`
	Timestamp  string      `json:"TimeStamp,omitempty"`
	Dataseries *Dataseries `json:"DataSeries,omitempty"`
}
type Dataseries struct {
	Open   *Data `json:"open,omitempty"`
	High   *Data `json:"high,omitempty"`
	Low    *Data `json:"low,omitempty"`
	Close  *Data `json:"close,omitempty"`
	Volume *Data `json:"volume,omitempty"`
}
type Data struct {
	Min     float32   `json:"min,omitempty"`
	Max     float32   `json:"max,omitempty"`
	MaxDate *ISOTime  `json:"maxDate,omitempty"`
	MinDate *ISOTime  `json:"minDate,omitempty"`
	Values  []float32 `json:"values,omitempty"`
}

// Markit API response format
type MarkitChartAPIResponse struct {
	Labels    *MarkitChartAPIResponseLabels
	Positions []float32
	Dates     []ISOTime
	Elements  []Element
}
type MarkitChartAPIResponseLabels struct {
	Dates      []string `json:"dates"`
	Pos        []string `json:"pos"`
	Priorities []string `json:"priorities"`
	Text       []string `json:"text"`
	UtcDates   []string `json:"utcdates"`
}

/*  - - - - - ISOTime  - - - - - */

// ISOTime extensions to time.Time including JSON (Un)Marshaling
type ISOTime struct {
	time.Time
}

const ISOFormat = "2006-01-02T15:04:05"

func (it ISOTime) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	s := it.Time.Format(ISOFormat)
	enc.Encode(s)
	return b.Bytes(), nil
}
func (it *ISOTime) UnmarshalJSON(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b)
	var s string
	if err := dec.Decode(&s); err != nil {
		return err
	}
	t, err := time.Parse(ISOFormat, s)
	if err != nil {
		return err
	}
	it.Time = t
	return nil
}
func (it ISOTime) String() string {
	return it.Format(ISOFormat)
}
