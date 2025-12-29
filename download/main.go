package main

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"os"
	"time"
)

func main() {
	DownloadRecordings()
}

func DownloadRecordings() {
	db := ConnectToDbSql()
	rows, err := db.Query("SELECT " +
		"start_moment, " +
		"COALESCE(end_moment, start_moment), " +
		"user, " +
		"release_version, " +
		"COALESCE(simulation_version, -1), " +
		"COALESCE(input_version, -1), " +
		"id, " +
		"playthrough " +
		"FROM playthroughs")
	Check(err)
	defer func(rows *sql.Rows) { Check(rows.Close()) }(rows)

	dbRows := []dbRow{}
	for rows.Next() {
		row := dbRow{}
		err = rows.Scan(&row.startMoment, &row.endMoment, &row.user,
			&row.releaseVersion, &row.simulationVersion, &row.inputVersion,
			&row.id, &row.data)
		Check(err)
		dbRows = append(dbRows, row)
	}

	for i := range dbRows {
		dir := dbRows[i].user
		_ = os.Mkdir(dir, os.ModeDir)
		m := dbRows[i].startMoment
		var filename string
		if dbRows[i].simulationVersion == -1 || dbRows[i].inputVersion == -1 {
			// -1 values mean the fields were NULL (check the SQL query above).
			// If the simulation or input version is NULL it means we are
			// dealing with a playthrough recorded before splitting the version
			// into release, simulation and input versions.
			// Use the old extension system (e.g. .clone1016).
			filename = fmt.Sprintf("%s/%d%02d%02d-%02d%02d%02d.clone1-%03d",
				dir, m.Year(), m.Month(), m.Day(), m.Hour(), m.Minute(),
				m.Second(), dbRows[i].releaseVersion)
		} else {
			// Use the extension system that includes both simulation and
			// input versions: .clone1-019-012
			filename = fmt.Sprintf(
				"%s/%d%02d%02d-%02d%02d%02d.clone1-%d-%d", dir, m.Year(),
				m.Month(), m.Day(), m.Hour(), m.Minute(), m.Second(),
				dbRows[i].simulationVersion, dbRows[i].inputVersion)
		}
		WriteFile(filename, dbRows[i].data)
	}
}

func ConnectToDbSql() *sql.DB {
	cfg := mysql.Config{
		User:                 os.Getenv("CLONE1_DBUSER"),
		Passwd:               os.Getenv("CLONE1_DBPASSWORD"),
		Net:                  "tcp",
		Addr:                 os.Getenv("CLONE1_DBADDR"),
		DBName:               os.Getenv("CLONE1_DBNAME"),
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	Check(err)
	err = db.Ping()
	Check(err)
	return db
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

type dbRow struct {
	startMoment       time.Time
	endMoment         time.Time
	user              string
	releaseVersion    int64
	simulationVersion int64
	inputVersion      int64
	id                uuid.UUID
	data              []byte
}

func WriteFile(name string, data []byte) {
	err := os.WriteFile(name, data, 0644)
	Check(err)
}
