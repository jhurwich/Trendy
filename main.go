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

//TrendyServer is an extension of a Negroni instance
type TrendyServer struct {
	negroni.Negroni
}

//TrendyRouter is a n extension of a httprouter instance
type TrendyRouter struct {
	httprouter.Router
}

func NewTrendyServer(flags Flags) TrendyServer {
	server := TrendyServer{*negroni.New()}

	// initialize the database
	if *flags.Local {
		stock.DB.Setup(stock.Local)
	} else {
		stock.DB.Setup(stock.Production)
	}

	// app manages request handling middleware, we use negroni package
	server.Use(negroni.NewRecovery())
	server.Use(negroni.NewLogger())

	// use secure middleware package to only receive https connections
	secureMiddleware := secure.New(secure.Options{
		SSLRedirect:          true,
		SSLTemporaryRedirect: false,        // false indicates using 301 (perm) redirect instead of 302 (temp)
		SSLHost:              "",           // "localhost:8443" This is optional in production. The default behavior is to just redirect the request to the HTTPS protocol. Example: http://github.com/some_page would be redirected
		IsDevelopment:        *flags.Local, // if running locally we run in development mode (SSL rqmt relaxed etc.)
	})
	server.Use(negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext))

	// protect all app routes using RestGate with static X-Auth-Key and Secret
	// if local we assume dev environment with HTTPS reqmt off and debug on
	key, secret := "key", "secret" // TODO implement real key and secret
	restgateConfig := restgate.Config{
		Key:                []string{key},
		Secret:             []string{secret},
		Debug:              *flags.Local,
		HTTPSProtectionOff: *flags.Local,
	}
	server.Use(restgate.New("X-Auth-Key", "X-Auth-Secret", restgate.Static, restgateConfig))

	// setup router, requests are dispatched using httprouter package
	router := NewTrendyRouter()

	server.UseHandler(&router)

	return server
}

func NewTrendyRouter() TrendyRouter {
	router := TrendyRouter{*httprouter.New()}

	//	Routes:
	// 		GET 	.../stock/<symbol>		GetStock()
	// TODO	POST	.../dev/add/<symbol>	AddStock()
	router.GET("/stock/:symbol", GetStock)
	return router
}

func main() {
	flags = Flags{Local: flag.Bool("local", false, "is the app running locally?")}
	flag.Parse()

	server := NewTrendyServer(flags)
	if *flags.Local {
		server.Run(":8080")
	} else {
		// TODO server.Run(<prod port>)
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
	loc, err := time.LoadLocation("America/New_York")
	if start != "" {
		startTime, err = time.ParseInLocation("2006-01-02", start, loc)
		if err != nil {
			errStr := fmt.Sprintf("Could not parse start as time. must be YYYY-MM-DD [%s]", start)
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}
	}
	if end != "" {
		endTime, err = time.ParseInLocation("2006-01-02", end, loc)
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
