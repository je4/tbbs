package tbbs

import (
	"database/sql"
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

func (bagit *IngestBagit) AddContent(zippath, diskpath string, filesize int64, sha256, sha512, md5, mimetype string, width, height, duration int64, indexer string) error {
	return bagit.ingest.bagitAddContent(bagit, zippath, diskpath, filesize, sha256, sha512, md5, mimetype, width, height, duration, indexer)
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

func (bagit *IngestBagit) ContentLoadAll(fn func(content *IngestBagitContent) error) error {
	const pageSize int64 = 100
	sqlstr := fmt.Sprintf("SELECT COUNT(*) FROM %s.content WHERE bagitid=?", bagit.ingest.schema)
	row := bagit.ingest.db.QueryRow(sqlstr, bagit.Id)
	var numRows int64
	if err := row.Scan(&numRows); err != nil {
		return emperror.Wrapf(err, "cannot get number of rows - %s", sqlstr)
	}

	sqlstr = fmt.Sprintf("SELECT contentid, zippath, diskpath, filesize, sha256, sha512, md5, mimetype, width, height, duration, indexer "+
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
			var zippath, sha256, sha512, md5, mimetype, indexer sql.NullString
			var width, height, duration sql.NullInt64
			if err := rows.Scan(&content.contentId, &zippath, &content.DiskPath, &content.Filesize, &sha256, &sha512, &md5, &mimetype, &width, &height, &duration, &indexer); err != nil {
				rows.Close()
				if err == sql.ErrNoRows {
					return nil
				}
				return emperror.Wrapf(err, "cannot get bagit %s", sqlstr)
			}
			content.ZipPath = zippath.String
			content.SHA256 = sha256.String
			content.SHA512 = sha512.String
			content.MD5 = md5.String
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
