// Copyright 2015 Jordan Hurwich - no license granted

package stock

import (
	"fmt"
	"log"
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
