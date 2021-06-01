package tbbs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/goph/emperror"
	"os"
	"path/filepath"
	"time"
)

//
// IngestBagit
//
type IngestBagit struct {
	ingest       *Ingest
	Id           int64
	Name         string
	Size         int64
	SHA512       string
	SHA512_aes   string
	Report       string
	Creator      string
	Creationdate time.Time
	Baginfo      string
}

func (bagit *IngestBagit) Store() error {
	_, err := bagit.ingest.bagitStore(bagit)
	return err
}

func (bagit *IngestBagit) ExistsAt(location *IngestLocation) (bool, error) {
	return bagit.ingest.bagitExistsAt(bagit, location)
}

func (bagit *IngestBagit) Check(location *IngestLocation, checkInterval time.Duration) (bool, error) {
	exists, err := bagit.ExistsAt(location)
	if err != nil {
		return false, emperror.Wrapf(err, "cannot check bagit location %v", location.name)
	}
	if !exists {
		return false, fmt.Errorf("bagit %v not at location %v", bagit.Name, location.name)
	}

	return true, nil
}

func (bagit *IngestBagit) AddContent(zippath, diskpath string, filesize int64, checksums map[string]string, mimetype string, width, height, duration int64, indexer string) error {
	return bagit.ingest.bagitAddContent(bagit, zippath, diskpath, filesize, checksums, mimetype, width, height, duration, indexer)
}

func (bagit *IngestBagit) GetKey() []byte {
	key, err := os.ReadFile(filepath.Join(bagit.ingest.keyDir, bagit.Name+".key"))
	if err != nil {
		return nil
	}
	return key
}

func (bagit *IngestBagit) SetKey(key []byte) error {
	fname := filepath.Join(bagit.ingest.keyDir, bagit.Name+".key")
	if err := os.WriteFile(fname, key, 0600); err != nil {
		return emperror.Wrapf(err, "cannot write file %s", fname)
	}
	return nil
}

func (bagit *IngestBagit) GetIV() []byte {
	iv, err := os.ReadFile(filepath.Join(bagit.ingest.keyDir, bagit.Name+".iv"))
	if err != nil {
		return nil
	}
	return iv
}

func (bagit *IngestBagit) SetIV(iv []byte) error {
	fname := filepath.Join(bagit.ingest.keyDir, bagit.Name+".iv")
	if err := os.WriteFile(fname, iv, 0600); err != nil {
		return emperror.Wrapf(err, "cannot write file %s", fname)
	}
	return nil
}

func (bagit *IngestBagit) TestLoadAll(loc *IngestLocation, fn func(test *IngestBagitTestLocation) error) error {
	const pageSize int64 = 100
	sqlstr := fmt.Sprintf("SELECT COUNT(*) FROM %s.bagit_test_location btl, %s.test t WHERE btl.testid=t.testid AND btl.bagitid=? AND btl.locationid=?", bagit.ingest.schema, bagit.ingest.schema)
	row := bagit.ingest.db.QueryRow(sqlstr, bagit.Id, loc.id)
	var numRows int64
	if err := row.Scan(&numRows); err != nil {
		return emperror.Wrapf(err, "cannot get number of rows - %s", sqlstr)
	}

	sqlstr = fmt.Sprintf("SELECT btl.bagit_location_testid, t.name, btl.start, btl.end, btl.status, btl.message, btl.data "+
		"FROM %s.bagit_test_location btl, %s.test t "+
		"WHERE btl.testid=t.testid AND btl.bagitid=? AND btl.locationid=? LIMIT ?,?", bagit.ingest.schema, bagit.ingest.schema)

	var start int64
	for start = 0; start < numRows; start += pageSize {
		rows, err := bagit.ingest.db.Query(sqlstr, bagit.Id, loc.id, start, pageSize)
		if err != nil {
			return emperror.Wrapf(err, "cannot get content %s", sqlstr)
		}
		var tests []*IngestBagitTestLocation
		for rows.Next() {
			test := &IngestBagitTestLocation{
				ingest:   bagit.ingest,
				bagit:    bagit,
				location: loc,
			}
			var start, end sql.NullTime
			var data, status, message sql.NullString
			var testname string
			if err := rows.Scan(&test.id, &testname, &start, &end, &status, &message, &data); err != nil {
				rows.Close()
				if err == sql.ErrNoRows {
					return nil
				}
				return emperror.Wrapf(err, "cannot get bagit %s", sqlstr)
			}
			var ok bool
			test.test, ok = bagit.ingest.tests[testname]
			if !ok {
				return fmt.Errorf("invalid test %s", testname)
			}
			test.status = status.String
			test.message = message.String
			test.data = data.String
			test.start = start.Time
			test.end = end.Time
			tests = append(tests, test)
		}
		rows.Close()
		for _, test := range tests {
			if err := fn(test); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bagit *IngestBagit) ContentLoadAll(fn func(content *IngestBagitContent) error) error {
	const pageSize int64 = 100
	sqlstr := fmt.Sprintf("SELECT COUNT(*) FROM %s.content WHERE bagitid=?", bagit.ingest.schema)
	row := bagit.ingest.db.QueryRow(sqlstr, bagit.Id)
	var numRows int64
	if err := row.Scan(&numRows); err != nil {
		return emperror.Wrapf(err, "cannot get number of rows - %s", sqlstr)
	}

	sqlstr = fmt.Sprintf("SELECT contentid, zippath, diskpath, filesize, checksums, mimetype, width, height, duration, indexer "+
		"FROM %s.content "+
		"WHERE bagitid=? LIMIT ?,?", bagit.ingest.schema)

	var start int64
	for start = 0; start < numRows; start += pageSize {
		rows, err := bagit.ingest.db.Query(sqlstr, bagit.Id, start, pageSize)
		if err != nil {
			return emperror.Wrapf(err, "cannot get content %s", sqlstr)
		}
		var contents []*IngestBagitContent
		for rows.Next() {
			content := &IngestBagitContent{
				bagit: bagit,
			}
			var zippath, checksumstr, mimetype, indexer sql.NullString
			var width, height, duration sql.NullInt64
			if err := rows.Scan(&content.contentId, &zippath, &content.DiskPath, &content.Filesize, &checksumstr, &mimetype, &width, &height, &duration, &indexer); err != nil {
				rows.Close()
				if err == sql.ErrNoRows {
					return nil
				}
				return emperror.Wrapf(err, "cannot get bagit %s", sqlstr)
			}
			var checksums = make(map[string]string)
			if err := json.Unmarshal([]byte(checksumstr.String), &checksums); err != nil {
				return emperror.Wrapf(err, "cannot unmarshal checksum %s", checksumstr.String)
			}
			content.Checksums = checksums
			content.ZipPath = zippath.String
			content.Mimetype = mimetype.String
			content.Indexer = indexer.String
			content.Width = width.Int64
			content.Height = height.Int64
			content.Duration = duration.Int64
			contents = append(contents, content)
		}
		rows.Close()
		for _, content := range contents {
			if err := fn(content); err != nil {
				return err
			}
		}
	}

	return nil
}
