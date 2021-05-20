package tbbs

import (
	"embed"
	"fmt"
	"github.com/goph/emperror"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Embed the entire directory.
//go:embed templates
var templates embed.FS

func recurseCopyTemplates(base, sub, target string) error {
	path := filepath.ToSlash(filepath.Join(base, sub))
	fs, err := templates.ReadDir(path)
	if err != nil {
		return emperror.Wrapf(err, "no reportbase compiled into executable")
	}
	for _, fe := range fs {
		subPath := filepath.ToSlash(filepath.Join(path, fe.Name()))
		if fe.IsDir() {
			if err := os.Mkdir(filepath.Join(target, fe.Name()), 0755); err != nil {
				return emperror.Wrapf(err, "cannot create folder %s", filepath.Join(target, fe.Name()))
			}

			if err := recurseCopyTemplates(base, subPath, target); err != nil {
				return emperror.Wrapf(err, "cannot copy template to %s", subPath)
			}
		} else {
			in, err := templates.Open(subPath)
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
				return emperror.Wrapf(err, "cannot copy data %s -> %s", subPath, filepath.Join(target, sub, fe.Name()))
			}
			in.Close()
			out.Close()
		}
	}
	return nil
}

func createSphinx(project, copyright, author, release, path string) error {
	if err := recurseCopyTemplates("reportbase", "", path); err != nil {
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

func (i *Ingest) Report() error {
	daTest, ok := i.tests["checksum"]
	if !ok {
		return fmt.Errorf("cannot find test checksum")
	}

	funcMap := template.FuncMap{
		"now": time.Now,
		"replace": func(input, from, to string) string {
			i.logger.Debugf("replace %s - %s -> %s", input, from, to)
			return strings.Replace(input, from, to, -1)
		},
	}
	indexTplStr, err := templates.ReadFile("templates/index.rst.tpl")
	if err != nil {
		return emperror.Wrapf(err, "cannot open template index.rst.tpl")
	}
	index, err := template.New("index").Funcs(funcMap).Parse(string(indexTplStr))
	if err != nil {
		return emperror.Wrapf(err, "cannot parse index.rst.tpl")
	}
	bagits := []*TplBagitEntry{}
	if err := i.BagitLoadAll(func(bagit *IngestBagit) error {
		b := &TplBagitEntry{
			Name:     bagit.name,
			Size:     bagit.size,
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

	ifp, err := os.OpenFile(filepath.Join(i.reportDir, "index.rst"), os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return emperror.Wrapf(err, "cannot open fiel %s", filepath.Join(i.reportDir, "index.rst"))
	}
	defer ifp.Close()
	if err := index.Execute(ifp, bagits); err != nil {
		return emperror.Wrapf(err, "cannot execute index template")
	}

	return nil
}
