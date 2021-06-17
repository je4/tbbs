package tbbs

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/goph/emperror"
	ffmpeg_models "github.com/je4/goffmpeg/models"
	"github.com/je4/indexer/pkg/indexer"
	siegfried_pronom "github.com/richardlehane/siegfried/pkg/pronom"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// Embed the entire directory.
//go:embed templates/* templates/reportbase/*
var templates embed.FS

func splitLine(str string, maxWidth int) []string {
	if maxWidth == 0 {
		return []string{str}
	}
	words := strings.Split(str, " ")
	lines := []string{""}
	line := 0
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(lines[line])+len(word) >= maxWidth {
			line++
			lines = append(lines, "")
		}
		lines[line] += " " + word
	}
	result := []string{}
	for _, s := range lines {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			result = append(result, s)
		}
	}
	return result
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

type TplBagitKeyVal struct {
	Key, Value string
}

func (tbkv TplBagitKeyVal) Cols() []string {
	return []string{"key", "value"}
}
func (tbkv TplBagitKeyVal) FieldSize(col string, maxWidth int) (w int, h int) {
	data := tbkv.Data(col, maxWidth)
	h = len(data)
	if h == 0 {
		return 0, 0
	}
	for l := 0; l < h; l++ {
		_w := len(data[l])
		if _w > w {
			w = _w
		}
	}
	return
}

func (tbkv TplBagitKeyVal) Data(col string, maxWidth int) []string {
	data := []string{}
	switch col {
	case "key":
		data = splitLine(tbkv.Key, maxWidth)
	case "value":
		data = splitLine(tbkv.Value, maxWidth)
	}
	return data
}
func (tbkv TplBagitKeyVal) Title(col string) string {
	result := ""
	switch col {
	case "key":
		result = "Key"
	case "value":
		result = "Value"
	}
	return result
}

type TplBagitEntryTest struct {
	Location string
	Date     time.Time
	Status   string
}
type TplBagitEntry struct {
	Name         string
	Size         string
	Ingested     string
	TestsMessage string
	Price        string
	Quality      string
}

func (tbe TplBagitEntry) Cols() []string {
	return []string{"Name", "Size", "ingested", "testsmessage", "quality", "price"}
}
func (tbe TplBagitEntry) FieldSize(col string, maxWidth int) (w int, h int) {
	data := tbe.Data(col, maxWidth)
	h = len(data)
	if h == 0 {
		return 0, 0
	}
	for l := 0; l < h; l++ {
		_w := len(data[l])
		if _w > w {
			w = _w
		}
	}
	return
}

func (tbe TplBagitEntry) Data(col string, maxWidth int) []string {
	data := []string{}
	switch col {
	case "Name":
		data = splitLine(tbe.Name, maxWidth)
	case "Size":
		data = splitLine(tbe.Size, maxWidth)
	case "ingested":
		data = splitLine(tbe.Ingested, maxWidth)
	case "testsmessage":
		data = splitLine(tbe.TestsMessage, maxWidth)
	case "price":
		data = []string{tbe.Price}
	case "quality":
		data = []string{tbe.Quality}
	}
	return data
}
func (tbe TplBagitEntry) Title(col string) string {
	result := ""
	switch col {
	case "Name":
		result = "Name"
	case "Size":
		result = "Grösse"
	case "ingested":
		result = "Vereinnahmung"
	case "price":
		result = "Kosten"
	case "quality":
		result = "Qualität"
	}
	return result
}

type TplBagitContent struct {
	OrigPath            string
	Mimetype, Dimension string
	Filesize            string
}

func (tbc TplBagitContent) Cols() []string {
	return []string{"origpath", "mimetype", "dimension", "filesize"}
}
func (tbc TplBagitContent) FieldSize(col string, maxWidth int) (w int, h int) {
	data := tbc.Data(col, maxWidth)
	h = len(data)
	if h == 0 {
		return 0, 0
	}
	for l := 0; l < h; l++ {
		_w := len(data[l])
		if _w > w {
			w = _w
		}
	}
	return
}

func (tbc TplBagitContent) Data(col string, maxWidth int) []string {
	data := []string{}
	switch col {
	case "origpath":
		data = splitLine(tbc.OrigPath, maxWidth)
	case "mimetype":
		data = splitLine(tbc.Mimetype, maxWidth)
	case "dimension":
		data = splitLine(tbc.Dimension, maxWidth)
	case "filesize":
		data = splitLine(tbc.Filesize, maxWidth)
	}
	return data
}
func (tbc TplBagitContent) Title(col string) string {
	result := ""
	switch col {
	case "origpath":
		result = "Original path"
	case "mimetype":
		result = "Mimetype"
	case "dimension":
		result = "Dimension"
	case "filesize":
		result = "Filesize"
	}
	return result
}

type TplBagitTest struct {
	End, Status, Test, Message string
}

func (tbt TplBagitTest) Cols() []string {
	return []string{"test", "end", "status", "message"}
}
func (tbt TplBagitTest) FieldSize(col string, maxWidth int) (w int, h int) {
	data := tbt.Data(col, maxWidth)
	h = len(data)
	if h == 0 {
		return 0, 0
	}
	for l := 0; l < h; l++ {
		_w := len(data[l])
		if _w > w {
			w = _w
		}
	}
	return
}

func (tbt TplBagitTest) Data(col string, maxWidth int) []string {
	data := []string{}
	switch col {
	case "end":
		data = splitLine(tbt.End, maxWidth)
	case "status":
		data = splitLine(tbt.Status, maxWidth)
	case "test":
		data = splitLine(tbt.Test, maxWidth)
	case "message":
		data = splitLine(tbt.Message, maxWidth)
	}
	return data
}
func (tbt TplBagitTest) Title(col string) string {
	result := ""
	switch col {
	case "end":
		result = "Time"
	case "status":
		result = "Status"
	case "test":
		result = "Test"
	case "message":
		result = "Message"
	}
	return result
}

func chunks(s string, chunkSize int) []string {
	if chunkSize >= len(s) {
		return []string{s}
	}
	var chunks []string
	chunk := make([]rune, chunkSize)
	len := 0
	for _, r := range s {
		chunk[len] = r
		len++
		if len == chunkSize {
			chunks = append(chunks, string(chunk))
			len = 0
		}
	}
	if len > 0 {
		chunks = append(chunks, string(chunk[:len]))
	}
	return chunks
}

func linebreak(s string, chunkSize int) []string {
	if chunkSize >= len(s) {
		return []string{s}
	}
	var lines []string
	var line = ""
	words := strings.Split(s, " ")
	for key, word := range words {
		if key == 0 || len(line)+len(word) <= chunkSize {
			line += word + " "
		} else {
			lines = append(lines, strings.TrimSpace(line))
			line = word + " "
		}
	}
	lines = append(lines, strings.TrimSpace(line))
	return lines
}

func recurseCopyTemplates(base, sub, target string) error {
	path := filepath.ToSlash(filepath.Join(base, sub))
	fs, err := templates.ReadDir(path)
	if err != nil {
		return emperror.Wrapf(err, "no reportbase compiled into executable")
	}
	for _, fe := range fs {
		subPath := filepath.ToSlash(filepath.Join(sub, fe.Name()))
		if fe.IsDir() {
			dir := filepath.Join(target, fe.Name())
			if _, err := os.Stat(dir); err != nil {
				if err := os.Mkdir(dir, 0755); err != nil {
					return emperror.Wrapf(err, "cannot create folder %s", filepath.Join(target, fe.Name()))
				}
			}

			if err := recurseCopyTemplates(base, subPath, target); err != nil {
				return emperror.Wrapf(err, "cannot copy template to %s", subPath)
			}
		} else {
			in, err := templates.Open(filepath.ToSlash(filepath.Join(base, subPath)))
			if err != nil {
				return emperror.Wrapf(err, "cannot open %s", subPath)
			}
			out, err := os.OpenFile(filepath.Join(target, sub, fe.Name()), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				in.Close()
				return emperror.Wrapf(err, "cannot create file %s", filepath.Join(target, sub, fe.Name()))
			}
			if _, err := io.Copy(out, in); err != nil {
				in.Close()
				out.Close()
				return emperror.Wrapf(err, "cannot copy data %s/%s -> %s", path, subPath, filepath.Join(target, sub, fe.Name()))
			}
			in.Close()
			out.Close()
		}
	}
	return nil
}

func createSphinx(project, copyright, author, release, path string) error {
	if _, err := os.Stat(path); err != nil {
		if err := os.Mkdir(path, 0755); err != nil {
			return emperror.Wrapf(err, "cannot create folder %s", path)
		}
	}
	if err := recurseCopyTemplates("templates/reportbase", "", path); err != nil {
		return emperror.Wrapf(err, "cannot copy shpinx template to %s", path)
	}
	confPyStr, err := templates.ReadFile("templates/conf.py.tpl")
	if err != nil {
		return emperror.Wrapf(err, "cannot open template conf.py.tpl")
	}
	conf, err := template.New("index").Parse(string(confPyStr))
	if err != nil {
		return emperror.Wrapf(err, "cannot parse conf.py.tpl")
	}

	out, err := os.OpenFile(filepath.Join(path, "conf.py"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return emperror.Wrapf(err, "cannot open %s", filepath.Join(path, "conf.py"))
	}
	defer out.Close()
	if err := conf.Execute(out, struct{ Project, Copyright, Author, Release string }{Project: project, Copyright: copyright, Author: author, Release: release}); err != nil {
		return emperror.Wrapf(err, "cannot execute conf.py.tpl - %s", confPyStr)
	}
	return nil
}

var templateFuncMap = template.FuncMap{
	"isnil": func(val interface{}) bool { return val == nil },
	"now":   time.Now,
	"replace": func(input, from, to string) string {
		return strings.Replace(input, from, to, -1)
	},
	//"strlen": func(str string) int { return len(str) },
	"repeat": func(str string, num int) (ret string) {
		for i := 0; i < num; i++ {
			ret += str
		}
		return
	},
	"multiline": func(str string, len int) []string { return chunks(str, len) },
	"linebreak": func(str string, len int) []string { return linebreak(str, len) },
	"blocks":    func(str, filler string, len int) string { chs := chunks(str, len); return strings.Join(chs, filler) },
	"format_duration": func(str string) string {
		flt, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return str
		}
		secs := int(math.Floor(flt))
		s := secs % 60
		secs -= s
		secs /= 60
		m := secs % 60
		secs -= m
		secs /= 60
		h := secs
		sf := float64(s) + flt - math.Floor(flt)
		return fmt.Sprintf("%02d:%02d:%02.3f", h, m, sf)
	},
	"quote":        func(str interface{}) string { return strings.Trim(strconv.Quote(fmt.Sprintf("%v", str)), `""`) },
	"string2array": func(str string) []string { return strings.Split(str, "\n") },
}

func (i *Ingest) ReportBagit(bagit *IngestBagit, t *template.Template, reportWriter io.Writer) error {
	p := message.NewPrinter(language.German)
	contents := []RSTTableRow{}
	SHA512 := make(map[string]string)
	files := make([]string, 0)
	if err := bagit.ContentLoadAll(func(content *IngestBagitContent) error {
		fname := fmt.Sprintf("file_%v.rst", content.contentId)
		files = append(files, fname)

		fileTplStr, err := templates.ReadFile("templates/file.rst.tpl")
		if err != nil {
			return emperror.Wrapf(err, "cannot open template file.rst.tpl")
		}
		file, err := template.New(fname).Funcs(templateFuncMap).Parse(string(fileTplStr))
		if err != nil {
			return emperror.Wrapf(err, "cannot parse file.rst.tpl")
		}

		ifp, err := os.OpenFile(filepath.Join(i.reportDir, bagit.Name, fname), os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return emperror.Wrapf(err, "cannot open file %s", filepath.Join(i.reportDir, fname))
		}
		var indexer Indexer
		// indexer.FFProbe.Format.Tags
		if err := json.Unmarshal([]byte(content.Indexer), &indexer); err != nil {
			return emperror.Wrapf(err, "cannot unmarshal indexer data - %s", content.Indexer)
		}
		if indexer.Identify != nil {
			if imgInt, ok := indexer.Identify["image"]; ok {
				img, ok := imgInt.(map[string]interface{})
				if ok {
					for _, key := range []string{"channelStatistics", "chromaticity", "imageStatistics", "name", "properties"} {
						delete(img, key)
					}
				}
			}
		}
		/*
			var tikaRows []RSTTableRow
			if indexer.Tika != nil {
				for key, val := range indexer.Tika[0] {
					tikaRows = append(tikaRows, TplBagitKeyVal{
						Key:   key,
						Value: fmt.Sprintf("%v", val),
					})
				}
			}
		*/

		//indexer.NSRL
		if len(indexer.Tika) == 0 {
			indexer.Tika = nil
		}
		if err := file.Execute(ifp, struct {
			Checksums map[string]string
			Name      string
			Indexer   Indexer
		}{
			Name:      strings.TrimLeft(content.DiskPath, "/"),
			Checksums: content.Checksums,
			Indexer:   indexer,
		}); err != nil {
			ifp.Close()
			return emperror.Wrapf(err, "cannot execute file template")
		}
		ifp.Close()

		ct := TplBagitContent{
			OrigPath: content.DiskPath,
			Mimetype: content.Mimetype,
			Filesize: p.Sprintf("%d", content.Filesize),
		}
		if content.Width > 0 || content.Height > 0 {
			ct.Dimension = fmt.Sprintf("%vx%vpixel", content.Width, content.Height)
		}
		if content.Duration > 0 {
			if len(ct.Dimension) > 0 {
				ct.Dimension += ", "
			}
			ct.Dimension += fmt.Sprintf("%vsec", content.Duration)
		}
		SHA512[strings.TrimLeft(content.DiskPath, "/")] = content.Checksums["sha512"]
		contents = append(contents, ct)
		return nil
	}); err != nil {
		return emperror.Wrapf(err, "cannot load bagits content for bagit %s", bagit.Name)
	}

	var locTests = make(map[string][]RSTTableRow)
	var locTransfer = make(map[string]*Transfer)
	for _, loc := range bagit.ingest.locations {
		var tests []RSTTableRow
		if err := bagit.TestLoadAll(loc, func(test *IngestBagitTestLocation) error {
			tbt := TplBagitTest{
				End:     test.end.Format("2006-01-02 15:04:05"),
				Status:  test.status,
				Test:    test.test.name,
				Message: test.message,
			}
			tests = append(tests, tbt)
			return nil
		}); err != nil {
			return emperror.Wrapf(err, "cannot load tests for bagit %s", bagit.Name)
		}
		locTests[loc.name] = tests
		transfer, err := loc.LoadTransfer(bagit)
		if err != nil {
			return emperror.Wrapf(err, "cannot load transfer of %s to %s", bagit.Name, loc.name)
		}
		locTransfer[loc.name] = transfer
	}

	var test = make(map[string]RSTTable)
	for key, val := range locTests {
		test[key] = RSTTable{Data: val}
	}
	var transfer = make(map[string]map[string]string)
	for key, val := range locTransfer {
		var tmap = make(map[string]string)
		tmap["Start"] = val.start.Format("2006-01-02 15:04:05")
		tmap["End"] = val.end.Format("2006-01-02 15:04:05")
		tmap["Status"] = val.status
		tmap["Message"] = val.message
		transfer[key] = tmap
	}
	if err := t.Execute(reportWriter, struct {
		Contents RSTTable
		Bagit    IngestBagit
		Tests    map[string]RSTTable
		Transfer map[string]map[string]string
		SHA512   map[string]string
		Files    []string
	}{Contents: RSTTable{Data: contents},
		Bagit:    *bagit,
		Tests:    test,
		Transfer: transfer,
		SHA512:   SHA512,
		Files:    files}); err != nil {
		return emperror.Wrapf(err, "cannot execute bagits template")
	}

	cntTplStr, err := templates.ReadFile("templates/bagit_contents.rst.tpl")
	if err != nil {
		return emperror.Wrapf(err, "cannot open template bagit_contents.rst.tpl")
	}
	cnt, err := template.New("cnt.rst").Funcs(templateFuncMap).Parse(string(cntTplStr))
	if err != nil {
		return emperror.Wrapf(err, "cannot parse bagit_contents.rst.tpl")
	}

	cfp, err := os.OpenFile(filepath.Join(i.reportDir, bagit.Name, "contents.rst"), os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return emperror.Wrapf(err, "cannot open file %s", filepath.Join(i.reportDir, bagit.Name, "contents.rst"))
	}

	if err := cnt.Execute(cfp, struct {
		Contents RSTTable
		Bagit    IngestBagit
		Tests    map[string]RSTTable
		Transfer map[string]map[string]string
		SHA512   map[string]string
		Files    []string
	}{Contents: RSTTable{Data: contents},
		Bagit:    *bagit,
		Tests:    test,
		Transfer: transfer,
		SHA512:   SHA512,
		Files:    files}); err != nil {
		cfp.Close()
		return emperror.Wrapf(err, "cannot execute bagits template")
	}
	cfp.Close()

	return nil
}

func (i *Ingest) ReportBagits(reportBagits, reportSHA512 *template.Template, reportWriter, checksumWriter io.Writer) error {
	p := message.NewPrinter(language.German)
	daTest, ok := i.tests["checksum"]
	if !ok {
		return fmt.Errorf("cannot find test checksum")
	}

	sha512s := make(map[string]string)
	bagits := []RSTTableRow{}
	if err := i.BagitLoadAll(func(bagit *IngestBagit) error {
		var quality, price float64
		b := TplBagitEntry{
			Name:     bagit.Name,
			Size:     p.Sprintf("%d", bagit.Size),
			Ingested: bagit.Creationdate.Format("2006-01-02"),
		}
		sha512s[bagit.Name] = bagit.SHA512
		var testPassed, testFailed int64
		for _, loc := range i.locations {
			t, err := i.IngestBagitTestLocationNew(bagit, loc, daTest)
			if err != nil {
				return emperror.Wrapf(err, "cannot create test for %s at %s", bagit.Name, loc.name)
			}
			if err := t.Last(); err != nil {
				i.logger.Errorf("cannot check %s at %s: %v", bagit.Name, loc.name, err)
				break
				//return emperror.Wrapf(err, "cannot check %s at %s", bagit.Name, loc.Name)
			}
			price += loc.costs * float64(bagit.Size) / 1000000
			switch t.status {
			case "passed":
				testPassed++
				quality += loc.quality
			case "failed":
				testFailed++
			default:
				return fmt.Errorf("invalid test status %s for test #%v", t.status, t.id)
			}
		}
		b.Price = fmt.Sprintf("%.2f", price)
		b.Quality = fmt.Sprintf("%.0f", quality)
		b.TestsMessage = fmt.Sprintf("%v/%v tests passed", testPassed, testFailed+testPassed)
		bagits = append(bagits, b)
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error iterating bagits")
	}
	table := RSTTable{Data: bagits}
	if err := reportBagits.Execute(reportWriter, struct {
		BagitTable RSTTable
		Bagits     []RSTTableRow
	}{BagitTable: table, Bagits: bagits}); err != nil {
		return emperror.Wrapf(err, "cannot execute bagits template")
	}
	if err := reportSHA512.Execute(checksumWriter, struct {
		ChecksumName string
		Checksums    map[string]string
	}{ChecksumName: "SHA512", Checksums: sha512s}); err != nil {
		return emperror.Wrapf(err, "cannot execute checksum template")
	}
	return nil
}

func (i *Ingest) ReportIndex(t *template.Template, wr io.Writer) error {
	daTest, ok := i.tests["checksum"]
	if !ok {
		return fmt.Errorf("cannot find test checksum")
	}

	var testPassed, testFailed int64
	if err := i.BagitLoadAll(func(bagit *IngestBagit) error {
		for _, loc := range i.locations {
			t, err := i.IngestBagitTestLocationNew(bagit, loc, daTest)
			if err != nil {
				return emperror.Wrapf(err, "cannot create test for %s at %s", bagit.Name, loc.name)
			}
			if err := t.Last(); err != nil {
				i.logger.Errorf("cannot check %s at %s: %v", bagit.Name, loc.name, err)
				break
				//return emperror.Wrapf(err, "cannot check %s at %s", bagit.Name, loc.Name)
			}
			switch t.status {
			case "passed":
				testPassed++
			case "failed":
				testFailed++
			default:
				return fmt.Errorf("invalid test status %s for test #%v", t.status, t.id)
			}
		}
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error iterating bagits")
	}
	if err := t.Execute(wr, struct {
		TestsFailed int64
	}{TestsFailed: testFailed}); err != nil {
		return emperror.Wrapf(err, "cannot execute index template")
	}
	return nil
}

type keyiv struct{ Key, IV string }

func (i *Ingest) ReportKeys(t *template.Template, wr io.Writer) error {
	var crypts = make(map[string]keyiv)
	if err := i.BagitLoadAll(func(bagit *IngestBagit) error {
		var ki keyiv
		ki.IV = fmt.Sprintf("%x", bagit.GetIV())
		ki.Key = fmt.Sprintf("%x", bagit.GetKey())
		crypts[bagit.Name] = ki
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error iterating bagits")
	}
	if err := t.Execute(wr, struct {
		KI map[string]keyiv
	}{KI: crypts}); err != nil {
		return emperror.Wrapf(err, "cannot execute keys template")
	}
	return nil
}

func (i *Ingest) Report() error {
	if err := createSphinx("The Archive", "info-age GmbH Basel", "Jürgen Enge", "0.1.1", i.reportDir+"/main"); err != nil {
		return emperror.Wrapf(err, "cannot create sphinx folder %s/main", i.reportDir)
	}

	//
	// keys
	//
	keysTplStr, err := templates.ReadFile("templates/keys.txt.tpl")
	if err != nil {
		return emperror.Wrapf(err, "cannot open template keys.txt.tpl")
	}
	keys, err := template.New("keys").Funcs(templateFuncMap).Parse(string(keysTplStr))
	if err != nil {
		return emperror.Wrapf(err, "cannot parse index.rst.tpl")
	}

	kfp, err := os.OpenFile(filepath.Join(i.keyDir, "keys.txt"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return emperror.Wrapf(err, "cannot open file %s", filepath.Join(i.keyDir, "keys.txt"))
	}

	if err := i.ReportKeys(keys, kfp); err != nil {
		kfp.Close()
		return emperror.Wrapf(err, "cannot execute template %s", "templates/keys.txt.tpl")
	}
	kfp.Close()
	//
	// index
	//
	indexTplStr, err := templates.ReadFile("templates/index.rst.tpl")
	if err != nil {
		return emperror.Wrapf(err, "cannot open template index.rst.tpl")
	}
	index, err := template.New("index").Funcs(templateFuncMap).Parse(string(indexTplStr))
	if err != nil {
		return emperror.Wrapf(err, "cannot parse index.rst.tpl")
	}

	ifp, err := os.OpenFile(filepath.Join(i.reportDir, "main", "index.rst"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return emperror.Wrapf(err, "cannot open fiel %s", filepath.Join(i.reportDir, "index.rst"))
	}

	if err := i.ReportIndex(index, ifp); err != nil {
		ifp.Close()
		return emperror.Wrapf(err, "cannot execute template %s", "templates/index.rst.tpl")
	}
	ifp.Close()

	//
	// bagits
	//
	bagitsTplStr, err := templates.ReadFile("templates/bagits.rst.tpl")
	if err != nil {
		return emperror.Wrapf(err, "cannot open template bagits.rst.tpl")
	}
	bagits, err := template.New("bagits").Funcs(templateFuncMap).Parse(string(bagitsTplStr))
	if err != nil {
		return emperror.Wrapf(err, "cannot parse bagits.rst.tpl")
	}

	ifp, err = os.OpenFile(filepath.Join(i.reportDir, "main", "bagits.rst"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return emperror.Wrapf(err, "cannot open file %s", filepath.Join(i.reportDir, "bagits.rst"))
	}

	checksumTplStr, err := templates.ReadFile("templates/bagit_checksum.txt.tpl")
	if err != nil {
		return emperror.Wrapf(err, "cannot open template bagits.rst.tpl")
	}
	checksums, err := template.New("checksums").Funcs(templateFuncMap).Parse(string(checksumTplStr))
	if err != nil {
		return emperror.Wrapf(err, "cannot parse bagits.rst.tpl")
	}

	cfp, err := os.OpenFile(filepath.Join(i.reportDir, "", "bagit_checksum.txt"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return emperror.Wrapf(err, "cannot open file %s", filepath.Join(i.reportDir, "bagit_checksum.txt"))
	}

	if err := i.ReportBagits(bagits, checksums, ifp, cfp); err != nil {
		cfp.Close()
		return emperror.Wrapf(err, "cannot execute template %s", "templates/bagit_checksum.txt.tpl")
	}
	cfp.Close()

	if err := i.BagitLoadAll(func(bagit *IngestBagit) error {
		if err := createSphinx(bagit.Name, "info-age GmbH Basel", "Jürgen Enge", "0.1.1", i.reportDir+"/"+bagit.Name); err != nil {
			return emperror.Wrapf(err, "cannot create sphinx folder %s/main", i.reportDir)
		}

		//
		// bagit
		//
		bagitTplStr, err := templates.ReadFile("templates/bagit.rst.tpl")
		if err != nil {
			return emperror.Wrapf(err, "cannot open template bagit.rst.tpl")
		}
		bagitTpl, err := template.New("bagit_" + bagit.Name).Funcs(templateFuncMap).Parse(string(bagitTplStr))
		if err != nil {
			return emperror.Wrapf(err, "cannot parse bagits.rst.tpl")
		}

		ifp, err = os.OpenFile(filepath.Join(i.reportDir, bagit.Name, "index.rst"), os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return emperror.Wrapf(err, "cannot open file %s", filepath.Join(i.reportDir, "index.rst"))
		}

		if err := i.ReportBagit(bagit, bagitTpl, ifp); err != nil {
			ifp.Close()
			return emperror.Wrapf(err, "cannot execute template %s", "templates/bagit.rst.tpl")
		}
		ifp.Close()

		return nil
	}); err != nil {
		return emperror.Wrapf(err, "cannot load bagits")
	}

	return nil
}
