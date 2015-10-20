// Copyright 2015 Jordan Hurwich - no license granted

package testhelpers

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jhurwich/trendy/stock"
)

// keep a tape of all the actions taken by a test so they can be reversed with tearDown
type Action int

const (
	CREATE Action = iota
	INSERT
)

type Key struct {
	Symbol string
	Date   time.Time
}

type Change struct {
	Table  string
	Action Action
	Key    Key
}

type TearDown []Change

func (td *TearDown) TearDown(tdb *stock.StockDB, t *testing.T) error {
	sort.Sort(td) // sort actions so that CREATE is reversed last
	for _, change := range *td {
		var err error = nil
		switch change.Action {
		case CREATE:
			// reverse of table creation is dropping the table
			exec := strings.Join([]string{"DROP TABLE ", change.Table}, "")
			_, err = tdb.Exec(exec)
		case INSERT:
			// reverse of insert is delete, symbol and time is key
			exec := strings.Join([]string{"DELETE FROM ", change.Table, " WHERE Symbol = $1 AND Time = $2"}, "")
			_, err = tdb.Exec(exec, change.Key.Symbol, stock.TimeForSQL(change.Key.Date))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (td *TearDown) TrackSpanInsert(sym string, s *stock.Span, tdb *stock.StockDB, t *testing.T) TearDown {
	for _, measure := range *s {
		result := append(*td, Change{Table: "measures", Action: INSERT, Key: Key{sym, measure.Time}})
		td = &result
	}
	return *td
}

// implement sort.Interface on tearDown
func (td TearDown) Len() int {
	return len(td)
}

func (td TearDown) Less(i, j int) bool {
	return td[i].Action > td[j].Action
}

func (td TearDown) Swap(i, j int) {
	td[i], td[j] = td[j], td[i]
}
