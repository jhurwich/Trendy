// Copyright 2015 Jordan Hurwich - no license granted

package stock_test

import (
	"testing"

	"github.com/jhurwich/trendy/stock"
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
