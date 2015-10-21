// Copyright 2015 Jordan Hurwich - no license granted

package stock

import (
	"sort"
	"time"
)

// Stock object manages all data accesses for a specific stock symbol
type Stock struct {
	Symbol string
	Span   Span
}

func NewStock(sym string) *Stock {
	return &Stock{
		Symbol: sym,
	}
}

// Query daily measure data between times provided for a stock.
// Range calls ActualRange with empty string for overrideUrl to get default url,
// which should be Markit's. This separation exists for dependency injection in tests.
func (s *Stock) Range(startDate time.Time, endDate time.Time) (Span, error) {
	return s.ActualRange(startDate, endDate, "")
}
func (s *Stock) ActualRange(startDate time.Time, endDate time.Time, overrideUrl string) (Span, error) {
	// Check if data is memoized in s.Span, if so return that subslice.
	if s.Span.Covers(startDate) && s.Span.Covers(endDate) {
		// Find the first date after startDate in Span. The smallest range that
		// includes startDate begins at that date - 1
		start := sort.Search(len(s.Span), func(i int) bool { return s.Span[i].Time.After(startDate) }) - 1
		end := sort.Search(len(s.Span), func(i int) bool { return s.Span[i].Time.After(endDate) })
		return s.Span[start:end], nil
	}

	// all or part of the data is missing from what is memoized, check the database
	dbSpan, err := DB.GetRange(s, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(dbSpan) > 0 {
		// information was stored in the database, return it
		return dbSpan, nil
	} else {
		// data wasn't in database, populate it
		newSpan, err := s.ActualPopulate(startDate, endDate, overrideUrl)
		if err != nil {
			return nil, err
		}
		return newSpan, nil
	}
}

// func (s *Stock) RangeAll() (Span, error) {
// 	// zero time is used as sentinel to query all data available
// 	return Span{}, nil // TODO implement
// }
// func (s *Stock) RangeFrom(startDate time.Time) (Span, error) {
// 	// return all data from startDate to now
// 	start := sort.Search(len(s.Span), func(i int) bool { return s.Span[i].Time.After(startDate) }) - 1
// 	return s.Span[start:], nil
// }

// func (s *Stock) RangeTo(endDate time.Time) (Span, error) {
// 	// return all data from beginning of time to endDate
// 	end := sort.Search(len(s.Span), func(i int) bool { return s.Span[i].Time.After(endDate) })
// 	return s.Span[:end], nil
// }

// Populate daily measure data (value field) between times provided.
// Populate calls ActualPopulate with empty string for overrideUrl to get default url,
// which should be Markit's. This separation exists for dependency injection in tests.
func (s *Stock) Populate(startDate time.Time, endDate time.Time) (Span, error) {
	return s.ActualPopulate(startDate, endDate, "")
}
func (s *Stock) ActualPopulate(startDate time.Time, endDate time.Time, overrideUrl string) (Span, error) {

	request, err := NewMarkitChartAPIRequest(s, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// if there's an overrideUrl specified, set the request to that url
	if overrideUrl != "" {
		request.Url = overrideUrl
	}

	response, err := request.Request()
	if err != nil {
		return nil, err
	}

	s.Span = response.GetSpan()

	DB.Insert(s, &s.Span)

	return s.Span, nil
}

// func (s *Stock) PopulateAll() (Span, error) {
// 	// zero time is used as sentinel to populate all data
// 	return s.Populate(time.Time{}, time.Time{})
// }

// calculate trend for measures between times provided
func (s *Stock) Analyze(startDate time.Time, endDate time.Time) error {

	return nil
}

// func (s *Stock) AnalyzeAll() error {
// 	// zero time is used as sentinel to analyze all data
// 	return s.Analyze(time.Time{}, time.Time{})
// }

// Span and Measure objects for calculating and saving trend data
type Span []Measure
type Measure struct {
	Time  time.Time
	Value float32
	// trend float32 TODO(jhurwich) figure out where this belongs,, Trend with Start and End might be best
}

// implement an equals function for both span and measure
func (l Span) Equal(r Span) bool {
	if len(l) != len(r) {
		return false
	}

	for i := 0; i < len(l); i++ {
		if !l[i].Equal(r[i]) {
			return false
		}
	}
	return true
}
func (l Measure) Equal(r Measure) bool {
	// measures are equal if the value, year, and day of year are the same
	return l.Value == r.Value &&
		l.Time.Year() == r.Time.Year() &&
		l.Time.YearDay() == r.Time.YearDay()
}

// implement sort.Interface on Span
func (s Span) Len() int {
	return len(s)
}

func (s Span) Less(i, j int) bool {
	return s[i].Time.Before(s[j].Time)
}

func (s Span) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// utility method for if time t is included within the spans timeframe
func (s *Span) Covers(t time.Time) bool {
	if len(*s) == 0 {
		return false
	}
	if !sort.IsSorted(s) {
		sort.Sort(s)
	}
	// get times for comparison with only hours, everything else can get messy
	compareTime := t.Truncate(time.Hour)
	firstDate, lastDate := (*s)[0].Time.Truncate(time.Hour), (*s)[len(*s)].Time.Truncate(time.Hour)

	return ((firstDate.Before(compareTime) || firstDate.Equal(compareTime)) &&
		(lastDate.After(compareTime) || lastDate.Equal(compareTime)))
}
