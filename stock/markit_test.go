// Copyright 2015 Jordan Hurwich - no license granted

package stock_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jhurwich/trendy/stock"
)

/* Tests */

// ensure the responses returned from Markit are in the expected format
func TestMarkitChartAPIResponse(t *testing.T) {

	t.Skip("Skipping MarkitChartAPI confirmation to avoid network delays")

	for _, test := range markitTestData {
		request, err := stock.NewMarkitChartAPIRequest(
			stock.NewStock(test.sym),
			test.startDate,
			test.endDate)
		if err != nil {
			t.Errorf("Could not create a MarkitChartAPIRequest: %v", err)
		}

		time.Sleep(1500 * time.Millisecond) // don't overload the Markit server

		// don't use request.Request() so that we can look at the response body
		r, err := http.Get(request.Url)
		if err != nil {
			t.Errorf("Request to MarkitChartAPI failed: %v", err)
		}
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			t.Errorf("Request to MarkitChartAPI failed (Status-%d)", r.StatusCode)
		}

		// compare the response body to the expected value
		contents, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Could not validate MarkitChartAPI response body: %v", err)
		}
		if string(contents) != test.expectedResponseBody {
			t.Errorf("Formatting change in MarkitResponse expected:\n%s\ngot:\n%s\n",
				test.expectedResponseBody,
				string(contents))
		}
		t.Logf("Test complete: %s, %s to %s", test.sym, test.startDate.String(), test.endDate.String())
	}
}

// testServer for integration test
type testServer struct {
	status     int
	requestUrl string
}

func (ts *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(int(ts.status))

	// if we're not modeling success, we're done with this request
	if ts.status != http.StatusOK {
		return
	}

	// parse parameters from requestUrl if set
	var params stock.MarkitChartAPIRequestParams // we'll marshal into this
	paramsSplit := strings.Split(ts.requestUrl, "?parameters=")
	if len(paramsSplit) <= 1 {
		w.WriteHeader(666) // could not parse parameters, this will cause an error
		return
	}
	err := json.Unmarshal([]byte(paramsSplit[1]), &params)
	if err != nil {
		w.WriteHeader(667) // could not parse paramStr into object, this will cause an error
		return
	}
	symbol := params.Elements[0].Symbol
	startDate, err := time.Parse(stock.ISOFormat, params.StartDate)
	endDate, err := time.Parse(stock.ISOFormat, params.EndDate)
	if err != nil {
		w.WriteHeader(668) // could not parse time into object, this will cause an error
		return
	}

	// Now we need to write in the correct response body by finding the test
	// we're currently doing. This will return and terminate if the test is found.
	for _, test := range markitTestData {
		if test.sym == symbol && test.startDate == startDate && test.endDate == endDate {
			fmt.Fprintln(w, test.expectedResponseBody) // once found, write the response
			return
		}
	}

	w.WriteHeader(669) // could not find the right response, this will cause an error
}

// handle various http responses (404, 200 etc.)
func TestMarkitChartAPIIntegration(t *testing.T) {
	var tests = markitTestData

	// run test cases
	for _, test := range tests {
		// initialize request
		request, err := stock.NewMarkitChartAPIRequest(
			stock.NewStock(test.sym),
			test.startDate,
			test.endDate)
		if err != nil {
			t.Errorf("Could not create a MarkitChartAPIRequest: %v", err)
		}
		actualUrl := request.Url // save the actualUrl for the request

		// start test server with success status
		tsParams := testServer{200, actualUrl}
		ts := httptest.NewServer(&tsParams)

		request.Url = ts.URL // override request URL to hit test server

		// test five most common error responses
		var errorCodes = []int{500, 404, 403, 400, 401}
		for _, errorCode := range errorCodes {
			tsParams = testServer{errorCode, actualUrl}

			response, err := request.Request()

			if err == nil {
				t.Errorf("Expected an error and got success (!?), response:\n%v", response)
			} else if !strings.Contains(err.Error(), strconv.Itoa(errorCode)) {
				t.Errorf("Expected %d, got... something else: %v", errorCode, err)
			}
		}

		// switch test server to success response and make request
		tsParams = testServer{200, actualUrl}
		response, err := request.Request()

		if test.expectError && err == nil {
			t.Errorf("Expected error response but got success: %v", response)
		} else if !test.expectError && err != nil {
			t.Errorf("Expected successful response but got error: %v", err)
		} else if !test.expectError {
			// expected success and got a response
			// compare response with expected
			if !CompareMarkitChartAPIResponses(&test.expectedMarkitResponse, response) {
				t.Errorf("Response from Markit differs from expected, expected:\n%v\ngot:\n%v\n", test.expectedMarkitResponse, response)
			}
		}

		t.Logf("Test complete: %s, %s to %s", test.sym, test.startDate.String(), test.endDate.String())
	}
}

/* Constants for Tests */

var markitTestData = []struct {
	sym                    string
	startDate              time.Time
	endDate                time.Time
	expectedResponseBody   string
	expectedMarkitResponse stock.MarkitChartAPIResponse
	expectError            bool
}{
	{"AMZN", arbitraryDate, arbitraryDate.AddDate(0, 0, 30), amznArbitrary30SavedResponseBody, amznArbitrary30MarkitResponse, false},
	{"MSFT", arbitraryDate, arbitraryDate.AddDate(0, 0, 300), msftArbitrary300SavedResponseBody, msftArbitrary300MarkitResponse, false},
	{"MSFT", arbitraryDate, arbitraryDate, startEndErrorResponseBody, stock.MarkitChartAPIResponse{}, true},           // Error response for bad dates, start == end
	{"MSFT", arbitraryDate.AddDate(0, 0, 30), arbitraryDate, emptyResponseBody, stock.MarkitChartAPIResponse{}, true}, // Markit sends empty date if startDate is after endDate, this is an error case
}

// an arbitrary date in the past to standardize which data is saved for comparison.
var arbitraryDate time.Time = time.Date(2011, time.May, 20, 12, 0, 0, 0, time.UTC) // "Thu, 05/20/11, 12:00"

// Saved Markit Chart API responses for comparison against a new Markit request
const amznArbitrary30SavedResponseBody string = `{"Labels":null,"Positions":[0,0.05,0.1,0.15,0.2,0.25,0.35,0.4,0.45,0.5,0.55,0.6,0.65,0.7,0.75,0.8,0.85,0.9,0.95,1],"Dates":["2011-05-20T00:00:00","2011-05-23T00:00:00","2011-05-24T00:00:00","2011-05-25T00:00:00","2011-05-26T00:00:00","2011-05-27T00:00:00","2011-05-31T00:00:00","2011-06-01T00:00:00","2011-06-02T00:00:00","2011-06-03T00:00:00","2011-06-06T00:00:00","2011-06-07T00:00:00","2011-06-08T00:00:00","2011-06-09T00:00:00","2011-06-10T00:00:00","2011-06-13T00:00:00","2011-06-14T00:00:00","2011-06-15T00:00:00","2011-06-16T00:00:00","2011-06-17T00:00:00"],"Elements":[{"Currency":"USD","TimeStamp":null,"Symbol":"AMZN","Type":"price","DataSeries":{"open":{"min":185.72,"max":197.95,"maxDate":"2011-05-20T00:00:00","minDate":"2011-06-07T00:00:00","values":[197.95,195.56,197,193.57,191.24,194.76,195.94,196.06,192.28,191.23,188.01,185.72,187.45,189.74,189.25,186.81,188.99,188.04,185.74,186.51]},"high":{"min":187,"max":199.8,"maxDate":"2011-05-20T00:00:00","minDate":"2011-06-16T00:00:00","values":[199.8,197.29,197,194.35,196.45,196.12,198.44,197.26,194.44,193.21,189.85,190.63,189.81,191.76,190.77,189.31,190.72,192.45,187,187.39]},"low":{"min":181.59,"max":197.24,"maxDate":"2011-05-20T00:00:00","minDate":"2011-06-16T00:00:00","values":[197.24,192.02,193,191.14,190.88,193.5,195.03,192.05,190.56,187.62,185.18,185.52,186.32,185.71,186.28,184.86,187.07,185.3,181.59,184.64]},"close":{"min":183.65,"max":198.65,"maxDate":"2011-05-20T00:00:00","minDate":"2011-06-16T00:00:00","values":[198.65,196.22,193.27,192.26,195,194.13,196.69,192.395,193.65,188.32,185.69,187.55,188.05,189.68,186.53,186.29,189.96,185.98,183.65,186.37]}}},{"Currency":"USD","TimeStamp":null,"Symbol":"AMZN","Type":"volume","DataSeries":{"volume":{"min":2353956,"max":6328369,"maxDate":"2011-06-17T00:00:00","minDate":"2011-05-27T00:00:00","values":[3382021,4230523,2972667,4661207,4075276,2353956,3412945,3449408,3045560,4975621,3713215,4867038,3717299,4187248,3763319,3870110,3960596,6318168,6032134,6328369]}}}]}`

var amznArbitrary30MarkitResponse = stock.MarkitChartAPIResponse{
	Labels:    nil,
	Positions: []float32{0, 0.05, 0.1, 0.15, 0.2, 0.25, 0.35, 0.4, 0.45, 0.5, 0.55, 0.6, 0.65, 0.7, 0.75, 0.8, 0.85, 0.9, 0.95, 1},
	Dates:     makeISOTimeArray(arbitraryDate, arbitraryDate.AddDate(0, 0, 30)),
	Elements:  nil, // for now, Elements is not checked - TODO(jhurwich) check Elements
}

const msftArbitrary300SavedResponseBody string = `{"Labels":null,"Positions":[0,0.005,0.009,0.014,0.019,0.023,0.033,0.037,0.042,0.047,0.051,0.056,0.061,0.065,0.07,0.075,0.079,0.084,0.089,0.093,0.098,0.103,0.107,0.112,0.117,0.121,0.126,0.131,0.136,0.14,0.15,0.154,0.159,0.164,0.168,0.173,0.178,0.182,0.187,0.192,0.196,0.201,0.206,0.21,0.215,0.22,0.224,0.229,0.234,0.238,0.243,0.248,0.252,0.257,0.262,0.266,0.271,0.276,0.28,0.285,0.29,0.294,0.299,0.304,0.308,0.313,0.318,0.322,0.327,0.332,0.336,0.341,0.346,0.35,0.36,0.364,0.369,0.374,0.379,0.383,0.388,0.393,0.397,0.402,0.407,0.411,0.416,0.421,0.425,0.43,0.435,0.439,0.444,0.449,0.453,0.458,0.463,0.467,0.472,0.477,0.481,0.486,0.491,0.495,0.5,0.505,0.509,0.514,0.519,0.523,0.528,0.533,0.537,0.542,0.547,0.551,0.556,0.561,0.565,0.57,0.575,0.579,0.584,0.589,0.593,0.598,0.603,0.607,0.612,0.617,0.621,0.631,0.636,0.64,0.645,0.65,0.654,0.659,0.664,0.668,0.673,0.678,0.682,0.687,0.692,0.696,0.701,0.706,0.71,0.715,0.72,0.724,0.734,0.738,0.743,0.748,0.757,0.762,0.766,0.771,0.776,0.78,0.785,0.79,0.794,0.804,0.808,0.813,0.818,0.822,0.827,0.832,0.836,0.841,0.846,0.85,0.855,0.86,0.864,0.869,0.874,0.879,0.883,0.888,0.893,0.897,0.902,0.907,0.911,0.921,0.925,0.93,0.935,0.939,0.944,0.949,0.953,0.958,0.963,0.967,0.972,0.977,0.981,0.986,0.991,0.995,1],"Dates":["2011-05-20T00:00:00","2011-05-23T00:00:00","2011-05-24T00:00:00","2011-05-25T00:00:00","2011-05-26T00:00:00","2011-05-27T00:00:00","2011-05-31T00:00:00","2011-06-01T00:00:00","2011-06-02T00:00:00","2011-06-03T00:00:00","2011-06-06T00:00:00","2011-06-07T00:00:00","2011-06-08T00:00:00","2011-06-09T00:00:00","2011-06-10T00:00:00","2011-06-13T00:00:00","2011-06-14T00:00:00","2011-06-15T00:00:00","2011-06-16T00:00:00","2011-06-17T00:00:00","2011-06-20T00:00:00","2011-06-21T00:00:00","2011-06-22T00:00:00","2011-06-23T00:00:00","2011-06-24T00:00:00","2011-06-27T00:00:00","2011-06-28T00:00:00","2011-06-29T00:00:00","2011-06-30T00:00:00","2011-07-01T00:00:00","2011-07-05T00:00:00","2011-07-06T00:00:00","2011-07-07T00:00:00","2011-07-08T00:00:00","2011-07-11T00:00:00","2011-07-12T00:00:00","2011-07-13T00:00:00","2011-07-14T00:00:00","2011-07-15T00:00:00","2011-07-18T00:00:00","2011-07-19T00:00:00","2011-07-20T00:00:00","2011-07-21T00:00:00","2011-07-22T00:00:00","2011-07-25T00:00:00","2011-07-26T00:00:00","2011-07-27T00:00:00","2011-07-28T00:00:00","2011-07-29T00:00:00","2011-08-01T00:00:00","2011-08-02T00:00:00","2011-08-03T00:00:00","2011-08-04T00:00:00","2011-08-05T00:00:00","2011-08-08T00:00:00","2011-08-09T00:00:00","2011-08-10T00:00:00","2011-08-11T00:00:00","2011-08-12T00:00:00","2011-08-15T00:00:00","2011-08-16T00:00:00","2011-08-17T00:00:00","2011-08-18T00:00:00","2011-08-19T00:00:00","2011-08-22T00:00:00","2011-08-23T00:00:00","2011-08-24T00:00:00","2011-08-25T00:00:00","2011-08-26T00:00:00","2011-08-29T00:00:00","2011-08-30T00:00:00","2011-08-31T00:00:00","2011-09-01T00:00:00","2011-09-02T00:00:00","2011-09-06T00:00:00","2011-09-07T00:00:00","2011-09-08T00:00:00","2011-09-09T00:00:00","2011-09-12T00:00:00","2011-09-13T00:00:00","2011-09-14T00:00:00","2011-09-15T00:00:00","2011-09-16T00:00:00","2011-09-19T00:00:00","2011-09-20T00:00:00","2011-09-21T00:00:00","2011-09-22T00:00:00","2011-09-23T00:00:00","2011-09-26T00:00:00","2011-09-27T00:00:00","2011-09-28T00:00:00","2011-09-29T00:00:00","2011-09-30T00:00:00","2011-10-03T00:00:00","2011-10-04T00:00:00","2011-10-05T00:00:00","2011-10-06T00:00:00","2011-10-07T00:00:00","2011-10-10T00:00:00","2011-10-11T00:00:00","2011-10-12T00:00:00","2011-10-13T00:00:00","2011-10-14T00:00:00","2011-10-17T00:00:00","2011-10-18T00:00:00","2011-10-19T00:00:00","2011-10-20T00:00:00","2011-10-21T00:00:00","2011-10-24T00:00:00","2011-10-25T00:00:00","2011-10-26T00:00:00","2011-10-27T00:00:00","2011-10-28T00:00:00","2011-10-31T00:00:00","2011-11-01T00:00:00","2011-11-02T00:00:00","2011-11-03T00:00:00","2011-11-04T00:00:00","2011-11-07T00:00:00","2011-11-08T00:00:00","2011-11-09T00:00:00","2011-11-10T00:00:00","2011-11-11T00:00:00","2011-11-14T00:00:00","2011-11-15T00:00:00","2011-11-16T00:00:00","2011-11-17T00:00:00","2011-11-18T00:00:00","2011-11-21T00:00:00","2011-11-22T00:00:00","2011-11-23T00:00:00","2011-11-25T00:00:00","2011-11-28T00:00:00","2011-11-29T00:00:00","2011-11-30T00:00:00","2011-12-01T00:00:00","2011-12-02T00:00:00","2011-12-05T00:00:00","2011-12-06T00:00:00","2011-12-07T00:00:00","2011-12-08T00:00:00","2011-12-09T00:00:00","2011-12-12T00:00:00","2011-12-13T00:00:00","2011-12-14T00:00:00","2011-12-15T00:00:00","2011-12-16T00:00:00","2011-12-19T00:00:00","2011-12-20T00:00:00","2011-12-21T00:00:00","2011-12-22T00:00:00","2011-12-23T00:00:00","2011-12-27T00:00:00","2011-12-28T00:00:00","2011-12-29T00:00:00","2011-12-30T00:00:00","2012-01-03T00:00:00","2012-01-04T00:00:00","2012-01-05T00:00:00","2012-01-06T00:00:00","2012-01-09T00:00:00","2012-01-10T00:00:00","2012-01-11T00:00:00","2012-01-12T00:00:00","2012-01-13T00:00:00","2012-01-17T00:00:00","2012-01-18T00:00:00","2012-01-19T00:00:00","2012-01-20T00:00:00","2012-01-23T00:00:00","2012-01-24T00:00:00","2012-01-25T00:00:00","2012-01-26T00:00:00","2012-01-27T00:00:00","2012-01-30T00:00:00","2012-01-31T00:00:00","2012-02-01T00:00:00","2012-02-02T00:00:00","2012-02-03T00:00:00","2012-02-06T00:00:00","2012-02-07T00:00:00","2012-02-08T00:00:00","2012-02-09T00:00:00","2012-02-10T00:00:00","2012-02-13T00:00:00","2012-02-14T00:00:00","2012-02-15T00:00:00","2012-02-16T00:00:00","2012-02-17T00:00:00","2012-02-21T00:00:00","2012-02-22T00:00:00","2012-02-23T00:00:00","2012-02-24T00:00:00","2012-02-27T00:00:00","2012-02-28T00:00:00","2012-02-29T00:00:00","2012-03-01T00:00:00","2012-03-02T00:00:00","2012-03-05T00:00:00","2012-03-06T00:00:00","2012-03-07T00:00:00","2012-03-08T00:00:00","2012-03-09T00:00:00","2012-03-12T00:00:00","2012-03-13T00:00:00","2012-03-14T00:00:00","2012-03-15T00:00:00"],"Elements":[{"Currency":"USD","TimeStamp":null,"Symbol":"MSFT","Type":"price","DataSeries":{"open":{"min":23.75,"max":32.79,"maxDate":"2012-03-15T00:00:00","minDate":"2011-06-16T00:00:00","values":[24.72,24.21,24.2,24.17,24.35,24.68,24.96,24.99,24.49,24.05,23.89,24.09,23.9,24.01,24.02,23.79,24.3,24,23.75,24.22,24.17,24.52,24.6,24.44,24.51,24.23,25.3,25.71,25.74,25.93,26.1,25.97,26.49,26.54,26.62,26.55,26.6,26.62,26.47,26.63,26.81,27.28,27.04,26.86,27.26,27.82,27.88,27.29,27.52,27.51,26.98,26.83,26.53,25.97,25.02,24.71,24.95,24.5,25.13,25.24,25.215,25.25,24.57,24.41,24.42,24.03,24.65,25.08,24.51,25.53,25.73,26.29,26.46,25.78,25.2,25.69,26,26,25.44,25.92,26.17,26.73,27.05,26.8,27.31,27.05,25.3,24.9,25.19,25.66,25.93,25.98,25.2,24.72,24.3,25.42,25.9,26.34,26.58,26.86,27.18,26.76,27.31,27.114,26.94,27.37,27.26,27.15,27.06,27.08,27.03,27.13,27.14,26.755,26.19,26.1,26.24,26.38,26.21,27.01,26.59,26.47,26.58,26.88,26.56,26.47,26.01,25.48,25.24,24.89,24.61,24.38,24.94,24.82,25.37,25.56,25.59,25.78,25.81,25.67,25.48,25.52,25.41,25.75,25.72,25.72,25.67,26.02,25.86,26.01,25.82,25.91,25.96,26.11,25.95,26,26.55,26.82,27.38,27.53,28.05,27.93,27.43,27.87,27.93,28.4,28.31,28.16,28.82,29.55,29.47,29.07,29.61,29.45,28.97,29.66,29.79,29.9,30.14,30.04,30.15,30.26,30.68,30.64,30.63,30.33,30.33,30.31,31.2,31.18,31.45,31.2,31.48,31.24,31.41,31.885,31.93,32.31,32.01,31.54,31.67,32.04,32.1,31.97,32.24,32.53,32.79]},"high":{"min":24.01,"max":32.94,"maxDate":"2012-03-15T00:00:00","minDate":"2011-06-15T00:00:00","values":[24.87,24.25,24.29,24.31,25.03,24.9,25.06,25.1,24.65,24.14,24.25,24.17,24.02,24.04,24.02,24.19,24.45,24.01,24.1,24.3,24.66,24.86,24.81,24.65,24.54,25.46,25.92,25.71,26,26.17,26.15,26.37,26.88,26.98,26.8,26.79,26.96,27.01,26.93,26.9,27.64,27.35,27.31,27.55,28.09,28.145,27.985,28.07,27.71,27.685,27.45,27,26.87,26.1,25.6,25.62,25.09,25.38,25.335,25.58,25.59,25.7,25.09,24.62,24.49,24.75,24.93,25.16,25.34,25.86,26.43,26.71,26.86,26,25.59,26,26.66,26.18,25.93,26.185,26.8,27.03,27.27,27.31,27.5,27.06,25.65,25.15,25.52,25.92,26.37,26.17,25.5,25.335,25.39,26.16,26.4,26.51,26.97,27.07,27.31,27.2,27.5,27.42,27.4,27.47,27.34,27.19,27.4,27.23,27.06,27.4,27.19,27,26.32,26.2,26.585,26.4,26.82,27.2,26.75,26.5,27.075,27,26.94,26.51,26.04,25.5,25.25,24.96,24.79,24.67,24.97,25.04,25.585,25.63,25.62,25.8,25.87,25.76,25.72,25.87,25.57,26.1,25.86,25.88,26.17,26.12,26.1,26.19,25.86,26.04,26.14,26.15,26.05,26.12,26.96,27.47,27.728,28.19,28.1,28.15,27.98,28.02,28.25,28.65,28.4,28.435,29.74,29.95,29.57,29.65,29.7,29.53,29.62,29.7,30.05,30.17,30.4,30.22,30.485,30.67,30.8,30.8,30.77,30.46,30.39,31.55,31.32,31.61,31.68,31.59,31.5,31.5,31.93,32,32.39,32.44,32.05,31.98,31.92,32.21,32.16,32.2,32.69,32.88,32.94]},"low":{"min":23.65,"max":32.58,"maxDate":"2012-03-15T00:00:00","minDate":"2011-06-16T00:00:00","values":[24.44,24.03,24.04,24.16,24.32,24.65,24.7,24.37,24.18,23.84,23.77,23.9,23.86,23.82,23.69,23.7,24.19,23.67,23.65,23.98,24.16,24.4,24.59,24.2,24.19,24.23,25.16,25.36,25.66,25.84,25.9,25.96,26.36,26.51,26.49,26.34,26.51,26.36,26.47,26.26,26.78,26.98,26.65,26.68,27.19,27.78,27.2,27.21,27.26,26.75,26.76,26.48,25.93,25.23,24.39,24.03,24.1,24.4,24.65,25.15,25.05,24.93,24.03,23.91,23.79,24.03,24.42,24.5,24.42,25.37,25.7,26.26,26.21,25.66,25.11,25.57,25.95,25.5,25.27,25.81,25.89,26.31,26.83,26.6,26.93,25.97,24.6,24.69,24.73,25.45,25.51,25.09,24.88,24.52,24.26,25.16,25.7,26.2,26.47,26.72,26.9,26.62,27.02,26.85,26.8,27.01,26.4,26.8,27.04,26.72,26.1,26.65,26.79,26.62,25.86,25.7,25.98,26,26.13,26.685,26.06,26.12,26.57,26.65,26.4,26.04,25.44,25.15,24.9,24.65,24.47,24.3,24.69,24.75,25.14,25.2,25.16,25.5,25.61,25.335,25.37,25.5,25.29,25.651,25.57,25.54,25.63,25.46,25.81,25.44,25.475,25.73,25.93,25.76,25.86,25.91,26.39,26.78,27.29,27.525,27.72,27.75,27.37,27.645,27.79,28.17,27.97,28.03,28.75,29.35,29.18,29.07,29.4,29.17,28.83,29.23,29.76,29.71,30.09,29.97,30.05,30.22,30.48,30.36,30.43,29.85,30.03,30.3,30.95,31.15,31.18,31,31.24,31.1,31.38,31.61,31.85,32,31.62,31.49,31.53,31.9,31.92,31.82,32.15,32.49,32.58]},"close":{"min":23.705,"max":32.85,"maxDate":"2012-03-15T00:00:00","minDate":"2011-06-10T00:00:00","values":[24.49,24.17,24.15,24.19,24.67,24.76,25.01,24.43,24.22,23.905,24.01,24.06,23.94,23.96,23.705,24.04,24.22,23.74,23.995,24.26,24.47,24.76,24.65,24.63,24.3,25.2,25.8,25.62,26,26.02,26.03,26.33,26.77,26.92,26.63,26.54,26.63,26.47,26.78,26.59,27.54,27.06,27.095,27.53,27.91,28.08,27.33,27.72,27.4,27.27,26.8,26.92,25.94,25.68,24.48,25.58,24.2,25.19,25.1,25.51,25.35,25.245,24.67,24.05,23.98,24.72,24.9,24.57,25.25,25.84,26.23,26.6,26.21,25.8,25.51,26,26.22,25.74,25.89,26.04,26.5,26.99,27.12,27.21,26.98,25.99,25.06,25.06,25.44,25.67,25.575,25.45,24.89,24.53,25.34,25.89,26.34,26.25,26.94,27,26.96,27.18,27.27,26.98,27.31,27.13,27.04,27.16,27.19,26.81,26.59,27.25,26.98,26.63,25.99,26.01,26.53,26.25,26.8,27.16,26.2,26.28,26.91,26.76,26.74,26.07,25.54,25.3,25,24.79,24.47,24.3,24.87,24.84,25.58,25.28,25.22,25.7,25.66,25.6,25.4,25.7,25.51,25.76,25.59,25.56,26,25.53,26.025,25.76,25.81,26.03,26.04,25.82,26.02,25.96,26.765,27.4,27.68,28.105,27.74,27.84,27.72,28,28.25,28.255,28.23,28.12,29.71,29.73,29.34,29.56,29.5,29.23,29.61,29.53,29.89,29.95,30.24,30.2,30.35,30.66,30.77,30.495,30.58,30.25,30.05,31.285,31.25,31.44,31.27,31.37,31.48,31.35,31.87,31.74,32.29,32.075,31.8,31.555,31.84,32.01,31.99,32.04,32.67,32.77,32.85]}}},{"Currency":"USD","TimeStamp":null,"Symbol":"MSFT","Type":"volume","DataSeries":{"volume":{"min":21287332,"max":165902897,"maxDate":"2012-01-20T00:00:00","minDate":"2011-12-27T00:00:00","values":[45451462,52703425,47692976,34903112,78016538,50254323,60196203,74036467,51487738,60697662,54778670,41112524,42206806,42882262,49327104,47574074,42902699,49410128,57190388,83352877,54344045,49712104,44290814,59472060,101387157,92044114,81032018,66052078,52536288,52914516,37803059,48748894,51950943,58332434,44000715,47320989,40869336,46385928,49134401,44506711,86730600,49795352,81737342,76380505,108486590,74643391,71492139,83766901,104394739,61846218,63883018,64583238,92953765,112072491,134257113,126278864,127819718,90697205,64791784,56529388,54256723,50923682,105715509,77402319,54720967,59671022,45329610,48191924,71959200,38863136,57341367,59301724,60511548,43897065,54931986,41960917,65818212,64531339,55047244,48794446,66742534,67809210,89685212,52324841,49211857,72750701,96285920,64769019,51057571,55623705,60740399,63411976,54086654,64596171,83485396,94061244,55113496,52748451,41822239,38826791,52493454,43830076,50949439,39453241,52491969,42881648,76300104,76620533,56897791,53554554,63029830,74515622,57712077,46798951,61186956,53536398,65837011,36553269,42586043,47825636,62950825,32517281,37903971,34199146,43877075,53262743,70977495,47627157,61882819,49204488,49105287,26164410,46771878,40920907,81353522,48545338,52295245,56818367,46175294,62669835,60522185,53790403,38945867,54581003,47927107,46217486,101410082,52258284,60767523,64134140,35794085,23205776,21287332,29823501,22616883,27396333,64735391,80519402,56082205,99459469,59708266,60014333,65586477,49375477,60204902,72395252,64860509,74053427,165902897,76081814,51711367,59236267,49107458,44190573,51114661,50572372,67413817,52226255,41845397,28040378,39242529,49662740,50481549,44606751,33322516,59662711,43316117,94705078,70040830,50832547,49253117,35035609,35577833,34575391,45230573,59326545,77348930,47318927,45239832,51938950,34340619,36752011,34628398,34076755,48951650,41987743,49070794]}}}]}`

var msftArbitrary300MarkitResponse = stock.MarkitChartAPIResponse{
	Labels:    nil,
	Positions: []float32{0, 0.005, 0.009, 0.014, 0.019, 0.023, 0.033, 0.037, 0.042, 0.047, 0.051, 0.056, 0.061, 0.065, 0.07, 0.075, 0.079, 0.084, 0.089, 0.093, 0.098, 0.103, 0.107, 0.112, 0.117, 0.121, 0.126, 0.131, 0.136, 0.14, 0.15, 0.154, 0.159, 0.164, 0.168, 0.173, 0.178, 0.182, 0.187, 0.192, 0.196, 0.201, 0.206, 0.21, 0.215, 0.22, 0.224, 0.229, 0.234, 0.238, 0.243, 0.248, 0.252, 0.257, 0.262, 0.266, 0.271, 0.276, 0.28, 0.285, 0.29, 0.294, 0.299, 0.304, 0.308, 0.313, 0.318, 0.322, 0.327, 0.332, 0.336, 0.341, 0.346, 0.35, 0.36, 0.364, 0.369, 0.374, 0.379, 0.383, 0.388, 0.393, 0.397, 0.402, 0.407, 0.411, 0.416, 0.421, 0.425, 0.43, 0.435, 0.439, 0.444, 0.449, 0.453, 0.458, 0.463, 0.467, 0.472, 0.477, 0.481, 0.486, 0.491, 0.495, 0.5, 0.505, 0.509, 0.514, 0.519, 0.523, 0.528, 0.533, 0.537, 0.542, 0.547, 0.551, 0.556, 0.561, 0.565, 0.57, 0.575, 0.579, 0.584, 0.589, 0.593, 0.598, 0.603, 0.607, 0.612, 0.617, 0.621, 0.631, 0.636, 0.64, 0.645, 0.65, 0.654, 0.659, 0.664, 0.668, 0.673, 0.678, 0.682, 0.687, 0.692, 0.696, 0.701, 0.706, 0.71, 0.715, 0.72, 0.724, 0.734, 0.738, 0.743, 0.748, 0.757, 0.762, 0.766, 0.771, 0.776, 0.78, 0.785, 0.79, 0.794, 0.804, 0.808, 0.813, 0.818, 0.822, 0.827, 0.832, 0.836, 0.841, 0.846, 0.85, 0.855, 0.86, 0.864, 0.869, 0.874, 0.879, 0.883, 0.888, 0.893, 0.897, 0.902, 0.907, 0.911, 0.921, 0.925, 0.93, 0.935, 0.939, 0.944, 0.949, 0.953, 0.958, 0.963, 0.967, 0.972, 0.977, 0.981, 0.986, 0.991, 0.995, 1},
	Dates:     makeISOTimeArray(arbitraryDate, arbitraryDate.AddDate(0, 0, 300)),
	Elements:  nil, // for now, Elements is not checked - TODO(jhurwich) check Elements
}

const startEndErrorResponseBody string = `{"ExceptionType":"Exception","Message":"Could not determine the desired start and end points of chart.  One or more of the following values is incorrect: NumberOfDays, EndOffsetDays, StartDate, EndDate.","Details":"Internal.MODApis","InnerException":null}`
const emptyResponseBody string = `{"Labels":null,"Positions":null,"Dates":null,"Elements":[]}`
const overloadResponseBody string = `Request blockedExceeded requests/sec limit.`

/* Utils */

var Holidays20112012 []time.Time = []time.Time{
	time.Date(2011, time.January, 17, 12, 0, 0, 0, time.UTC),  // MLK day 2011
	time.Date(2011, time.February, 21, 12, 0, 0, 0, time.UTC), // Washington's Bday 2011
	time.Date(2011, time.April, 22, 12, 0, 0, 0, time.UTC),    // Good Friday 2011
	time.Date(2011, time.May, 30, 12, 0, 0, 0, time.UTC),      // Memorial Day 2011
	time.Date(2011, time.July, 4, 12, 0, 0, 0, time.UTC),      // Independence Day 2011
	time.Date(2011, time.September, 5, 12, 0, 0, 0, time.UTC), // Labor Day 2011
	time.Date(2011, time.November, 24, 12, 0, 0, 0, time.UTC), // Thanksgiving D 2011ay
	time.Date(2011, time.December, 26, 12, 0, 0, 0, time.UTC), // Christmas Day 2011
	time.Date(2012, time.January, 2, 12, 0, 0, 0, time.UTC),   // New year's day 2012
	time.Date(2012, time.January, 16, 12, 0, 0, 0, time.UTC),  // MLK day 2012
	time.Date(2012, time.February, 20, 12, 0, 0, 0, time.UTC), // Washington's Bday 2012
	time.Date(2012, time.April, 6, 12, 0, 0, 0, time.UTC),     // Good Friday 2012
	time.Date(2012, time.May, 28, 12, 0, 0, 0, time.UTC),      // Memorial Day 2012
	time.Date(2012, time.July, 4, 12, 0, 0, 0, time.UTC),      // Independence Day 2012
	time.Date(2012, time.September, 3, 12, 0, 0, 0, time.UTC), // Labor Day 2012
	time.Date(2012, time.November, 22, 12, 0, 0, 0, time.UTC), // Thanksgiving Day 2012
	time.Date(2012, time.December, 25, 12, 0, 0, 0, time.UTC), // Christmas Day 2012
}

func makeISOTimeArray(startDate time.Time, endDate time.Time) []stock.ISOTime {
	var timeArray []stock.ISOTime

	// Make sure startDate and endDate are in 2011/2012, those are the only years
	// that we have holiday information for. If not, return empy.
	if !(startDate.Year() == 2011 || startDate.Year() == 2012) && (endDate.Year() == 2011 || endDate.Year() == 2012) {
		return timeArray
	}

	var day stock.ISOTime = stock.ISOTime{startDate.AddDate(0, 0, -1)}
	for day.Before(endDate) {
		day = stock.ISOTime{day.AddDate(0, 0, 1)}

		// market is not open on Saturday or Sunday
		if day.Weekday() == time.Weekday(0) ||
			day.Weekday() == time.Weekday(6) {
			continue
		}

		// market is not open on certain holidays
		isHoliday := false
		for _, holiday := range Holidays20112012 {
			if day.Year() == holiday.Year() && day.YearDay() == holiday.YearDay() {
				isHoliday = true
				break
			}
		}

		if !isHoliday {
			timeArray = append(timeArray, day)
		}
	}
	return timeArray
}

// a quick comparison function for MarkitChartAPIResponses
// currently this just checks that the dates and positions values are the same
func CompareMarkitChartAPIResponses(r *stock.MarkitChartAPIResponse, l *stock.MarkitChartAPIResponse) bool {
	if len(r.Dates) != len(l.Dates) ||
		len(r.Positions) != len(l.Positions) ||
		len(r.Dates) != len(r.Positions) {
		return false
	}

	var rD, lD stock.ISOTime = r.Dates[0], l.Dates[0]
	var rP, lP = r.Positions[0], l.Positions[0]
	for i := 1; rD.Year() == lD.Year() && rD.YearDay() == lD.YearDay() && rP == lP; i++ {
		if i >= len(r.Dates) {
			return true
		}
		rD, lD = r.Dates[i], l.Dates[i]
		rP, lP = r.Positions[i], l.Positions[i]
	}
	return false
}
