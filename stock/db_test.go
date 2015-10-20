// Copyright 2015 Jordan Hurwich - no license granted

package stock_test

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jhurwich/trendy/stock"
	"github.com/jhurwich/trendy/testhelpers"
)

// Confirm that DB.Setup completes with a functioning db and tables to populate
func TestSetup(t *testing.T) {
	var tests = []struct {
		table string // the name for each table that should be created in Setup
	}{
		{"measures"},
	}

	// test that a new db can be created, panics if fails
	tdb := stock.DB.Setup(stock.TestLocal)

	// record the changes we make so they can be reversed
	td := testhelpers.TearDown{}
	for _, test := range tests {
		td = append(td, (testhelpers.Change{Table: test.table, Action: testhelpers.CREATE}))
	}
	defer func() {
		err := td.TearDown(tdb, t)
		if err != nil {
			t.Error(err)
		}
	}()

	// and ensure it has all the tables that we expect
	for _, test := range tests {
		testQuery := strings.Join([]string{"SELECT * FROM ", test.table}, "")
		rows, err := tdb.Query(testQuery)
		if err != nil {
			// if the table does not exist we'll get an error
			t.Error(err)
		} else {
			// we'll get rows if the table exists, but won't be able to access any data
			hasNext := rows.Next()
			if hasNext {
				// if we have data something is wrong
				t.Errorf("Expected empty table but got results with query `%s`", testQuery)
			}
		}
	}

}

func TestInsert(t *testing.T) {
	var tests = []struct {
		symbol string
		span   stock.Span
	}{
		{"GOOG", testSpan1},
	}

	// create test db connection
	tdb := stock.DB.Setup(stock.TestLocal)

	// record the changes we make so they can be reversed, all measures in span will be inserted
	td := testhelpers.TearDown{}
	for _, test := range tests {
		td = td.TrackSpanInsert(test.symbol, &test.span, tdb, t)
	}
	defer func() {
		err := td.TearDown(tdb, t)
		if err != nil {
			t.Error(err)
		}
	}()

	for _, test := range tests {
		// do the DB.Insert()
		err := tdb.Insert(stock.NewStock(test.symbol), &test.span)
		if err != nil {
			t.Error(err)
		}

		// query for all data for test.symbol and compare against provided span
		// use a direct Queryx here, we'll test GetRange separately
		rows, err := tdb.Queryx(`SELECT Time, Value FROM Measures where Symbol = $1`, test.symbol)
		if err != nil {
			t.Error(err)
		}

		spanToCheck := *new(stock.Span)
		for rows.Next() {
			m := new(stock.Measure)
			err = rows.StructScan(m)
			if err != nil {
				t.Error(err)
			}
			spanToCheck = append(spanToCheck, *m)
		}

		// compare retrieved and expected spans
		if !test.span.Equal(spanToCheck) {
			t.Errorf("Error on DB.Insert(), database did not respond with expected number of results after insert -\nexpected:\n%+v\ngot:\n%+v\n", test.span, spanToCheck)
		}
	}

}

// test getting data after DB.Insert(), same as above but we use the GetRange()
// method where we talked directly to the database before
func TestGetRange(t *testing.T) {
	var tests = []struct {
		symbol string
		span   stock.Span
	}{
		{"GOOG", testSpan1},
	}

	// create test db connection
	tdb := stock.DB.Setup(stock.TestLocal)

	// record the changes we make so they can be reversed, all measures in span will be inserted
	td := testhelpers.TearDown{}
	for _, test := range tests {
		td = td.TrackSpanInsert(test.symbol, &test.span, tdb, t)
	}
	defer func() {
		err := td.TearDown(tdb, t)
		if err != nil {
			t.Error(err)
		}
	}()

	for _, test := range tests {
		s := stock.NewStock(test.symbol)

		// do the DB.Insert()
		err := tdb.Insert(s, &test.span)
		if err != nil {
			t.Error(err)
		}

		// now get the range we just inserted
		sort.Sort(testSpan1)
		startDate := testSpan1[0].Time
		endDate := testSpan1[len(testSpan1)-1].Time
		span, err := tdb.GetRange(s, startDate, endDate)
		if err != nil {
			t.Error(err)
		}

		// compare retrieved and expected spans
		if !test.span.Equal(span) {
			t.Errorf("Error on DB.GetRange(), database did not respond with expected number of results after insert -\nexpected:\n%+v\ngot:\n%+v\n", test.span, span)
		}
	}
}

/* Utils */

/* Global Constants and Vars */

var testSpan1 stock.Span = (stock.Span)([]stock.Measure{
	{time.Date(2015, time.June, 1, 12, 0, 0, 0, time.UTC), 0.012345},
	{time.Date(2015, time.June, 2, 12, 0, 0, 0, time.UTC), 0.012346},
	{time.Date(2015, time.June, 3, 12, 0, 0, 0, time.UTC), 0.012347},
	{time.Date(2015, time.June, 4, 12, 0, 0, 0, time.UTC), 0.012348},
	{time.Date(2015, time.June, 5, 12, 0, 0, 0, time.UTC), 0.012349},
	{time.Date(2015, time.June, 6, 12, 0, 0, 0, time.UTC), 0.012350},
	{time.Date(2015, time.June, 7, 12, 0, 0, 0, time.UTC), 0.012351},
	{time.Date(2015, time.June, 8, 12, 0, 0, 0, time.UTC), 0.012352},
	{time.Date(2015, time.June, 9, 12, 0, 0, 0, time.UTC), 0.012353},
	{time.Date(2015, time.June, 10, 12, 0, 0, 0, time.UTC), 0.012354},
	{time.Date(2015, time.June, 11, 12, 0, 0, 0, time.UTC), 0.012355},
	{time.Date(2015, time.June, 12, 12, 0, 0, 0, time.UTC), 0.012354},
	{time.Date(2015, time.June, 13, 12, 0, 0, 0, time.UTC), 0.012353},
	{time.Date(2015, time.June, 14, 12, 0, 0, 0, time.UTC), 0.012352},
	{time.Date(2015, time.June, 15, 12, 0, 0, 0, time.UTC), 0.012351},
})
