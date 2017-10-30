// +build go1.9

package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func TestOutputParam(t *testing.T) {
	sqltextcreate := `
CREATE PROCEDURE abassign
   @aid INT,
   @bid INT OUTPUT
AS
BEGIN
   SELECT @bid = @aid
END;
`
	sqltextdrop := `DROP PROCEDURE abassign;`
	sqltextrun := `abassign`

	checkConnStr(t)
	SetLogger(testLogger{t})

	db, err := sql.Open("sqlserver", makeConnStr(t).String())
	if err != nil {
		t.Fatalf("failed to open driver sqlserver")
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db.ExecContext(ctx, sqltextdrop)
	_, err = db.ExecContext(ctx, sqltextcreate)
	if err != nil {
		t.Fatal(err)
	}
	var bout int64
	_, err = db.ExecContext(ctx, sqltextrun,
		sql.Named("aid", 5),
		sql.Named("bid", sql.Out{Dest: &bout}),
	)
	defer db.ExecContext(ctx, sqltextdrop)
	if err != nil {
		t.Error(err)
	}

	if bout != 5 {
		t.Errorf("expected 5, got %d", bout)
	}
}

func TestOutputINOUTParam(t *testing.T) {
	sqltextcreate := `
CREATE PROCEDURE abinout
   @aid INT,
   @bid INT OUTPUT
AS
BEGIN
   SELECT @bid = @aid + @bid;
END;
`
	sqltextdrop := `DROP PROCEDURE abinout;`
	sqltextrun := `abinout`

	checkConnStr(t)
	SetLogger(testLogger{t})

	db, err := sql.Open("sqlserver", makeConnStr(t).String())
	if err != nil {
		t.Fatalf("failed to open driver sqlserver")
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db.ExecContext(ctx, sqltextdrop)
	_, err = db.ExecContext(ctx, sqltextcreate)
	if err != nil {
		t.Fatal(err)
	}
	var bout int64 = 3
	_, err = db.ExecContext(ctx, sqltextrun,
		sql.Named("aid", 5),
		sql.Named("bid", sql.Out{Dest: &bout}),
	)
	defer db.ExecContext(ctx, sqltextdrop)
	if err != nil {
		t.Error(err)
	}

	if bout != 8 {
		t.Errorf("expected 8, got %d", bout)
	}
}

// TestTLSServerReadClose tests writing to an encrypted database connection.
// Currently the database server will close the connection while the server is
// reading the TDS packets and before any of the data has been parsed.
//
// When two queries are sent in reverse order, they PASS, but if we send only
// a single ping (SELECT 1;) first, then the long query the query fails.
//
// The long query text is never parsed. In fact, you can comment out, return
// early, or have malformed sql in the long query text. Just the length matters.
// The error happens when sending the TDS Batch packet to SQL Server the server
// closes the connection..
//
// It appears the driver sends valid TDS packets. In fact, if prefixed with 4
// "SELECT 1;" TDS Batch queries then the long query works, but if zero or one
// "SELECT 1;" TDS Batch queries are send prior the long query fails to send.
//
// Lastly, this only manafests itself with an encrypted connection. This has been
// observed with SQL Server Azure, SQL Server 13.0.1742 on Windows, and SQL Server
// 14.0.900.75 on Linux. It also fails when using the "dev.boringcrypto" (a C based
// TLS crypto). I haven't found any knobs on SQL Server to expose the error message
// nor have I found a good way to decrypt the TDS stream. KeyLogWriter in the TLS
// config may help with that, but wireshark wasn't decrypting TDS based TLS streams
// even when using that.
//
// Issue https://github.com/denisenkom/go-mssqldb/issues/166
func TestTLSServerReadClose(t *testing.T) {
	query := `
with
    config_cte (config) as (
            select *
                    from ( values
                    ('_partition:{\"Fill\":{\"PatternType\":\"solid\",\"FgColor\":\"99ff99\"}}')
                    , ('_separation:{\"Fill\":{\"PatternType\":\"solid\",\"FgColor\":\"99ffff\"}}')
                    , ('Monthly Earnings:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Weekly Earnings:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Total Earnings:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Average Earnings:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Last Month Earning:#,##0.00 ;(#,##0.00)')
                    , ('Award:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Amount:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Grand Total:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Total:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Price Each:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Hyperwallet:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Credit/Debit:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Earning:#,##0.00 ;(#,##0.00)')
                    , ('Change Earning:#,##0.00 ;(#,##0.00)')
                    , ('CheckAmount:#,##0.00 ;(#,##0.00)')
                    , ('Residual:#,##0.00 ;(#,##0.00)')
                    , ('Prev Residual:#,##0.00 ;(#,##0.00)')
                    , ('Team Bonuses:#,##0.00 ;(#,##0.00)')
                    , ('Change:#,##0.00 ;(#,##0.00)')
                    , ('Shipping Total:#,##0.00 ;(#,##0.00)')
                    , ('SubTotal:\$#,##0.00 ;(\$#,##0.00)')
                    , ('Total Diff:#,##0.00 ;(#,##0.00)')
                    , ('SubTotal Diff:#,##0.00 ;(#,##0.00)')
                    , ('Return Total:#,##0.00 ;(#,##0.00)')
                    , ('Return SubTotal:#,##0.00 ;(#,##0.00)')
                    , ('Return Total Diff:#,##0.00 ;(#,##0.00)')
                    , ('Return SubTotal Diff:#,##0.00 ;(#,##0.00)')
                    , ('Cancel Total:#,##0.00 ;(#,##0.00)')
                    , ('Cancel SubTotal:#,##0.00 ;(#,##0.00)')
                    , ('Cancel Total Diff:#,##0.00 ;(#,##0.00)')
                    , ('Cancel SubTotal Diff:#,##0.00 ;(#,##0.00)')
                    , ('Replacement Total:#,##0.00 ;(#,##0.00)')
                    , ('Replacement SubTotal:#,##0.00 ;(#,##0.00)')
                    , ('Replacement Total Diff:#,##0.00 ;(#,##0.00)')
                    , ('Replacement SubTotal Diff:#,##0.00 ;(#,##0.00)')
                    , ('Jan Residual:#,##0.00 ;(#,##0.00)')
                    , ('Jan Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Jan Total:#,##0.00 ;(#,##0.00)')
                    , ('January Residual:#,##0.00 ;(#,##0.00)')
                    , ('Feb Residual:#,##0.00 ;(#,##0.00)')
                    , ('Feb Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Feb Total:#,##0.00 ;(#,##0.00)')
                    , ('February Residual:#,##0.00 ;(#,##0.00)')
                    , ('Mar Residual:#,##0.00 ;(#,##0.00)')
                    , ('Mar Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Mar Total:#,##0.00 ;(#,##0.00)')
                    , ('March Residual:#,##0.00 ;(#,##0.00)')
                    , ('Apr Residual:#,##0.00 ;(#,##0.00)')
                    , ('Apr Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Apr Total:#,##0.00 ;(#,##0.00)')
                    , ('April Residual:#,##0.00 ;(#,##0.00)')
                    , ('May Residual:#,##0.00 ;(#,##0.00)')
                    , ('May Bonus:#,##0.00 ;(#,##0.00)')
                    , ('May Total:#,##0.00 ;(#,##0.00)')
                    , ('Jun Residual:#,##0.00 ;(#,##0.00)')
                    , ('Jun Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Jun Total:#,##0.00 ;(#,##0.00)')
                    , ('June Residual:#,##0.00 ;(#,##0.00)')
                    , ('Jul Residual:#,##0.00 ;(#,##0.00)')
                    , ('Jul Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Jul Total:#,##0.00 ;(#,##0.00)')
                    , ('July Residual:#,##0.00 ;(#,##0.00)')
                    , ('Aug Residual:#,##0.00 ;(#,##0.00)')
                    , ('Aug Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Aug Total:#,##0.00 ;(#,##0.00)')
                    , ('August Residual:#,##0.00 ;(#,##0.00)')
                    , ('Sep Residual:#,##0.00 ;(#,##0.00)')
                    , ('Sep Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Sep Total:#,##0.00 ;(#,##0.00)')
                    , ('September Residual:#,##0.00 ;(#,##0.00)')
                    , ('Oct Residual:#,##0.00 ;(#,##0.00)')
                    , ('Oct Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Oct Total:#,##0.00 ;(#,##0.00)')
                    , ('October Residual:#,##0.00 ;(#,##0.00)')
                    , ('Nov Residual:#,##0.00 ;(#,##0.00)')
                    , ('Nov Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Nov Total:#,##0.00 ;(#,##0.00)')
                    , ('November Residual:#,##0.00 ;(#,##0.00)')
                    , ('Dec Residual:#,##0.00 ;(#,##0.00)')
                    , ('Dec Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Dec Total:#,##0.00 ;(#,##0.00)')
                    , ('December Residual:#,##0.00 ;(#,##0.00)')
                    , ('January Bonus:#,##0.00 ;(#,##0.00)')
                    , ('February Bonus:#,##0.00 ;(#,##0.00)')
                    , ('March Bonus:#,##0.00 ;(#,##0.00)')
                    , ('April Bonus:#,##0.00 ;(#,##0.00)')
                    , ('May Bonus:#,##0.00 ;(#,##0.00)')
                    , ('June Bonus:#,##0.00 ;(#,##0.00)')
                    , ('July Bonus:#,##0.00 ;(#,##0.00)')
                    , ('August Bonus:#,##0.00 ;(#,##0.00)')
                    , ('September Bonus:#,##0.00 ;(#,##0.00)')
                    , ('October Bonus:#,##0.00 ;(#,##0.00)')
                    , ('November Bonus:#,##0.00 ;(#,##0.00)')
                    , ('December Bonus:#,##0.00 ;(#,##0.00)')
                    , ('January Adj:#,##0.00 ;(#,##0.00)')
                    , ('February Adj:#,##0.00 ;(#,##0.00)')
                    , ('March Adj:#,##0.00 ;(#,##0.00)')
                    , ('April Adj:#,##0.00 ;(#,##0.00)')
                    , ('May Adj:#,##0.00 ;(#,##0.00)')
                    , ('June Adj:#,##0.00 ;(#,##0.00)')
                    , ('July Adj:#,##0.00 ;(#,##0.00)')
                    , ('August Adj:#,##0.00 ;(#,##0.00)')
                    , ('September Adj:#,##0.00 ;(#,##0.00)')
                    , ('October Adj:#,##0.00 ;(#,##0.00)')
                    , ('November Adj:#,##0.00 ;(#,##0.00)')
                    , ('December Adj:#,##0.00 ;(#,##0.00)')
                    , ('2016- 2015 YTD Dif:#,##0.00 ;(#,##0.00)')
                    , ('2017- 2016 YTD Dif:#,##0.00 ;(#,##0.00)')
                    , ('2018- 2017 YTD Dif:#,##0.00 ;(#,##0.00)')
                    , ('Dec to Jan Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Jan to Feb Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Feb to Mar Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Mar to Apr Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Apr to May Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('May to Jun Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Jun to Jul Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Jul to Aug Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Aug to Sep Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Sep to Oct Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Oct to Nov Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Nov to Dec Dif Residual:#,##0.00 ;(#,##0.00)')
                    , ('Dec to Jan Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Jan to Feb Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Feb to Mar Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Mar to Apr Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Apr to May Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('May to Jun Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Jun to Jul Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Jul to Aug Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Aug to Sep Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Sep to Oct Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Oct to Nov Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Nov to Dec Dif Bonus:#,##0.00 ;(#,##0.00)')
                    , ('Dec to Jan Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Jan to Feb Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Feb to Mar Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Mar to Apr Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Apr to May Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('May to Jun Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Jun to Jul Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Jul to Aug Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Aug to Sep Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Sep to Oct Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Oct to Nov Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Nov to Dec Dif Total:#,##0.00 ;(#,##0.00)')
                    , ('Jan Refund Cnt:#,##0 ;(#,##0)')
                    , ('Feb Refund Cnt:#,##0 ;(#,##0)')
                    , ('Mar Refund Cnt:#,##0 ;(#,##0)')
                    , ('Apr Refund Cnt:#,##0 ;(#,##0)')
                    , ('May Refund Cnt:#,##0 ;(#,##0)')
                    , ('Jun Refund Cnt:#,##0 ;(#,##0)')
                    , ('Jul Refund Cnt:#,##0 ;(#,##0)')
                    , ('Aug Refund Cnt:#,##0 ;(#,##0)')
                    , ('Sep Refund Cnt:#,##0 ;(#,##0)')
                    , ('Oct Refund Cnt:#,##0 ;(#,##0)')
                    , ('Nov Refund Cnt:#,##0 ;(#,##0)')
                    , ('Dec Refund Cnt:#,##0 ;(#,##0)')
                    , ('Jan Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Feb Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Mar Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Apr Purchase Cnt:#,##0 ;(#,##0)')
                    , ('May Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Jun Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Jul Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Aug Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Sep Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Oct Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Nov Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Dec Purchase Cnt:#,##0 ;(#,##0)')
                    , ('Jan Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Feb Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Mar Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Apr Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('May Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Jun Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Jul Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Aug Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Sep Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Oct Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Nov Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Dec Refund Amt:#,##0.00 ;(#,##0.00)')
                    , ('Jan Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Feb Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Mar Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Apr Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('May Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Jun Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Jul Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Aug Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Sep Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Oct Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Nov Purchase Amt:#,##0.00 ;(#,##0.00)')
                    , ('Dec Purchase Amt:#,##0.00 ;(#,##0.00)')
                    ) X(a))
    select * from config_cte
	`

	db := open(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type run struct {
		name  string
		pings []int
		pass  bool

		conn *sql.Conn
	}

	// Use separate Conns from the connection pool to ensure separation.
	runs := []*run{
		{name: "rev", pings: []int{4, 1}, pass: true},
		{name: "forward", pings: []int{1}, pass: false},
	}
	for _, r := range runs {
		var err error
		r.conn, err = db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer r.conn.Close()
	}

	for _, r := range runs {
		for _, ping := range r.pings {
			t.Run(fmt.Sprintf("%s-ping-%d", r.name, ping), func(t *testing.T) {
				for i := 0; i < ping; i++ {
					if err := r.conn.PingContext(ctx); err != nil {
						if r.pass {
							t.Error("failed to ping server", err)
						} else {
							t.Log("failed to ping server", err)
						}
						return
					}
				}

				rows, err := r.conn.QueryContext(ctx, query)
				if err != nil {
					if r.pass {
						t.Errorf("QueryContext: %+v", err)
					} else {
						t.Logf("QueryContext: %+v", err)
					}
					return
				}
				for rows.Next() {
					// Nothing.
				}
				rows.Close()
			})
		}
	}
}
