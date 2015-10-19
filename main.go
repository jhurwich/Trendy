// Copyright 2015 Jordan Hurwich - no license granted

package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/jhurwich/trendy/stock"
)

type Flags struct {
	Local *bool
}

var flags Flags

func main() {
	flags = Flags{Local: flag.Bool("local", false, "is the app running locally?")}
	flag.Parse()

	stock.DB.Setup(*flags.Local)

	s := stock.NewStock("GOOG")
	start := time.Date(2005, time.October, 1, 12, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 30)
	span, err := s.Range(start, end)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("SPAN: %+v\n", span)
}
