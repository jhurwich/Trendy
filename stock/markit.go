// Copyright 2015 Jordan Hurwich - no license granted

package stock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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
		Normalized: false,
		StartDate:  request.StartDate,
		EndDate:    request.EndDate,
		DataPeriod: "Day",
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
		fmt.Printf("Could not decode response from Markit to JSON: %v", err)
		return nil, err
	}

	// return any error that might have been provided by Markit in the response
	if response.ExceptionType != "" {
		str := "Exception Response from MarkitChartAPI"
		if response.Message != "" {
			response.Message = strings.Join([]string{`"`, response.Message, `"`}, "")
			str = strings.Join([]string{str, response.Message}, ": ")
		}
		if response.Details != "" {
			response.Details = strings.Join([]string{`"`, response.Details, `"`}, "")
			str = strings.Join([]string{str, response.Details}, " - ")
		}
		return nil, errors.New(str)
	}

	if response.Positions == nil {
		return nil, errors.New("No data")
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
	StartDate    string `json:",omitempty"`
	EndDate      string `json:",omitempty"`
	NumberOfDays int    `json:",omitempty"`
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

func (ds *Dataseries) String() string {
	str := "{"
	if ds.Open != nil {
		str = fmt.Sprintf(`%sopen:%s, `, str, ds.Open)
	}
	if ds.Close != nil {
		str = fmt.Sprintf(`%sclose:%s, `, str, ds.Close)
	}
	if ds.High != nil {
		str = fmt.Sprintf(`%shigh:%s, `, str, ds.High)
	}
	if ds.Low != nil {
		str = fmt.Sprintf(`%slow:%s, `, str, ds.Low)
	}
	if ds.Volume != nil {
		str = fmt.Sprintf(`%svolume:%s, `, str, ds.Volume)
	}

	// return empty string if no data has been added
	if len(str) <= 1 {
		return ""
	}
	str = str[:len(str)-2] // remove trailing comma and space

	return fmt.Sprintf(`%s}`, str)
}

func (d *Data) String() string {
	str := "{"
	if d.Max != 0 {
		str = fmt.Sprintf(`%smax:%f, `, str, d.Max)
	}
	if d.Min != 0 {
		str = fmt.Sprintf(`%smin:%f, `, str, d.Min)
	}
	if d.MaxDate != nil {
		str = fmt.Sprintf(`%smaxDate:%s, `, str, d.MaxDate)
	}
	if d.MinDate != nil {
		str = fmt.Sprintf(`%sminDate:%s, `, str, d.MinDate)
	}
	if d.Values != nil {
		str = fmt.Sprintf(`%svalues:%v, `, str, d.Values)
	}

	// return empty string if no data has been added
	if len(str) <= 1 {
		return ""
	}
	str = str[:len(str)-2] // remove trailing comma and space

	return fmt.Sprintf(`%s}`, str)
}

// Markit API response format
type MarkitChartAPIResponse struct {
	Labels         *MarkitChartAPIResponseLabels
	Positions      []float32
	Dates          []ISOTime
	Elements       []Element
	ExceptionType  string `json:",omitempty"`
	Message        string `json:",omitempty"`
	Details        string `json:",omitempty"`
	InnerException string `json:",omitempty"`
}
type MarkitChartAPIResponseLabels struct {
	Dates      []string `json:"dates"`
	Pos        []string `json:"pos"`
	Priorities []string `json:"priorities"`
	Text       []string `json:"text"`
	UtcDates   []string `json:"utcdates"`
}

type DataType int

const (
	Open DataType = iota
	High
	Low
	Close
	// Volume
)

func (response *MarkitChartAPIResponse) GetSpan() Span {
	def := Close
	return response.GetSpanForDataType(def)
}

func (response *MarkitChartAPIResponse) GetSpanForDataType(dt DataType) Span {
	foundPrice := false
	var priceElem Element
	for _, elem := range response.Elements {
		// we only care about the price element
		if elem.Type != "price" {
			continue
		}
		priceElem = elem
		foundPrice = true
	}

	if !foundPrice {
		// price element not found, return empty span
		return Span{}
	}

	var data *Data
	switch dt {
	case Open:
		data = priceElem.Dataseries.Open
	case High:
		data = priceElem.Dataseries.High
	case Low:
		data = priceElem.Dataseries.Low
	case Close:
		data = priceElem.Dataseries.Close
		// case Volume:
		// 	data := priceElem.Dataseries.Volume // I'm not sure this one works, I think we need the elem.Type = "volume"
	}

	span := Span{}
	for i := 0; i < len(data.Values); i++ {
		value := data.Values[i]
		time := response.Dates[i].UTC() // not sure this is kosher, but it converts to time.Time type...
		m := Measure{Time: time, Value: value}
		span = append(span, m)
	}

	return span
}

func (reponse *MarkitChartAPIResponse) String() string {
	json, err := json.Marshal(reponse)
	if err != nil {
		return err.Error()
	}
	return string(json)
}

/*  - - - - - ISOTime  - - - - - */

// ISOTime extensions to time.Time including JSON (Un)Marshaling
type ISOTime struct {
	time.Time
}

const ISOFormat = "2006-01-02T15:04:05-00"

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

	// "-00" is often truncated by Markit. Make sure it's added if not present
	if string(s[len(s)-3]) != "-" {
		s = fmt.Sprintf("%s-00", s)
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
