package report

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/inhies/go-bytesize"
	ffmpeg_models "github.com/je4/goffmpeg/models"
	"github.com/je4/indexer/pkg/indexer"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	siegfried_pronom "github.com/richardlehane/siegfried/pkg/pronom"
	"time"
)

type Statistics struct {
	db     *sql.DB
	log    *logging.Logger
	schema string
}

func NewStatistics(db *sql.DB, schema string, log *logging.Logger) (*Statistics, error) {
	stats := &Statistics{
		db:     db,
		schema: schema,
		log:    log,
	}
	return stats, nil
}

type Indexer struct {
	Duration  int64                             `json:"duration,omitempty"`
	Width     int64                             `json:"width,omitempty"`
	Height    int64                             `json:"height,omitempty"`
	Errors    map[string]string                 `json:"errors,omitempty"`
	Mimetype  string                            `json:"mimetype,omitempty"`
	NSRL      []indexer.ActionNSRLMeta          `json:"nsrl,omitempty"`
	Identify  map[string]interface{}            `json:"identify,omitempty"`
	FFProbe   ffmpeg_models.Metadata            `json:"ffprobe,omitempty"`
	Siegfried []siegfried_pronom.Identification `json:"siegfried,omitempty"`
	Tika      []map[string]interface{}          `json:"tika,omitempty"`
	Exif      map[string]interface{}            `json:"exif,omitempty"`
	Clamav    map[string]string                 `json:"clamav,omitempty"`
}

type Content struct {
	ContentID               int64
	ZipPath, DiskPath       string
	Filesize                int64
	Mimetype                string
	Checksums               map[string]string
	Width, Height, Duration int64
	Indexer                 Indexer
}

func (stat *Statistics) getContent(bagitID int64) ([]*Content, error) {
	var result = []*Content{}

	sqlstr := fmt.Sprintf("SELECT `contentid`, `zippath`, `diskpath`, `filesize`, "+
		" `checksums`, `mimetype`, `width`, `height`, `duration`, `indexer` "+
		" FROM %s.`content`"+
		" WHERE bagitid=?", stat.schema)
	rows, err := stat.db.Query(sqlstr, bagitID)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot execute query %s", sqlstr)
	}
	defer rows.Close()
	for rows.Next() {
		var content = &Content{
			Checksums: make(map[string]string),
			Indexer:   Indexer{},
		}
		var checksums, indexer string
		if err := rows.Scan(&content.ContentID, &content.ZipPath, &content.DiskPath, &content.Filesize,
			&checksums, &content.Mimetype, &content.Width, &content.Height, &content.Duration, &indexer); err != nil {
			return nil, errors.Wrapf(err, "cannot scan result of query %s", sqlstr)
		}
		stat.log.Debugf("Path: %s", content.DiskPath)
		if err := json.Unmarshal([]byte(checksums), &content.Checksums); err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal checksums - %s", checksums)
		}
		if err := json.Unmarshal([]byte(indexer), &content.Indexer); err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal indexer - %s", indexer)
		}
		result = append(result, content)
	}
	return result, nil
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

func (stat *Statistics) getCosts(bagitID int64) (costs float64, err error) {
	sqlstr := fmt.Sprintf("SELECT b.filesize, SUM(costs) AS costs FROM %s.bagit b, %s.bagit_location bl, %s.location l "+
		" WHERE bl.locationid=l.locationid AND bl.bagitid=b.bagitid AND bl.bagitid=?"+
		" GROUP BY bl.bagitid", stat.schema, stat.schema, stat.schema)
	row := stat.db.QueryRow(sqlstr, bagitID)
	var filesize int64
	if err := row.Scan(&filesize, &costs); err != nil {
		return 0, errors.Wrapf(err, "cannot get number of rows - %s", sqlstr)
	}
	costs *= float64(filesize) / float64(bytesize.MB)
	return
}

func (stat *Statistics) getHealth(bagitID int64) (quality float64, err error) {
	sqlstr := fmt.Sprintf("SELECT "+
		"       (SELECT status FROM bagit_test_location WHERE bagit_location_testid=MAX(btl1.bagit_location_testid)) AS status, "+
		"        l.quality, l.costs "+
		" FROM %s.`bagit_test_location` btl1, %s.location l "+
		" WHERE l.locationid=btl1.locationid AND btl1.bagitid=? "+
		" GROUP BY btl1.bagitid, btl1.locationid", stat.schema, stat.schema)
	//stat.log.Debugf("%s [%v]", sqlstr, bagitID)
	rows, err := stat.db.Query(sqlstr, bagitID)
	if err != nil {
		return 0, errors.Wrapf(err, "cannot execute query %s", sqlstr)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var q, costs float64
		if err := rows.Scan(&status, &q, &costs); err != nil {
			return 0, errors.Wrapf(err, "cannot scan result of query %s", sqlstr)
		}
		if status == "passed" {
			quality += q
		}
	}
	return
}

type OverviewBagit struct {
	BagitID  int64
	Size     int64
	Files    int64
	Name     string
	HealthOK bool
	Quality  float64
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
		bagit.Quality, err = stat.getHealth(bagit.BagitID)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get health of bagit #%v", bagit.BagitID)
		}
		if bagit.Quality >= 1.0 {
			bagit.HealthOK = true
			ov.HealthOK++
		} else {
			bagit.HealthOK = false
			ov.HealthFailed++
		}
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
	Quality          float64
	Costs            float64
}

type BagitOverviewCheck struct {
	Name       string
	Test       string
	Location   string
	LocationID int64
	Start      time.Time
	Duration   time.Duration
	Status     string
	Message    string
}

type BagitOverviewMime struct {
	Mimetype string
	Count    int64
	Size     int64
}

type BagitOverview struct {
	Name               string
	Size               int64
	Creator            string
	SHA512, SHA512_AES string
	CreationDate       time.Time
	HealthOK           bool
	Quality            float64
	BagInfo            string
	Checks             []*BagitOverviewCheck
	Files              []*BagitOverviewFile
	Ingest             []*BagitOverviewIngest
	Mimetypes          []*BagitOverviewMime
	Content            []*Content
	Costs              float64
}

func (stat *Statistics) BagitOverview(id int64) (*BagitOverview, error) {
	var err error
	var bo = &BagitOverview{
		Ingest:    []*BagitOverviewIngest{},
		Files:     []*BagitOverviewFile{},
		Checks:    []*BagitOverviewCheck{},
		Mimetypes: []*BagitOverviewMime{},
		Content:   []*Content{},
	}
	bo.Quality, err = stat.getHealth(id)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get health of #%v", id)
	}
	bo.HealthOK = bo.Quality >= 1

	bo.Costs, err = stat.getCosts(id)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get costs of #%v", id)
	}

	bo.Content, err = stat.getContent(id)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get content of #%v", id)
	}

	var creationDate = sql.NullTime{}
	sqlstr := fmt.Sprintf("SELECT name, baginfo, filesize, sha512, sha512_aes, creator, creationdate"+
		" FROM %s.bagit WHERE bagitid=?", stat.schema)
	row := stat.db.QueryRow(sqlstr, id)
	if err := row.Scan(&bo.Name, &bo.BagInfo, &bo.Size, &bo.SHA512, &bo.SHA512_AES, &bo.Creator, &creationDate); err != nil {
		return nil, errors.Wrapf(err, "cannot execute query %s", sqlstr)
	}
	bo.CreationDate = creationDate.Time

	sqlstr = fmt.Sprintf("SELECT l.locationid, l.name AS location, l.encrypted, l.quality, l.costs, bl.transfer_start, bl.transfer_end, bl.status, bl.message"+
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
		if err := rows.Scan(&boi.LocationID, &boi.Location, &encrypted, &boi.Quality, &boi.Costs,
			&transferStart, &transferEnd, &boi.Status, &boi.Message); err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "cannot scan query row %s", sqlstr)
		}
		boi.Encrypted = encrypted > 0
		boi.TransferStart = transferStart.Time
		boi.TransferDuration = transferEnd.Time.Sub(transferStart.Time)
		bo.Ingest = append(bo.Ingest, boi)
	}
	rows.Close()

	sqlstr = fmt.Sprintf("SELECT l.locationid, l.name AS location, t.name AS test, btl.start, btl.end, "+
		" btl.status, btl.message"+
		" FROM %s.bagit_test_location btl, %s.location l, %s.test t"+
		" WHERE btl.locationid=l.locationid AND btl.testid=t.testid AND btl.bagitid=?"+
		" ORDER BY btl.bagit_location_testid DESC", stat.schema, stat.schema, stat.schema)
	rows, err = stat.db.Query(sqlstr, id)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot execute query %s", sqlstr)
	}
	for rows.Next() {
		var boc = &BagitOverviewCheck{}
		var start, end sql.NullTime
		if err := rows.Scan(&boc.LocationID, &boc.Location, &boc.Test, &start, &end, &boc.Status, &boc.Message); err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "cannot scan query row %s", sqlstr)
		}
		boc.Start = start.Time
		boc.Duration = end.Time.Sub(start.Time)
		bo.Checks = append(bo.Checks, boc)
	}
	rows.Close()

	sqlstr = fmt.Sprintf("SELECT mimetype, SUM(filesize) AS size, COUNT(*) AS num "+
		" FROM %s.`content` "+
		" WHERE `bagitid`=? "+
		" GROUP BY mimetype "+
		" ORDER BY COUNT(*) DESC", stat.schema)
	rows, err = stat.db.Query(sqlstr, id)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot execute query %s", sqlstr)
	}
	for rows.Next() {
		var bom = &BagitOverviewMime{}
		if err := rows.Scan(&bom.Mimetype, &bom.Size, &bom.Count); err != nil {
			rows.Close()
			return nil, errors.Wrapf(err, "cannot scan query row %s", sqlstr)
		}
		bo.Mimetypes = append(bo.Mimetypes, bom)
	}
	rows.Close()

	return bo, nil
}
