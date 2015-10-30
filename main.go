// Copyright 2015 Jordan Hurwich - no license granted

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	"github.com/pjebs/restgate"
	"github.com/unrolled/secure"

	"github.com/jhurwich/trendy/stock"
)

type Flags struct {
	Local *bool
}

var flags Flags

func main() {
	flags = Flags{Local: flag.Bool("local", false, "is the app running locally?")}
	flag.Parse()

	// initialize the database
	if *flags.Local {
		stock.DB.Setup(stock.Local)
	} else {
		stock.DB.Setup(stock.Production)
	}

	// app manages request handling middleware, we use negroni package
	app := negroni.New()
	app.Use(negroni.NewRecovery())
	app.Use(negroni.NewLogger())

	// use secure middleware package to only receive https connections
	secureMiddleware := secure.New(secure.Options{
		SSLRedirect:          true,
		SSLTemporaryRedirect: false,        // false indicates using 301 (perm) redirect instead of 302 (temp)
		SSLHost:              "",           // "localhost:8443" This is optional in production. The default behavior is to just redirect the request to the HTTPS protocol. Example: http://github.com/some_page would be redirected
		IsDevelopment:        *flags.Local, // if running locally we run in development mode (SSL rqmt relaxed etc.)
	})
	app.Use(negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext))

	// protect all app routes using RestGate with static X-Auth-Key and Secret
	// if local we assume dev environment with HTTPS reqmt off and debug on
	key, secret := "key", "secret" // TODO implement real key and secret
	restgateConfig := restgate.Config{
		Key:                []string{key},
		Secret:             []string{secret},
		Debug:              *flags.Local,
		HTTPSProtectionOff: *flags.Local,
	}
	app.Use(restgate.New("X-Auth-Key", "X-Auth-Secret", restgate.Static, restgateConfig))

	// 	requests are dispatched using httprouter package
	//	Routes:
	// 		GET 	.../stock/<symbol>		GetStock()
	//		POST	.../dev/add/<symbol>	AddStock()
	router := httprouter.New()
	router.GET("/stock/:symbol", GetStock)

	app.UseHandler(router)

	if *flags.Local {
		app.Run(":8080")
	} else {
		// TODO app.Run(<prod port>)
	}
}

// ps includes "symbol" param
// queryValues may include "start" and "end", as YYYY-MM-DD, and requested optional "fields"
func GetStock(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// additional fields as url parameters
	queryValues := r.URL.Query()

	// TODO make start and end optional
	// parse start and end times if provided
	start, end := queryValues.Get("start"), queryValues.Get("end")
	var startTime, endTime time.Time
	var err error
	if start != "" {
		startTime, err = time.Parse("2006-01-02", start)
		if err != nil {
			errStr := fmt.Sprintf("Could not parse start as time. must be YYYY-MM-DD [%s]", start)
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}
	}
	if end != "" {
		endTime, err = time.Parse("2006-01-02", end)
		if err != nil {
			errStr := fmt.Sprintf("Could not parse end as time. must be YYYY-MM-DD [%s]", end)
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}
	}

	fields := queryValues.Get("fields")
	if fields != "" {
		// TODO handle optional fields
	}

	stock := stock.NewStock(ps.ByName("symbol"))
	span, err := stock.Range(startTime, endTime)
	if err != nil {
		errStr := fmt.Sprintf("Could not get range for provided stock over start to end [%s:%s-%s]", ps.ByName("symbol"), start, end)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	stock.Span = span // override memoized span

	json, err := json.Marshal(stock)
	if err != nil {
		errStr := fmt.Sprintf("Error generating JSON response for stock over start to end [%s:%s-%s]", ps.ByName("symbol"), start, end)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}
