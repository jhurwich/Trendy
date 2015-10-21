// Copyright 2015 Jordan Hurwich - no license granted

package stock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhurwich/trendy/stock"
	"github.com/jhurwich/trendy/testhelpers"
)

func TestStockConstruction(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		sym string
	}{
		{"GOOG"},
		{"AAPL"},
		{"FB"},
		{"AMZN"},
		{"MSFT"},
	}

	for _, test := range tests {
		stock := stock.NewStock(test.sym)
		if stock.Symbol != test.sym {
			t.Errorf("Improperly constructed stock: %+v vs %+v", stock, test)
		}
	}
}

func TestRangeIntegration(t *testing.T) {
	// test has db impact, setup test db
	tdb := stock.DB.Setup(stock.TestLocal)

	// run default test set for Markit
	testdata := testhelpers.MarkitTestData
	for _, test := range testdata {
		s := stock.NewStock(test.Sym)

		// initialize a request to get the URL so that we can inform the test server
		// of the target url, this request is not used otherwise.
		unusedRequest, err := stock.NewMarkitChartAPIRequest(s, test.StartDate, test.EndDate)
		if err != nil {
			t.Errorf("Could not create a MarkitChartAPIRequest: %v", err)
		}
		targetUrl := unusedRequest.Url // save the targetUrl for the request

		// start test server with success status
		tsParams := testhelpers.TestServer{
			Status: http.StatusOK, RequestUrl: targetUrl, TestData: testdata, T: t,
		}
		ts := httptest.NewServer(&tsParams)

		// make sure database and memory are empty for stock
		checkMemoryAndDatabase(s, &stock.Span{}, tdb, t)

		// test five most common error responses
		var errorCodes = []int{500, 404, 403, 400, 401}
		for _, errorCode := range errorCodes {
			tsParams = testhelpers.TestServer{
				Status: errorCode, RequestUrl: targetUrl, TestData: testdata, T: t,
			}

			span, err := s.ActualRange(test.StartDate, test.EndDate, ts.URL)
			if err == nil || len(span) > 0 {
				t.Errorf("Expected an error but got success: %+v\n", span)
			}
		}

		// make sure database and memory are still empty
		checkMemoryAndDatabase(s, &stock.Span{}, tdb, t)

		// change server to success status
		tsParams = testhelpers.TestServer{
			Status: http.StatusOK, RequestUrl: targetUrl, TestData: testdata, T: t,
		}

		// get the span here
		span, err := s.ActualRange(test.StartDate, test.EndDate, ts.URL)
		if test.ExpectError {
			if err == nil {
				t.Errorf("Expected error but got success")
			} else {
				// checkMemoryAndDatabase are empty and then continue, no teardown
				checkMemoryAndDatabase(s, &stock.Span{}, tdb, t)
				t.Logf("Test complete: %s, %s to %s", test.Sym, test.StartDate.String(), test.EndDate.String())
				continue
			}
		} else if err != nil {
			t.Error(err)
		}

		// record the changes we make so they can be reversed, all measures in span will be inserted
		td := testhelpers.TearDown{}
		td = td.TrackSpanInsert(s.Symbol, &span, tdb, t)

		// ensure that the returned span equals the expected
		expectedSpan := test.ExpectedMarkitResponse.GetSpan()
		if !span.Equal(expectedSpan) {
			t.Errorf("Error - stock.Range() did not return expected span\nexpected:\n%+v\ngot:\n%+v\n", expectedSpan, span)
		}

		// make sure database and memory also reflect the expected
		checkMemoryAndDatabase(s, &expectedSpan, tdb, t)

		// teardown changes
		err = td.TearDown(tdb, t)
		if err != nil {
			t.Error(err)
		}

		t.Logf("Test complete: %s, %s to %s", test.Sym, test.StartDate.String(), test.EndDate.String())
	}
}

func checkMemoryAndDatabase(st *stock.Stock, sp *stock.Span, tdb *stock.StockDB, t *testing.T) {
	selectSchema := `SELECT Time, Value FROM Measures where Symbol = $1`

	if len(*sp) == 0 {
		// sp is empty, we should expect the stock to be unpopulated

		// Confirm memory is missing this stock
		if len(st.Span) > 0 {
			t.Errorf("Expected empty stock.Span but had memoized data, stock.Span:\n%+v\n", st.Span)
		}
		// Confirm database does not include data for this stock
		r, e := tdb.Queryx(selectSchema, st.Symbol)
		if e != nil {
			t.Error(e)
		} else {
			// we'll get rows if the table exists, but won't be able to access any data
			hasNext := r.Next()
			if hasNext {
				// if we have data something is wrong
				t.Errorf("Expected nothing in database, but found data for stock.Symbol: %s\n", st.Symbol)
			}
		}
	} else {
		// sp is the expected results
		if !st.Span.Equal(*sp) {
			t.Errorf("Expected span to be memoized. Expected:\n%+v\n Got:\n%+v\n", st.Span, sp)
		}

		r, e := tdb.Queryx(selectSchema, st.Symbol)
		if e != nil {
			t.Error(e)
		}
		dbSpan := *new(stock.Span)
		for r.Next() {
			m := new(stock.Measure)
			e = r.StructScan(m)
			if e != nil {
				t.Error(e)
			}
			dbSpan = append(dbSpan, *m)
		}

		// sp is the expected results
		if !st.Span.Equal(dbSpan) {
			t.Errorf("Expected span to be in the database. Expected:\n%+v\n Got:\n%+v\n", st.Span, dbSpan)
		}
	}
}
