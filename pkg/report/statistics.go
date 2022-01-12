package report

import (
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"time"
)

type Statistics struct {
	db     *sql.DB
	schema string
}

func NewStatistics(db *sql.DB, schema string) (*Statistics, error) {
	stats := &Statistics{
		db:     db,
		schema: schema,
	}
	return stats, nil
}

func (stat *Statistics) getSums() (num int64, size int64, files int64, err error) {
	sqlstr := fmt.Sprintf("SELECT COUNT(*) FROM %s.bagit", stat.schema)
	row := stat.db.QueryRow(sqlstr)
	var numRows int64
	if err := row.Scan(&numRows); err != nil {
		return 0, 0, 0, errors.Wrapf(err, "cannot get number of rows - %s", sqlstr)
	}
	return
}

type OverviewBagit struct {
	BagitID  int64
	Size     int64
	Files    int64
	Name     string
	HealthOK bool
}

type Overview struct {
	Size                   int64
	Files                  int64
	Status                 string
	Bagits                 []*OverviewBagit
	HealthOK, HealthFailed int64
}

func (stat *Statistics) Overview() (*Overview, error) {
	var ov = &Overview{
		Bagits: []*OverviewBagit{},
	}

	sqlstr := fmt.Sprintf("SELECT b.bagitid, b.name, SUM(c.filesize) AS filesize, COUNT(c.contentid) AS files "+
		" FROM %s.bagit b, %s.content c "+
		" WHERE b.bagitid = c.bagitid GROUP BY b.bagitid", stat.schema, stat.schema)
	rows, err := stat.db.Query(sqlstr)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot execute query %s", sqlstr)
	}
	for rows.Next() {
		var bo = &OverviewBagit{}
		if err := rows.Scan(&bo.BagitID, &bo.Name, &bo.Size, &bo.Files); err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "cannot scan result of query %s", sqlstr)
		}

		ov.Bagits = append(ov.Bagits, bo)
		ov.Size += bo.Size
		ov.Files += bo.Files
	}
	rows.Close()
	for _, bagit := range ov.Bagits {
		sqlstr := fmt.Sprintf("SELECT "+
			"       (SELECT status FROM bagit_test_location WHERE bagit_location_testid=MAX(btl1.bagit_location_testid)) AS status, "+
			"        l.quality, l.costs "+
			" FROM %s.`bagit_test_location` btl1, %s.location l "+
			" WHERE l.locationid=btl1.locationid AND btl1.bagitid=? "+
			" GROUP BY btl1.bagitid, btl1.locationid", stat.schema, stat.schema)
		rows, err := stat.db.Query(sqlstr, bagit.BagitID)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot execute query %s", sqlstr)
		}
		var totalQuality float64
		for rows.Next() {
			var status string
			var quality, costs float64
			if err := rows.Scan(&status, &quality, &costs); err != nil {
				rows.Close()
				return nil, errors.Wrapf(err, "cannot scan result of query %s", sqlstr)
			}
			if status == "ok" {
				totalQuality += quality
			}
		}
		if totalQuality >= 1.0 {
			bagit.HealthOK = true
			ov.HealthOK++
		} else {
			bagit.HealthOK = false
			ov.HealthFailed++
		}
		rows.Close()
	}
	return ov, nil
}

type BagitOverviewFile struct {
	Name                  string
	Size                  int64
	Mimetype              string
	Width, Height, Length int64
}

type BagitOverviewIngest struct {
	LocationID       int64
	Location         string
	TransferStart    time.Time
	TransferDuration time.Duration
	Status           string
	Message          string
	Encrypted        bool
}

type BagitOverviewCheck struct {
	Name       string
	Location   string
	LocationID int64
	Start      time.Time
	Duration   time.Duration
	Status     string
	Message    string
}

type BagitOverview struct {
	Name               string
	Size               int64
	Creator            string
	SHA512, SHA512_AES string
	CreationDate       time.Time
	HealthOK           bool
	Checks             []*BagitOverviewCheck
	Files              []*BagitOverviewFile
	Ingest             []*BagitOverviewIngest
}

func (stat *Statistics) BagitOverview(id int64) (*BagitOverview, error) {
	var bo = &BagitOverview{
		Ingest: []*BagitOverviewIngest{},
		Files:  []*BagitOverviewFile{},
	}
	var creationDate = sql.NullTime{}
	sqlstr := fmt.Sprintf("SELECT name, filesize, sha512, sha512_aes, creator, creationdate"+
		" FROM %s.bagit WHERE bagitid=?", stat.schema)
	row := stat.db.QueryRow(sqlstr, id)
	if err := row.Scan(&bo.Name, &bo.Size, &bo.SHA512, &bo.SHA512_AES, &bo.Creator, &creationDate); err != nil {
		return nil, errors.Wrapf(err, "cannot execute query %s", sqlstr)
	}
	bo.CreationDate = creationDate.Time

	sqlstr = fmt.Sprintf("SELECT l.locationid, l.name AS location, l.encrypted, bl.transfer_start, bl.transfer_end, bl.status, bl.message"+
		" FROM %s.bagit_location bl, %s.location l"+
		" WHERE bl.locationid=l.locationid AND bl.bagitid=?", stat.schema, stat.schema)
	rows, err := stat.db.Query(sqlstr, id)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot execute query %s", sqlstr)
	}
	for rows.Next() {
		var boi = &BagitOverviewIngest{}
		var transferStart = sql.NullTime{}
		var transferEnd = sql.NullTime{}
		var encrypted int64
		if err := rows.Scan(&boi.LocationID, &boi.Location, &encrypted, &transferStart, &transferEnd, &boi.Status, &boi.Message); err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "cannot scan query row %s", sqlstr)
		}
		boi.Encrypted = encrypted > 0
		boi.TransferStart = transferStart.Time
		boi.TransferDuration = transferEnd.Time.Sub(transferStart.Time)
		bo.Ingest = append(bo.Ingest, boi)
	}
	rows.Close()
	return bo, nil
}
