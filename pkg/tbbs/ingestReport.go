package tbbs

import (
	"embed"
	"fmt"
	"github.com/goph/emperror"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Embed the entire directory.
//go:embed templates/* templates/reportbase/*
var templates embed.FS

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
}

func splitLine(str string, maxWidth int) []string {
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

func (tbe TplBagitEntry) Cols() []string {
	return []string{"name", "size", "ingested", "testsmessage"}
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
	case "name":
		data = splitLine(tbe.Name, maxWidth)
	case "size":
		data = splitLine(tbe.Size, maxWidth)
	case "ingested":
		data = splitLine(tbe.Ingested, maxWidth)
	case "testsmessage":
		data = splitLine(tbe.TestsMessage, maxWidth)
	}
	return data
}
func (tbe TplBagitEntry) Title(col string) string {
	result := ""
	switch col {
	case "name":
		result = "Name"
	case "size":
		result = "Grösse"
	case "ingested":
		result = "Vereinnahmung"
		//	case "testsmessage":
	}
	return result
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
	"now": time.Now,
	"replace": func(input, from, to string) string {
		return strings.Replace(input, from, to, -1)
	},
}

func (i *Ingest) ReportTemplate(t *template.Template, wr io.Writer) error {
	p := message.NewPrinter(language.German)
	daTest, ok := i.tests["checksum"]
	if !ok {
		return fmt.Errorf("cannot find test checksum")
	}

	bagits := []RSTTableRow{}
	if err := i.BagitLoadAll(func(bagit *IngestBagit) error {
		b := TplBagitEntry{
			Name:     bagit.name,
			Size:     p.Sprintf("%d", bagit.size),
			Ingested: bagit.creationdate.Format("2006-01-02"),
		}
		var testPassed, testFailed int64
		for _, loc := range i.locations {
			t, err := i.IngestBagitTestLocationNew(bagit, loc, daTest)
			if err != nil {
				return emperror.Wrapf(err, "cannot create test for %s at %s", bagit.name, loc.name)
			}
			if err := t.Last(); err != nil {
				i.logger.Errorf("cannot check %s at %s: %v", bagit.name, loc.name, err)
				break
				//return emperror.Wrapf(err, "cannot check %s at %s", bagit.name, loc.name)
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
		b.TestsMessage = fmt.Sprintf("%v/%v tests passed", testPassed, testFailed+testPassed)
		bagits = append(bagits, b)
		return nil
	}); err != nil {
		return emperror.Wrap(err, "error iterating bagits")
	}
	table := RSTTable{Data: bagits}
	if err := t.Execute(wr, struct {
		BagitTable RSTTable
		Bagits     []RSTTableRow
	}{BagitTable: table, Bagits: bagits}); err != nil {
		return emperror.Wrapf(err, "cannot execute index template")
	}
	return nil
}

func (i *Ingest) Report() error {
	indexTplStr, err := templates.ReadFile("templates/index.rst.tpl")
	if err != nil {
		return emperror.Wrapf(err, "cannot open template index.rst.tpl")
	}
	index, err := template.New("index").Funcs(templateFuncMap).Parse(string(indexTplStr))
	if err != nil {
		return emperror.Wrapf(err, "cannot parse index.rst.tpl")
	}

	if err := createSphinx("The Archive", "info-age GmbH Basel", "Jürgen Enge", "0.1.1", i.reportDir+"/main"); err != nil {
		return emperror.Wrapf(err, "cannot create sphinx folder %s/main", i.reportDir)
	}

	ifp, err := os.OpenFile(filepath.Join(i.reportDir, "main", "index.rst"), os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return emperror.Wrapf(err, "cannot open fiel %s", filepath.Join(i.reportDir, "index.rst"))
	}
	defer ifp.Close()

	if err := i.ReportTemplate(index, ifp); err != nil {
		return emperror.Wrapf(err, "cannot execute template %s", "templates/index.rst.tpl")
	}

	return nil
}
