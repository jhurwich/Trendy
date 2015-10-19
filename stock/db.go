// Copyright 2015 Jordan Hurwich - no license granted

package stock

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type StockDB struct {
	sqlx.DB
}

var DB *StockDB

func (db *StockDB) Setup(local bool) {
	// TODO(jhurwich) implement user/pass/address switch based on local or prod environment
	var dbname, password, host, user, suffix string
	if local {
		dbname = "trendydb"
		password = "localpass"
		host = "localhost"
		user = "localuser"
		suffix = "?sslmode=disable"
	} else {
		// TODO(jhurwich) define for production environment
	}
	dbSource := fmt.Sprintf("postgres://%s:%s@%s/%s%s", user, password, host, dbname, suffix)

	// initialize the db, note that it's a global object, it is never closed
	db = &StockDB{*(sqlx.MustConnect("postgres", dbSource))}
	db.CreateIfNotExists()
	DB = db // not entirely sure why we need this line with the address assignment two up, but whatever
}

const createMeasuresSchema string = `CREATE TABLE IF NOT EXISTS Measures ( Symbol varchar(255) NOT NULL, Time date NOT NULL, Value float8 NOT NULL, PRIMARY KEY (Symbol, Time))`

func (db *StockDB) CreateIfNotExists() {
	db.MustExec(createMeasuresSchema)
}

const insertMeasuresSchema string = `INSERT INTO Measures VALUES ($1, $2, $3)` //$1 is symbol, $2 is date, $3 is value

func (db *StockDB) Insert(stock *Stock, span *Span) error {
	// new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	for _, measure := range *span {
		_, err := tx.Exec(insertMeasuresSchema, stock.Symbol, timeForSQL(measure.Time), measure.Value)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

const selectMeasuresRangeSchema string = `SELECT Time, Value FROM Measures where Symbol = $1 AND Time >= $2 AND TIME < $3`
const selectMeasuresRangeFromSchema string = `SELECT Time, Value FROM Measures where Symbol = $1 AND Time >= $2`
const selectMeasuresRangeToSchema string = `SELECT Time, Value FROM Measures where Symbol = $1 AND Time < $2`
const selectMeasuresAllSchema string = `SELECT Time, Value FROM Measures where Symbol = $1`

func timeForSQL(time time.Time) string {
	// YYYY-MM-DD
	return time.Format("2006-01-02")
}

func (db *StockDB) GetRange(stock *Stock, startDate time.Time, endDate time.Time) (Span, error) {
	fmt.Printf("DB GET RANGE: %s, %s to %s\n", stock.Symbol, timeForSQL(startDate), timeForSQL(endDate))

	rows, err := db.Queryx(selectMeasuresRangeSchema, stock.Symbol, timeForSQL(startDate), timeForSQL(endDate))
	if err != nil {
		return nil, err
	}

	span := *new(Span)
	for rows.Next() {
		m := new(Measure)
		err = rows.StructScan(m)
		if err != nil {
			return nil, err
		}
		span = append(span, *m)
	}

	return span, nil
}
