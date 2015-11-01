// Copyright 2015 Jordan Hurwich - no license granted

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jhurwich/trendy/stock"
	"github.com/jhurwich/trendy/testhelpers"
)

type TestData struct {
	Sym                    string
	StartDate              time.Time
	EndDate                time.Time
	ExpectedResponseBody   string
	ExpectedMarkitResponse stock.MarkitChartAPIResponse
	ExpectError            bool
}

func TestGetStockIntegration(t *testing.T) {
	// start a new server so we can access it's ServeHTTP method
	trueVal := true
	ts := NewTrendyServer(Flags{Local: &trueVal})

	// test has db impact, setup test db
	tdb := stock.DB.Setup(stock.TestLocal)

	// run default test set for Markit
	testdata := testhelpers.MarkitTestData
	for _, test := range testdata {
		// .../stock/<test.Sym>?start=<test.StartDate>&end=<test.EndDate>, dates as YYYY-MM-DD
		path := strings.Join([]string{`/stock/`, test.Sym, `?start=`, test.StartDate.Format("2006-01-02"), `&end=`, test.EndDate.Format("2006-01-02")}, "")
		t.Logf("path: `%s`\n", path)

		// first try without auth parameters, this should be rejected
		r, _ := http.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		ts.ServeHTTP(w, r)

		b := w.Body.String()
		if w.Code != http.StatusUnauthorized || !strings.Contains(b, "No Key Or Secret") {
			t.Errorf("Attempted access without key/secret succeeded. Invalid access granted.\n>> Status:%d for %s", w.Code, path)
		}

		r, _ = http.NewRequest("GET", path, nil)
		r.Header.Set("X-Auth-Key", "key")
		r.Header.Set("X-Auth-Secret", "secret")
		w = httptest.NewRecorder()
		ts.ServeHTTP(w, r)

		if !test.ExpectError {
			// expect success
			if w.Code != http.StatusOK {
				t.Errorf("Attempted access with key/secret but failed, expected success.\n>> Status:%d for %s", w.Code, path)
			}

			// record the changes we make so they can be reversed, all measures in span will be inserted
			td := testhelpers.TearDown{}
			expectedSpan := test.ExpectedMarkitResponse.GetSpan()
			td = td.TrackSpanInsert(test.Sym, &expectedSpan, tdb, t)

			expectedStock := stock.NewStock(test.Sym)
			expectedStock.Span = expectedSpan
			expectedJson, err := json.Marshal(expectedStock)
			if err != nil {
				t.Errorf("Error generating JSON response for expected stock [%s]", test.Sym)
				return
			}
			expectedStr := strings.Replace(string(expectedJson), "T12:", "T00:", -1) // expectation is at T12, returned at T00, this is a quick hacky fix

			if b = w.Body.String(); b != expectedStr {
				t.Errorf("(%d) Got unexpected response from server, expected:\n%s\ngot:\n%s\n", w.Code, expectedJson, b)
			}

			// teardown changes
			err = td.TearDown(tdb, t)
			if err != nil {
				t.Error(err)
			}

		} else {
			// expect error
			if w.Code == http.StatusOK {
				t.Errorf("Attempted access expecting failure but got success.\n>> Status:%d for %s", w.Code, path)
			}
		}
	}
}
