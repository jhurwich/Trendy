// Copyright 2015 Jordan Hurwich - no license granted

package stock_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jhurwich/trendy/stock"
	"github.com/jhurwich/trendy/testhelpers"
)

/* Tests */

// ensure the responses returned from Markit are in the expected format
func TestMarkitChartAPIResponse(t *testing.T) {

	t.Skip("Skipping MarkitChartAPI confirmation to avoid network delays")

	for _, test := range testhelpers.MarkitTestData {
		request, err := stock.NewMarkitChartAPIRequest(
			stock.NewStock(test.Sym),
			test.StartDate,
			test.EndDate)
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
		if string(contents) != test.ExpectedResponseBody {
			t.Errorf("Formatting change in MarkitResponse expected:\n%s\ngot:\n%s\n",
				test.ExpectedResponseBody,
				string(contents))
		}
		t.Logf("Test complete: %s, %s to %s", test.Sym, test.StartDate.String(), test.EndDate.String())
	}
}

// handle various http responses (404, 200 etc.)
func TestMarkitChartAPIIntegration(t *testing.T) {
	// run test cases
	testdata := testhelpers.MarkitTestData
	for _, test := range testdata {
		// initialize request
		request, err := stock.NewMarkitChartAPIRequest(
			stock.NewStock(test.Sym),
			test.StartDate,
			test.EndDate)
		if err != nil {
			t.Errorf("Could not create a MarkitChartAPIRequest: %v", err)
		}
		actualUrl := request.Url // save the actualUrl for the request

		// start test server with success status
		tsParams := testhelpers.TestServer{
			Status: 200, RequestUrl: actualUrl, TestData: testdata, T: t,
		}
		ts := httptest.NewServer(&tsParams)

		request.Url = ts.URL // override request URL to hit test server

		// test five most common error responses
		var errorCodes = []int{500, 404, 403, 400, 401}
		for _, errorCode := range errorCodes {
			tsParams = testhelpers.TestServer{
				Status: errorCode, RequestUrl: actualUrl, TestData: testdata, T: t,
			}

			response, err := request.Request()

			if err == nil {
				t.Errorf("Expected an error and got success (!?), response:\n%v", response)
			} else if !strings.Contains(err.Error(), strconv.Itoa(errorCode)) {
				t.Errorf("Expected %d, got... something else: %v", errorCode, err)
			}
		}

		// switch test server to success response and make request
		tsParams = testhelpers.TestServer{
			Status: 200, RequestUrl: actualUrl, TestData: testdata, T: t,
		}
		response, err := request.Request()

		if test.ExpectError && err == nil {
			t.Errorf("Expected error response but got success: %v", response)
		} else if !test.ExpectError && err != nil {
			t.Errorf("Expected successful response but got error: %v", err)
		} else if !test.ExpectError {
			// expected success and got a response
			// compare response with expected
			if !CompareMarkitChartAPIResponses(&test.ExpectedMarkitResponse, response) {
				t.Errorf("Response from Markit differs from expected, expected:\n%v\ngot:\n%v\n", test.ExpectedMarkitResponse, response)
			}
		}

		t.Logf("Test complete: %s, %s to %s", test.Sym, test.StartDate.String(), test.EndDate.String())
	}
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
