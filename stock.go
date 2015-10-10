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

type Stock struct {
	symbol string
}

func NewStock(sym string) *Stock {
	return &Stock{
		symbol: sym,
	}
}

const chartAPIURL string = "http://dev.markitondemand.com/Api/v2/InteractiveChart/json"

type ChartAPIRequest struct {
	stock     *Stock
	startDate string
	endDate   string
	url       string
}

type ChartAPIRequestParams struct {
	Normalized   bool
	NumberOfDays int
	DataPeriod   string
	Elements     []Element
}

type Element struct {
	Symbol string
	Type   string
	Params []string `json:",omitempty"`
}

type ChartAPIResponse struct {
}

func NewChartAPIRequest(s *Stock, start time.Time, end time.Time) (*ChartAPIRequest, error) {
	request := &ChartAPIRequest{
		stock:     s,
		startDate: start.Format("2006-01-02T15:04:05-00"), // formatted like 2011-06-01T00:00:00-00
		endDate:   end.Format("2006-01-02T15:04:05-00"),
	}
	request.url = chartAPIURL

	params := ChartAPIRequestParams{
		Normalized:   false,
		NumberOfDays: 30, // TODO change!
		DataPeriod:   "Day",
		Elements: []Element{
			Element{
				Symbol: s.symbol,
				Type:   "price",
				Params: []string{"ohlc"},
			},
			Element{
				Symbol: s.symbol,
				Type:   "volume",
			},
		},
	}

	jsonStr, err := json.Marshal(params)
	request.url = fmt.Sprintf("%s?parameters=%s", request.url, jsonStr)

	fmt.Printf("request.url:\n%s\n\n", request.url)

	resp, err := http.Get(request.url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	fmt.Println("response Body:", string(body))

	return request, nil

}

func PollNewData(symbol string) (string, error) {
	stock := NewStock(symbol)
	fmt.Printf("Stock:\n %+v \n", stock)

	request, err := NewChartAPIRequest(
		stock,
		time.Now().AddDate(0, 0, -3),
		time.Now())

	if err != nil {
		panic(err)
	}

	fmt.Printf("\n\nREQUEST:\n %+v \n", request)
	return "", nil
}
