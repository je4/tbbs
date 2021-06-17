package tbbs

import (
	"crypto/sha512"
	"fmt"
	"github.com/goph/emperror"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type IngestBagitTestLocation struct {
	id                    int64
	ingest                *Ingest
	bagit                 *IngestBagit
	location              *IngestLocation
	test                  *IngestTest
	status, message, data string
	start, end            time.Time
}

func NewIngestBagitTestLocation(ingest *Ingest, bagit *IngestBagit, test *IngestTest, location *IngestLocation) (*IngestBagitTestLocation, error) {
	ibl := &IngestBagitTestLocation{ingest: ingest, bagit: bagit, test: test, location: location}
	return ibl, nil
}

func (ibl *IngestBagitTestLocation) SetData(status, message, data string, start, end time.Time) error {
	ibl.start = start
	ibl.end = end
	ibl.status = status
	ibl.message = message
	ibl.data = data
	return nil
}

func (ibl *IngestBagitTestLocation) Store() error {
	return ibl.ingest.bagitTestLocationStore(ibl)
}

func (ibl *IngestBagitTestLocation) Last() error {
	return ibl.ingest.bagitTestLocationLast(ibl)
}

func (ibl *IngestBagitTestLocation) checksumSFTP() ([]byte, error) {
	shaSink := sha512.New()
	urlstring := ibl.location.path.String()
	urlstring += "/" + ibl.bagit.Name
	if ibl.location.IsEncrypted() {
		urlstring += "." + encExt
	}
	u, err := url.Parse(urlstring)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot parse url %s", urlstring)
	}
	if _, err := ibl.ingest.sftp.Get(u, shaSink); err != nil {
		return nil, emperror.Wrapf(err, "cannot generate checksum of %s at %s", ibl.bagit.Name, ibl.location.name)
	}
	return shaSink.Sum(nil), nil
}

func (ibl *IngestBagitTestLocation) checksumFile() ([]byte, error) {
	path := strings.Trim(ibl.location.path.Path, "/")
	if runtime.GOOS == "windows" {
		path = strings.Replace(path, "|", ":", -1)
	} else {
		path = "/" + path
	}
	path = filepath.Join(path, ibl.bagit.Name)
	if ibl.location.IsEncrypted() {
		path += "." + encExt
	}
	fp, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot open %s", path)
	}
	defer fp.Close()
	shaSink := sha512.New()
	if _, err := io.Copy(shaSink, fp); err != nil {
		return nil, emperror.Wrapf(err, "cannot create checksum of %s at %s", ibl.bagit.Name, ibl.location.name)
	}
	return shaSink.Sum(nil), nil
}

func (ibl *IngestBagitTestLocation) Test() error {
	testNeeded, err := ibl.ingest.bagitTestLocationNeeded(ibl)
	if err != nil {
		return emperror.Wrap(err, "cannot check for test need")
	}
	if !testNeeded {
		ibl.ingest.logger.Infof("no %s test needed for %s at %s", ibl.test.name, ibl.bagit.Name, ibl.location.name)
		return nil
	}
	if ibl.test.name != "checksum" {
		return fmt.Errorf("invalid test %s", ibl.test.name)
	}
	ibl.start = time.Now()

	var targetChecksum, checksum string

	switch ibl.location.path.Scheme {
	case "sftp":
		checksumBytes, err := ibl.checksumSFTP()
		if err != nil {
			ibl.message = fmt.Sprintf("cannot get checksum of %s at %s: %v", ibl.bagit.Name, ibl.location.name, err)
		} else {
			checksum = fmt.Sprintf("%x", checksumBytes)
		}
	case "file":
		checksumBytes, err := ibl.checksumFile()
		if err != nil {
			ibl.message = fmt.Sprintf("cannot get checksum of %s at %s: %v", ibl.bagit.Name, ibl.location.name, err)
		} else {
			checksum = fmt.Sprintf("%x", checksumBytes)
		}
	default:
		return fmt.Errorf("cannot handle location %s with protocol %s", ibl.location.name, ibl.location.path.Scheme)
	}

	if ibl.location.IsEncrypted() {
		targetChecksum = ibl.bagit.SHA512_aes
	} else {
		targetChecksum = ibl.bagit.SHA512
	}

	if checksum == targetChecksum && targetChecksum != "" {
		ibl.status = "passed"
	} else {
		ibl.status = "failed"
		if len(ibl.message) > 0 {
			ibl.message += " // "
		}
		ibl.message += fmt.Sprintf("invalid checksum %s", checksum)
	}
	ibl.end = time.Now()
	if err := ibl.Store(); err != nil {
		return emperror.Wrapf(err, "cannot store test for %s at %s", ibl.bagit.Name, ibl.location.name)
	}
	ibl.ingest.logger.Infof("checksum for %s at %s %s", ibl.bagit.Name, ibl.location.name, ibl.status)
	return nil
}
