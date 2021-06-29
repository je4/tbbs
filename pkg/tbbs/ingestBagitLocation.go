package tbbs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"github.com/blend/go-sdk/crypto"
	"github.com/goph/emperror"
	xstream "github.com/je4/sftp/v2/pkg/stream"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type IngestBagitLocation struct {
	ingest          *Ingest
	bagit           *IngestBagit
	location        *IngestLocation
	status, message string
	start, end      time.Time
}

func NewIngestBagitLocation(ingest *Ingest, bagit *IngestBagit, location *IngestLocation) (*IngestBagitLocation, error) {
	ibl := &IngestBagitLocation{ingest: ingest, bagit: bagit, location: location}
	return ibl, nil
}

func (ibl *IngestBagitLocation) SetData(status, message string, start, end time.Time) error {
	ibl.start = start
	ibl.end = end
	ibl.status = status
	ibl.message = message
	return nil
}

func (ibl *IngestBagitLocation) Exists() (bool, error) {
	o, err := ibl.ingest.bagitLocationLoad(ibl.bagit, ibl.location)
	if err != nil {
		return false, emperror.Wrapf(err, "cannot load bagitLocation %s - %s", ibl.bagit.Name, ibl.location.name)
	}
	if o != nil {
		return o.status == "ok", nil
	} else {
		return false, nil
	}
}

func (ibl *IngestBagitLocation) createEncrypt() (*xstream.EncryptReader, error) {
	var err error
	key := ibl.bagit.GetKey()
	if key == nil {
		key, err = crypto.CreateKey(crypto.DefaultKeySize)
		if err != nil {
			return nil, emperror.Wrap(err, "cannot generate key")
		}
		if err := ibl.bagit.SetKey(key); err != nil {
			return nil, emperror.Wrap(err, "cannot write key")
		}
	}
	iv := ibl.bagit.GetIV()
	if iv == nil {
		iv = make([]byte, aes.BlockSize)
		_, err = rand.Read(iv)
		if err != nil {
			return nil, emperror.Wrap(err, "cannot create iv")
		}
		if err := ibl.bagit.SetIV(iv); err != nil {
			return nil, emperror.Wrap(err, "cannot write iv")
		}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		ibl.status = "error"
		ibl.Store()
		return nil, emperror.Wrap(err, "cannot create cipher block")
	}
	ctrStream := cipher.NewCTR(block, iv)
	mac := hmac.New(sha256.New, key)
	er := xstream.NewEncryptReader(block, ctrStream, mac, iv, ibl.ingest.logger)
	return er, nil
}

func (ibl *IngestBagitLocation) Transfer(source *IngestBagitLocation) error {
	// can only handle file source
	if source.location.path.Scheme != "file" {
		return fmt.Errorf("cannot copy from %s location of %s", source.location.path.Scheme, source.location.name)
	}

	// build source path
	sourceFolder := strings.Trim(source.location.GetPath().Path, "/") + "/"
	if runtime.GOOS == "windows" {
		sourceFolder = strings.Replace(sourceFolder, "|", ":", -1)
	} else {
		sourceFolder = "/" + sourceFolder
	}
	sourcePath := filepath.Join(sourceFolder, ibl.bagit.Name)

	// check existence of source
	info, err := os.Stat(sourcePath)
	if err != nil {
		return emperror.Wrapf(err, "cannot stat %s", sourcePath)
	}
	if info.IsDir() {
		return fmt.Errorf("source is a directory - %s", sourcePath)
	}

	ibl.start = time.Now()
	ibl.message = ""
	var message string

	src, err := os.OpenFile(sourcePath, os.O_RDONLY, 0666)
	if err != nil {
		ibl.status = "error"
		ibl.Store()
		return emperror.Wrapf(err, "cannot open source file %s", sourcePath)
	}
	defer src.Close()

	rsc, err := xstream.NewReadStreamQueue()
	if err != nil {
		ibl.status = "error"
		ibl.Store()
		return emperror.Wrap(err, "cannot create read stream queue")
	}

	if ibl.location.IsEncrypted() {
		/* Encryption */
		er, err := ibl.createEncrypt()
		if err != nil {
			return emperror.Wrap(err, "cannot create encryption pipeline")
		}
		rsc.Append(er)
	}

	/* ProgressReader Bar */
	stat, err := src.Stat()
	if err != nil {
		return emperror.Wrapf(err, "cannot stat %s", sourcePath)
	}
	rsc.Append(xstream.NewProgressReaderWriter(
		stat.Size(),
		time.Second,
		func(remaining time.Duration, percent float64, estimated time.Time, complete bool) {
			if !complete {
				fmt.Printf("\r% 3d%% - % 3dsec   ", int(math.Round(percent)+1), int(math.Round(float64(estimated.Sub(time.Now()))/float64(time.Second))))
			} else {
				fmt.Print("\r                                \r")
			}
		}),
	)

	/* ChecksumReaderWriter */
	hashSha512 := sha512.New()
	rsc.Append(xstream.NewChecksumReaderWriter(hashSha512, ibl.ingest.logger))

	switch strings.ToLower(ibl.location.path.Scheme) {
	case "file":
		// build target path
		targetFolder := strings.Trim(ibl.location.GetPath().Path, "/") + "/"
		if runtime.GOOS == "windows" {
			targetFolder = strings.Replace(targetFolder, "|", ":", -1)
		} else {
			targetFolder = "/" + targetFolder
		}
		targetPath := filepath.Join(targetFolder, ibl.bagit.Name)
		if ibl.location.IsEncrypted() {
			targetPath += "." + encExt
		}
		dest, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			ibl.status = "error"
			ibl.Store()
			return emperror.Wrapf(err, "cannot create destination file %s", targetPath)
		}
		defer dest.Close()
		ibl.ingest.logger.Infof("copying %s --> %s", sourcePath, targetPath)
		if ibl.location.IsEncrypted() {
			message = fmt.Sprintf("decrypt using openssl: \n openssl enc -aes-256-ctr -nosalt -d -in %s.%s -out %s -K '`cat %s/%s.key`' -iv '`cat %s/%s.iv`'", ibl.bagit.Name, encExt, ibl.bagit.Name, ibl.ingest.keyDir, ibl.bagit.Name, ibl.ingest.keyDir, ibl.bagit.Name)
		}
		size, err := io.Copy(dest, rsc.StartReader(src))
		if err != nil {
			ibl.status = "error"
			ibl.Store()
			return emperror.Wrapf(err, "cannot copy %s --> %s", sourcePath, targetPath)
		}
		ibl.message = fmt.Sprintf("copied %v bytes: %s --> %s", size, sourcePath, targetPath)
		ibl.ingest.logger.Infof("copying %v bytes", size)
		if message != "" {
			ibl.ingest.logger.Infof(message)
		}
	case "sftp":
		targetUrlStr := strings.TrimRight(ibl.location.path.String(), "/") + "/" + ibl.bagit.Name
		if ibl.location.IsEncrypted() {
			targetUrlStr += "." + encExt
		}
		targetUrl, err := url.Parse(targetUrlStr)
		if err != nil {
			ibl.status = "error"
			ibl.Store()
			return emperror.Wrapf(err, "cannot parse url %s", targetUrlStr)
		}
		ibl.ingest.logger.Infof("copying %s --> %s", sourcePath, targetUrl.String())

		size, err := ibl.ingest.sftp.Put(targetUrl, rsc.StartReader(src))
		if err != nil {
			ibl.status = "error"
			ibl.Store()
			return emperror.Wrapf(err, "cannot put %s --> %s", sourcePath, targetUrl.String())
		}
		ibl.message = fmt.Sprintf("copied %v bytes: %s --> %s", size, sourcePath, targetUrl.String())
		ibl.ingest.logger.Infof("copying %v bytes", size)
		if message != "" {
			ibl.ingest.logger.Infof(message)
		}
	default:
		return fmt.Errorf("invalid target scheme %s", ibl.location.path.Scheme)
	}
	sha512Str := fmt.Sprintf("%x", hashSha512.Sum(nil))
	if ibl.location.IsEncrypted() {
		if ibl.bagit.SHA512_aes == "" {
			ibl.bagit.SHA512_aes = sha512Str
			ibl.bagit.Store()
		} else {
			if sha512Str != ibl.bagit.SHA512_aes {
				return emperror.Wrapf(err, "invalid checksum %s != %s", ibl.bagit.SHA512_aes, sha512Str)
			}
		}
	} else {
		if sha512Str != ibl.bagit.SHA512 {
			return emperror.Wrapf(err, "invalid checksum %s != %s", ibl.bagit.SHA512, sha512Str)
		}
	}

	ibl.end = time.Now()
	ibl.status = "ok"
	return ibl.Store()
}

func (ibl *IngestBagitLocation) Store() error {
	return ibl.ingest.bagitLocationStore(ibl)
}
