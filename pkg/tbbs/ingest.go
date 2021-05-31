package tbbs

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/goph/emperror"
	"github.com/je4/bagarc/v2/pkg/bagit"
	"github.com/op/go-logging"
	"io/fs"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const encExt = "aes256"

type Ingest struct {
	tempDir     string
	logger      *logging.Logger
	keyDir      string
	reportDir   string
	db          *sql.DB
	schema      string
	initLocName string
	initLoc     *IngestLocation
	tests       map[string]*IngestTest
	locations   map[string]*IngestLocation
	sftp        *SFTP
}

func NewIngest(tempDir, keyDir, initLocName, reportDir string, db *sql.DB, dbschema string, privateKeys []string, logger *logging.Logger) (*Ingest, error) {
	sftp, err := NewSFTP(privateKeys, "", "", logger)
	if err != nil {
		return nil, emperror.Wrap(err, "cannot create sftp")
	}
	i := &Ingest{
		db:          db,
		tempDir:     tempDir,
		logger:      logger,
		keyDir:      keyDir,
		reportDir:   reportDir,
		schema:      dbschema,
		initLocName: initLocName,
		sftp:        sftp,
	}
	return i, i.Init()
}

func (i *Ingest) Init() error {
	var err error
	i.locations, err = i.locationLoadAll()
	if err != nil {
		return emperror.Wrapf(err, "cannot load locations")
	}

	i.tests, err = i.TestLoadAll()
	if err != nil {
		return emperror.Wrapf(err, "cannot load locations")
	}

	var ok bool
	i.initLoc, ok = i.locations[i.initLocName]
	if !ok {
		return fmt.Errorf("cannot get init location %s", i.initLoc)
	}
	return nil
}

func (i *Ingest) TestLoadAll() (map[string]*IngestTest, error) {
	sqlstr := fmt.Sprintf("SELECT testid, Name, description FROM %s.test", i.schema)

	var tests = make(map[string]*IngestTest)
	rows, err := i.db.Query(sqlstr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, emperror.Wrapf(err, "cannot get tests %s", sqlstr)
	}
	defer rows.Close()
	for rows.Next() {
		test := &IngestTest{}
		if err := rows.Scan(&test.id, &test.name, &test.description); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, emperror.Wrapf(err, "cannot get tests %s", sqlstr)
		}
		tests[test.name] = test
	}
	return tests, nil
}

/* Database functions */

func (i *Ingest) BagitLoadAll(fn func(bagit *IngestBagit) error) error {
	const pageSize int64 = 100
	sqlstr := fmt.Sprintf("SELECT COUNT(*) FROM %s.bagit", i.schema)
	row := i.db.QueryRow(sqlstr)
	var numRows int64
	if err := row.Scan(&numRows); err != nil {
		return emperror.Wrapf(err, "cannot get number of rows - %s", sqlstr)
	}

	sqlstr = fmt.Sprintf("SELECT bagitid, Name, filesize, sha512, sha512_aes, baginfo, creator, Report, creationdate FROM %s.bagit LIMIT ?,?", i.schema)

	var start int64
	for start = 0; start < numRows; start += pageSize {
		rows, err := i.db.Query(sqlstr, start, pageSize)
		if err != nil {
			return emperror.Wrapf(err, "cannot get locations %s", sqlstr)
		}
		var bagits []*IngestBagit
		for rows.Next() {
			bagit := &IngestBagit{
				ingest: i,
			}
			var sha512_aes, report sql.NullString
			if err := rows.Scan(&bagit.Id, &bagit.Name, &bagit.Size, &bagit.SHA512, &sha512_aes, &bagit.Baginfo, &bagit.Creator, &report, &bagit.Creationdate); err != nil {
				rows.Close()
				if err == sql.ErrNoRows {
					return nil
				}
				return emperror.Wrapf(err, "cannot get bagit %s", sqlstr)
			}
			bagit.SHA512_aes = sha512_aes.String
			bagit.Report = report.String
			bagits = append(bagits, bagit)
		}
		rows.Close()
		for _, bagit := range bagits {
			if err := fn(bagit); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *Ingest) testLoad(name string) (*IngestTest, error) {
	sqlstr := fmt.Sprintf("SELECT testid, Name, description FROM %s.test WHERE Name = ?", i.schema)
	row := i.db.QueryRow(sqlstr, name)
	test := &IngestTest{}
	if err := row.Scan(&test.id, &test.name, test.description); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, emperror.Wrapf(err, "cannot get location %s - %s", sqlstr, name)
	}
	return test, nil
}

func (i *Ingest) locationLoadAll() (map[string]*IngestLocation, error) {
	sqlstr := fmt.Sprintf("SELECT locationid, Name, path, params, encrypted, quality, costs, testinterval FROM %s.location", i.schema)

	var locations = make(map[string]*IngestLocation)

	rows, err := i.db.Query(sqlstr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, emperror.Wrapf(err, "cannot get locations %s", sqlstr)
	}
	defer rows.Close()
	for rows.Next() {
		loc := &IngestLocation{}
		var p, testIntervalStr string
		if err := rows.Scan(&loc.id, &loc.name, &p, &loc.params, &loc.encrypted, &loc.quality, &loc.costs, &testIntervalStr); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, emperror.Wrapf(err, "cannot get location %s", sqlstr)
		}
		loc.path, err = url.Parse(p)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot parse url %s", p)
		}
		loc.checkInterval, err = time.ParseDuration(testIntervalStr)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot parse interval %s of location %s", testIntervalStr, loc.name)
		}
		locations[loc.name] = loc
	}
	return locations, nil
}

func (i *Ingest) locationLoad(name string) (*IngestLocation, error) {
	sqlstr := fmt.Sprintf("SELECT locationid, Name, path, testinterval, params, encrypted, quality, costs FROM %s.location WHERE Name = ?", i.schema)
	row := i.db.QueryRow(sqlstr, name)
	loc := &IngestLocation{ingest: i}
	var err error
	var p, testIntervalStr string
	if err := row.Scan(&loc.id, &loc.name, &p, &testIntervalStr, &loc.params, &loc.encrypted, &loc.quality, &loc.costs); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, emperror.Wrapf(err, "cannot get location %s - %s", sqlstr, name)
	}
	loc.path, err = url.Parse(p)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot parse url %s", p)
	}
	loc.checkInterval, err = time.ParseDuration(testIntervalStr)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot parse interval %s", testIntervalStr)
	}
	return loc, nil
}

func (i *Ingest) locationStore(loc *IngestLocation) (*IngestLocation, error) {
	if loc.id == 0 {
		sqlstr := fmt.Sprintf("INSERT INTO %s.location(Name, path, testinterval, params, encrypted, quality, costs) VALUES(?, ?, ?, ?, ?, ?, ?, ?) returning locationid", i.schema)
		row := i.db.QueryRow(sqlstr, loc.name, loc.path.String(), loc.checkInterval.String(), loc.params, loc.encrypted, loc.quality, loc.costs)
		if err := row.Scan(&loc.id); err != nil {
			return nil, emperror.Wrapf(err, "cannot insert location %s - %s", sqlstr, loc.name)
		}
		return loc, nil
	} else {
		sqlstr := fmt.Sprintf("UPDATE %s.location SET Name=?, path=?, params=?, encrypted=?, quality=?, costs=? WHERE locationid=?", i.schema)
		if _, err := i.db.Exec(sqlstr, loc.name, loc.path.String(), loc.params, loc.encrypted, loc.quality, loc.costs, loc.id); err != nil {
			return nil, emperror.Wrapf(err, "cannot update location %s - %v", sqlstr, loc.id)
		}
	}
	return nil, fmt.Errorf("LocationStore() - strange things happen - %v", loc)
}

func (i *Ingest) transferLoad(loc *IngestLocation, bagit *IngestBagit) (*Transfer, error) {
	sqlstr := fmt.Sprintf("SELECT transfer_start, transfer_end, status, message FROM %s.bagit_location WHERE bagitid=? AND locationid=?",
		loc.ingest.schema)
	row := loc.ingest.db.QueryRow(sqlstr, loc.id, bagit.Id)
	trans := &Transfer{
		ingest: i,
		loc:    loc,
		bagit:  bagit,
	}
	var start, end sql.NullTime
	if err := row.Scan(&start, &end, &trans.status, &trans.message); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, emperror.Wrapf(err, "cannot load transfer - %s - %v, %v", sqlstr, loc.id, bagit.Id)
	}
	trans.start = start.Time
	trans.end = end.Time
	return trans, nil
}

func (i *Ingest) transferStore(transfer *Transfer) (*Transfer, error) {
	sqlstr := fmt.Sprintf("REPLACE INTO %s.transfer(bagitid, locationid, transfer_start, transfer_end, status, message) VALUES(?, ?, ?, ?, ?, ?)", i.schema)
	if _, err := i.db.Exec(sqlstr, transfer.bagit.Id, transfer.loc.id, transfer.start, transfer.end, transfer.status, transfer.message); err != nil {
		return nil, emperror.Wrapf(err, "cannot insert transfer %s - %s -> %s", sqlstr, transfer.loc.name, transfer.bagit.Name)
	}
	return transfer, nil
}

func (i *Ingest) bagitContentStore(ibc *IngestBagitContent) (*IngestBagitContent, error) {
	if ibc.contentId == 0 {
		var _mimetype, _indexer sql.NullString
		var _width, _height, _duration sql.NullInt64
		_indexer.Scan(ibc.Indexer)
		if ibc.Indexer == "" {
			_indexer.Valid = false
		}
		_mimetype.Scan(ibc.Mimetype)
		if ibc.Mimetype == "" {
			_mimetype.Valid = false
		}
		_width.Scan(ibc.Width)
		if ibc.Width == 0 {
			_width.Valid = false
		}
		_height.Scan(ibc.Height)
		if ibc.Height == 0 {
			_height.Valid = false
		}
		_duration.Scan(ibc.Duration)
		if ibc.Duration == 0 {
			_duration.Valid = false
		}
		sqlstr := fmt.Sprintf("INSERT INTO %s.content (bagitid, zippath, diskpath, filesize, sha256, sha512, md5, mimetype, width, height, duration, indexer) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", i.schema)
		res, err := i.db.Exec(sqlstr, ibc.bagit.Id, ibc.ZipPath, ibc.DiskPath, ibc.Filesize, ibc.SHA256, ibc.SHA512, ibc.MD5, _mimetype, _width, _height, _duration, _indexer)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot insert content of bagit %s at %s", ibc.bagit.Name, sqlstr)
		}
		ibc.contentId, err = res.LastInsertId()
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot get insert Id of content of bagit %s at %s - %s", ibc.bagit.Name, sqlstr)
		}
		return ibc, nil
	} else {
		return nil, fmt.Errorf("cannot store bagit content %v:%v", ibc.bagit.Name, ibc.DiskPath)
	}

}

func (i *Ingest) bagitStore(bagit *IngestBagit) (*IngestBagit, error) {
	if bagit.Id == 0 {
		var sha512_aes, report sql.NullString
		if bagit.SHA512_aes != "" {
			sha512_aes.String = bagit.SHA512_aes
			sha512_aes.Valid = true
		}
		if bagit.Report != "" {
			report.String = bagit.Report
			report.Valid = true
		}
		sqlstr := fmt.Sprintf("INSERT INTO %s.bagit(Name, filesize, sha512, sha512_aes, Report, baginfo, creator, creationdate) VALUES(?, ?, ?, ?, ?, ?, ?, ?)", i.schema)
		res, err := i.db.Exec(sqlstr, bagit.Name, bagit.Size, bagit.SHA512, sha512_aes, report, bagit.Baginfo, bagit.Creator, bagit.Creationdate)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot insert bagit %s - %s", sqlstr, bagit.Name)
		}
		bagit.Id, err = res.LastInsertId()
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot get last insert Id %s", sqlstr)
		}

		return bagit, nil
	} else {
		sqlstr := fmt.Sprintf("UPDATE %s.bagit SET Name=?, filesize=?, sha512=?, sha512_aes=?, Report=?, baginfo=?, creator=?, creationdate=? WHERE bagitid=?", i.schema)
		if _, err := i.db.Exec(sqlstr, bagit.Name, bagit.Size, bagit.SHA512, bagit.SHA512_aes, bagit.Report, bagit.Baginfo, bagit.Creator, bagit.Creationdate, bagit.Id); err != nil {
			return nil, emperror.Wrapf(err, "cannot update bagit %s", sqlstr)
		}
	}
	return nil, fmt.Errorf("BagitStore() - strange things happen - %v", bagit)
}

func (i *Ingest) bagitExistsAt(bagit *IngestBagit, location *IngestLocation) (bool, error) {
	sqlstr := fmt.Sprintf("SELECT COUNT(*) FROM bagit_location WHERE bagitid=? AND locationid=?", i.schema)
	row := i.db.QueryRow(sqlstr, bagit.Id, location.id)
	var num int64
	if err := row.Scan(&num); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, emperror.Wrapf(err, "cannot check bagit %v at location %v", bagit.Id, location.id)
	}
	return num > 0, nil
}

func (i *Ingest) hasBagit(loc *IngestLocation, bagit *IngestBagit) (bool, error) {
	sqlstr := fmt.Sprintf("SELECT COUNT(*) FROM %s.bagit_location bl, %s.bagit b WHERE bl.bagitid=b.bagitid AND bl.locationid=? AND b.bagitid=?",
		loc.ingest.schema,
		loc.ingest.schema)
	row := loc.ingest.db.QueryRow(sqlstr, loc.id, bagit.Id)
	var num int64
	if err := row.Scan(&num); err != nil {
		return false, emperror.Wrapf(err, "cannot check for bagit - %s - %v, %v", sqlstr, loc.id, bagit.Id)
	}
	return num > 0, nil
}

func (i *Ingest) bagitLocationStore(ibl *IngestBagitLocation) error {
	sqlstr := fmt.Sprintf("REPLACE INTO %s.bagit_location (bagitid, locationid, transfer_start, transfer_end, status, message) VALUES (?, ?, ?, ?, ?, ?)", i.schema)
	_, err := i.db.Exec(sqlstr, ibl.bagit.Id, ibl.location.id, ibl.start, ibl.end, ibl.status, ibl.message)
	if err != nil {
		return emperror.Wrapf(err, "cannot insert bagit %s at %s - %s", sqlstr, ibl.bagit.Name, ibl.location.name)
	}
	return nil
}

func (i *Ingest) bagitTestLocationStore(ibl *IngestBagitTestLocation) error {
	if ibl.id == 0 {
		sqlstr := fmt.Sprintf("INSERT INTO %s.bagit_test_location (bagitid, testid, locationid, start, end, status, data, message) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", i.schema)
		_, err := i.db.Exec(sqlstr, ibl.bagit.Id, ibl.test.id, ibl.location.id, ibl.start, ibl.end, ibl.status, ibl.data, ibl.message)
		if err != nil {
			return emperror.Wrapf(err, "cannot insert bagit_test_location - %s", sqlstr)
		}
	} else {
		sqlstr := fmt.Sprintf("REPLACE INTO %s.bagit_test_location (bagit_location_testid, bagitid, testid, locationid, start, end, status, data, message) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", i.schema)
		_, err := i.db.Exec(sqlstr, ibl.id, ibl.bagit.Id, ibl.test.id, ibl.location.id, ibl.start, ibl.end, ibl.status, ibl.data, ibl.message)
		if err != nil {
			return emperror.Wrapf(err, "cannot insert bagit_test_location - %s", sqlstr)
		}
	}
	return nil
}

func (i *Ingest) bagitTestLocationNeeded(ibl *IngestBagitTestLocation) (bool, error) {
	checkDate := time.Now().Local().Add(-ibl.location.checkInterval)
	sqlstr := fmt.Sprintf("SELECT COUNT(*) FROM %s.bagit_test_location WHERE status=? AND bagitid=? AND testid=? AND locationid=? AND start > ?", i.schema)
	row := i.db.QueryRow(sqlstr, "passed", ibl.bagit.Id, ibl.test.id, ibl.location.id, checkDate)
	var num int64
	if err := row.Scan(&num); err != nil {
		return false, emperror.Wrapf(err, "cannot check for test interval - %s", sqlstr)
	}
	return num == 0, nil
}

func (i *Ingest) bagitTestLocationLast(ibl *IngestBagitTestLocation) error {
	sqlstr := fmt.Sprintf("SELECT status, start, end, message FROM %s.bagit_test_location WHERE bagitid=? AND testid=? AND locationid=? ORDER BY bagit_location_testid DESC", i.schema)
	row := i.db.QueryRow(sqlstr, ibl.bagit.Id, ibl.test.id, ibl.location.id)
	if err := row.Scan(&ibl.status, &ibl.start, &ibl.end, &ibl.message); err != nil {
		return emperror.Wrapf(err, "cannot check for test interval - %s", sqlstr)
	}
	return nil
}

func (i *Ingest) bagitLoad(name string) (*IngestBagit, error) {
	sqlstr := fmt.Sprintf("SELECT bagitid, Name, filesize, sha512, sha512_aes, creator, baginfo, Report FROM %s.bagit WHERE Name=?", i.schema)
	row := i.db.QueryRow(sqlstr, name)
	bagit := &IngestBagit{
		ingest: i,
	}
	var sha512_aes, report sql.NullString
	if err := row.Scan(&bagit.Id, &bagit.Name, &bagit.Size, &bagit.SHA512, &sha512_aes, &bagit.Creator, &bagit.Baginfo, &report); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, emperror.Wrapf(err, "cannot get bagit %s - %s", sqlstr, name)
	}
	bagit.SHA512_aes = sha512_aes.String
	bagit.Report = report.String
	return bagit, nil
}

func (i *Ingest) bagitAddContent(bagit *IngestBagit, zippath string, diskpath string, filesize int64, sha256 string, sha512 string, md5 string, mimetype string, width int64, height int64, duration int64, indexer string) error {
	var _mimetype, _indexer sql.NullString
	var _width, _height, _duration sql.NullInt64
	_indexer.Scan(indexer)
	if indexer == "" {
		_indexer.Valid = false
	}
	_mimetype.Scan(mimetype)
	if mimetype == "" {
		_mimetype.Valid = false
	}
	_width.Scan(width)
	if width == 0 {
		_width.Valid = false
	}
	_height.Scan(height)
	if height == 0 {
		_height.Valid = false
	}
	_duration.Scan(duration)
	if duration == 0 {
		_duration.Valid = false
	}
	sqlstr := fmt.Sprintf("INSERT INTO %s.content (bagitid, zippath, diskpath, filesize, sha256, sha512, md5, mimetype, width, height, duration, indexer) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", i.schema)
	_, err := i.db.Exec(sqlstr, bagit.Id, zippath, diskpath, filesize, sha256, sha512, md5, mimetype, width, height, duration, indexer)
	if err != nil {
		return emperror.Wrapf(err, "cannot insert content of bagit %s at %s - %s", bagit.Name, sqlstr)
	}
	return nil

}

func (i *Ingest) bagitLocationLoad(bagit *IngestBagit, location *IngestLocation) (*IngestBagitLocation, error) {
	sqlstr := fmt.Sprintf("SELECT transfer_start, transfer_end, status, message FROM %s.bagit_location WHERE bagitid=? AND locationid=?", i.schema)
	row := i.db.QueryRow(sqlstr, bagit.Id, location.id)
	ibl, err := NewIngestBagitLocation(i, bagit, location)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot create ingestbagitlocation")
	}
	var start, end sql.NullTime
	if err := row.Scan(&start, &end, &ibl.status, &ibl.message); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, emperror.Wrapf(err, "cannot get bagit %s - %s - %s", sqlstr, bagit.Name, location.name)
	}
	ibl.start = start.Time
	ibl.end = end.Time
	return ibl, nil
}

/* Creator functions */

func (i *Ingest) BagitNew(name string, size int64, sha512, sha512_aes, report, creator, baginfo string, creationtime time.Time) (*IngestBagit, error) {
	bagit := &IngestBagit{
		ingest:       i,
		Id:           0,
		Name:         name,
		Size:         size,
		SHA512:       sha512,
		SHA512_aes:   sha512_aes,
		Report:       report,
		Baginfo:      baginfo,
		Creator:      creator,
		Creationdate: creationtime,
	}
	return bagit, nil
}

func (i *Ingest) IngestBagitLocationNew(bagit *IngestBagit, loc *IngestLocation) (*IngestBagitLocation, error) {
	return NewIngestBagitLocation(i, bagit, loc)
}

/* Actions */

func (i *Ingest) Transfer() error {
	if err := i.BagitLoadAll(func(bagit *IngestBagit) error {
		source, err := i.bagitLocationLoad(bagit, i.initLoc)
		if err != nil {
			return emperror.Wrapf(err, "cannot load initial transfer of %s - %s", bagit.Name, i.initLoc.name)
		}

		for key, loc := range i.locations {
			if key == i.initLocName {
				continue
			}
			target, err := i.IngestBagitLocationNew(bagit, loc)
			if err != nil {
				return emperror.Wrapf(err, "cannot create transfer object")
			}
			if exists, _ := target.Exists(); exists {
				i.logger.Infof("bagit %s in %s already exists", bagit.Name, loc.name)
				continue
			}
			if err := target.Transfer(source); err != nil {
				return emperror.Wrapf(err, "cannot transfer bagit %s to %s", bagit.Name, target.location.name)
			}
		}
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error iterating bagits")
	}
	return nil
}

func (i *Ingest) Ingest() error {
	// get path of init location
	fp := strings.Trim(i.initLoc.GetPath().Path, "/") + "/"
	if runtime.GOOS == "windows" {
		fp = strings.Replace(fp, "|", ":", -1)
	} else {
		fp = "/" + fp
	}
	// walk through the path
	if err := filepath.Walk(fp, func(path string, info fs.FileInfo, err error) error {
		// ignore directory
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		// hope: there's no empty filename
		// ignore . files
		if name[0] == '.' {
			return nil
		}

		// ignore any non-zip file
		if !strings.HasSuffix(name, ".zip") {
			return nil
		}

		// if it's already in database ignore file
		ib, err := i.bagitLoad(name)
		if err != nil {
			return emperror.Wrapf(err, "cannot load bagit %s", name)
		}
		if ib != nil {
			i.logger.Infof("bagit %s already ingested", name)
			return nil
		}

		bagitPath := path

		// create tempdir for database
		tmpdir, err := ioutil.TempDir(i.tempDir, filepath.Base(bagitPath))
		if err != nil {
			return emperror.Wrapf(err, "cannot create temporary folder in %s", i.tempDir)
		}

		// initialize badger database
		bconfig := badger.DefaultOptions(filepath.Join(tmpdir, "/badger"))
		bconfig.Logger = i.logger // use our logger...
		checkDB, err := badger.Open(bconfig)
		if err != nil {
			return emperror.Wrapf(err, "cannot open badger database")
		}
		// close database and delete temporary files
		defer func() {
			checkDB.Close()
			if err := os.RemoveAll(tmpdir); err != nil {
				i.logger.Errorf("cannot remove %s: %v", tmpdir, err)
			}
		}()

		// check bagit file
		checker, err := bagit.NewBagit(bagitPath, tmpdir, checkDB, i.logger)
		i.logger.Infof("deep checking bagit file %s", bagitPath)

		var metaBytes bytes.Buffer
		metaWriter := bufio.NewWriter(&metaBytes)
		var baginfoBytes bytes.Buffer
		baginfoWriter := bufio.NewWriter(&baginfoBytes)
		if err := checker.Check(metaWriter, baginfoWriter); err != nil {
			return emperror.Wrapf(err, "error checking file %v", bagitPath)
		}
		// paranoia
		metaWriter.Flush()
		baginfoWriter.Flush()

		type Metadata []struct {
			Path     string                 `json:"path"`
			Zippath  string                 `json:"zippath"`
			Checksum map[string]string      `json:"checksum"`
			Size     int64                  `json:"Size"`
			Indexer  map[string]interface{} `json:"indexer"`
		}
		var metadata Metadata

		if err := json.Unmarshal(metaBytes.Bytes(), &metadata); err != nil {
			return emperror.Wrapf(err, "cannot unmarshal json %s", metaBytes.String())
		}

		// create checksum
		cs, err := bagit.SHA512(bagitPath)
		if err != nil {
			return emperror.Wrap(err, "cannot calculate checksum")
		}

		// create bagit ingest object
		bagit, err := i.BagitNew(name, info.Size(), cs, "", "", "bagarc", baginfoBytes.String(), time.Now())
		if err != nil {
			return emperror.Wrapf(err, "cannot create bagit entity %s", name)
		}
		// store it in database
		if err := bagit.Store(); err != nil {
			return emperror.Wrapf(err, "cannot store %s", name)
		}

		ibl, err := i.IngestBagitLocationNew(bagit, i.initLoc)
		if err != nil {
			return emperror.Wrapf(err, "cannot create initial ingestbagitlocation")
		}
		if err := ibl.SetData("ok", "initial ingest location", time.Now().Local(), time.Now().Local()); err != nil {
			return emperror.Wrapf(err, "cannot set data for ingestbagitlocation")
		}
		if err := ibl.Store(); err != nil {
			return emperror.Wrapf(err, "cannot store initial bagit location")
		}

		for _, meta := range metadata {
			var sha256, sha512, md5, mimetype, indexer string
			var width, height, duration int64
			if str, ok := meta.Checksum["sha256"]; ok {
				sha256 = str
			}
			if str, ok := meta.Checksum["SHA512"]; ok {
				sha512 = str
			}
			if str, ok := meta.Checksum["md5"]; ok {
				md5 = str
			}
			if i, ok := meta.Indexer["mimetype"]; ok {
				mimetype, ok = i.(string)
			}
			if i, ok := meta.Indexer["width"]; ok {
				if fl, ok := i.(float64); ok {
					width = int64(fl)
				}
			}
			if i, ok := meta.Indexer["height"]; ok {
				if fl, ok := i.(float64); ok {
					height = int64(fl)
				}
			}
			if i, ok := meta.Indexer["duration"]; ok {
				if fl, ok := i.(float64); ok {
					duration = int64(fl)
				}
			}
			if data, err := json.Marshal(meta.Indexer); err == nil {
				indexer = string(data)
			}

			bagit.AddContent(meta.Zippath, meta.Path, meta.Size, sha256, sha512, md5, mimetype, width, height, duration, indexer)
		}

		/*
			if err := bi.Encrypt(Name, bagitPath); err != nil {
				return emperror.Wrapf(err, "cannot encrypt %s", bagitPath)
			}
		*/
		return nil
	}); err != nil {
		return emperror.Wrapf(err, "error walking %s", fp)
	}
	return nil
}

func (i *Ingest) Check() error {
	daTest, ok := i.tests["checksum"]
	if !ok {
		return fmt.Errorf("cannot find test checksum")
	}
	if err := i.BagitLoadAll(func(bagit *IngestBagit) error {
		for _, loc := range i.locations {
			t, err := i.IngestBagitTestLocationNew(bagit, loc, daTest)
			if err != nil {
				return emperror.Wrapf(err, "cannot create test for %s at %s", bagit.Name, loc.name)
			}
			if err := t.Test(); err != nil {
				return emperror.Wrapf(err, "cannot check %s at %s", bagit.Name, loc.name)
			}
		}
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error iterating bagits")
	}
	return nil
}

func (i *Ingest) DELETE_Encrypt(name, bagitPath string) error {
	if _, err := os.Stat(bagitPath + "." + encExt); err == nil {
		return fmt.Errorf("encrypted bagit file %s.%s already exists", name, encExt)
	} else if !os.IsNotExist(err) {
		return emperror.Wrapf(err, "error checking existence of %s.%s", name, encExt)
	}

	// create checksum of bagit
	//		bi.logger.Infof("calculating checksum of %s", bagitFile)
	//		checksum := SHA512.New()

	fin, err := os.Open(bagitPath)
	if err != nil {
		return emperror.Wrapf(err, "cannot open %s", bagitPath)
	}
	defer fin.Close()
	fout, err := os.OpenFile(bagitPath+"."+encExt, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return emperror.Wrapf(err, "cannot create encrypted bagit %s.%s", bagitPath, encExt)
	}
	defer fout.Close()
	if err != nil {
		return emperror.Wrapf(err, "cannot open %s.%s", bagitPath, encExt)
	}
	defer fout.Close()

	key, iv, hashBytes, err := Encrypt(fin, fout)
	if err != nil {
		return emperror.Wrapf(err, "cannot encrypt %s", bagitPath)
	}

	os.WriteFile(fmt.Sprintf("%s/%s.%s.key", i.keyDir, name, encExt), []byte(fmt.Sprintf("%x", key)), 0600)
	os.WriteFile(fmt.Sprintf("%s/%s.%s.iv", i.keyDir, name, encExt), []byte(fmt.Sprintf("%x", iv)), 0600)

	i.logger.Infof("key: %x, iv: %x, hash: %x", key, iv, hashBytes)
	i.logger.Infof("decrypt using openssl: \n openssl enc -aes-256-ctr -nosalt -d -in %s.%s -out %s -K '%x' -iv '%x'", bagitPath, encExt, bagitPath, key, iv)

	/*
		if _, err := io.Copy(checksum, fin); err != nil {
			return emperror.Wrapf(err, "cannot calculate checksum of %s", bagitFile)
		}
	*/
	return nil
}

func (i *Ingest) IngestBagitTestLocationNew(ingestBagit *IngestBagit, loc *IngestLocation, test *IngestTest) (*IngestBagitTestLocation, error) {
	t := &IngestBagitTestLocation{ingest: i, bagit: ingestBagit, location: loc, test: test}
	return t, nil
}
