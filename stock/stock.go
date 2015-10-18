// Copyright 2015 Jordan Hurwich - no license granted

package stock

import (
	"fmt"
	"log"
	"time"
)

// Span and Measure objects for calculating and saving trend data
type Span []Measure
type Measure struct {
	time  time.Time
	value float32
	trend float32
}

// Stock object for manages all data accesses for a specific stock symbol
type Stock struct {
	Symbol   string
	Measures Span
}

func NewStock(sym string) *Stock {
	return &Stock{
		Symbol: sym,
	}
}

// query daily measure data between times provided for a stock
func (s *Stock) Range(startDate time.Time, endDate time.Time) *Span {
	if startDate.IsZero() && endDate.IsZero() {
		// zero time is used as sentinel to query entire range available
		//TODO(jhurwich) implement
	}

	return &Span{}
}
func (s *Stock) RangeAll() *Span {
	// zero time is used as sentinel to query all data available
	return s.Range(time.Time{}, time.Time{})
}

// populate daily measure data (value field) between times provided
func (s *Stock) Populate(startDate time.Time, endDate time.Time) error {
	if startDate.IsZero() && endDate.IsZero() {
		// zero time is used as sentinel to query entire range available
		//TODO(jhurwich) implement
	}

	return nil
}
func (s *Stock) PopulateAll() error {
	// zero time is used as sentinel to populate all data
	return s.Populate(time.Time{}, time.Time{})
}

// calculate trend for measures between times provided
func (s *Stock) Analyze(startDate time.Time, endDate time.Time) error {
	if startDate.IsZero() && endDate.IsZero() {
		// zero time is used as sentinel to query entire range available
		//TODO(jhurwich) implement
	}

	return nil
}
func (s *Stock) AnalyzeAll() error {
	// zero time is used as sentinel to analyze all data
	return s.Analyze(time.Time{}, time.Time{})
}

func PollNewData(symbol string) (string, error) {
	stock := NewStock(symbol)

	arbitraryTime := time.Date(2011, time.May, 19, 22, 47, 0, 0, time.UTC)
	request, err := NewMarkitChartAPIRequest(
		stock,
		arbitraryTime.AddDate(0, 0, 30),
		arbitraryTime)

	if err != nil {
		log.Fatal(err)
	}
	response, err := request.Request()
	if err != nil {
		fmt.Printf("Request failed:%s\n", err)
	}

	// do something with response
	fmt.Printf("RESPONSE:\n%s\n\n", response.String())

	return "", nil
}
