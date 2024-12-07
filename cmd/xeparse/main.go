package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/billgraziano/mssqlh"
	"github.com/billgraziano/xelogstash/pkg/log"
	"github.com/billgraziano/xelogstash/pkg/logstash"
	"github.com/billgraziano/xelogstash/pkg/xe"
	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

var opts struct {
	//Source string `long:"source" description:"source file"`
	Server string `long:"server" description:"SQL Server for meta data"`
}

func main() {
	var err error

	var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	_, err = parser.Parse()
	if err != nil {
		log.Error(errors.Wrap(err, "flags.Parse"))
		os.Exit(1)
	}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	log.Info("path:", dir)
	dirname := ".\\samplexml"

	files, err := os.ReadDir(dirname)
	if err != nil {
		log.Fatal(err)
	}

	cxn := mssqlh.NewConnection(opts.Server, "", "", "master", "")
	info, err := xe.NewSQLInfo("mssql", cxn.String(), "", "")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fi := filepath.Join(dirname, f.Name())
		fi = filepath.Clean(fi)
		if filepath.Ext(fi) != ".xml" {
			continue
		}
		log.Info("file:", fi)
		b, err := os.ReadFile(fi) //#nosec G304 -- file doesn't come from user input
		if err != nil {
			log.Fatal(err)
		}
		x := string(b)

		event, err := xe.Parse(&info, x, false)
		if err != nil {
			log.Fatal(err)
		}

		lr := logstash.NewRecord()

		for k, v := range event {
			lr[k] = v
		}

		rs, err := lr.ToJSON()
		if err != nil {
			log.Fatal(err)
		}
		// log.Debug(rs)
		var out bytes.Buffer
		err = json.Indent(&out, []byte(rs), "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		// get output file name
		basefile := strings.TrimSuffix(fi, filepath.Ext(fi))
		newfile := basefile + ".json"
		//outfile := filepath.Join(dirname, newfile)
		err = os.WriteFile(newfile, out.Bytes(), 0600)
		if err != nil {
			log.Fatal(err)
		}
		// if _, err = file.Write([]byte(fmt.Sprintf("%s\r\n\r\n", out.String()))); err != nil {
		// 	return errors.Wrap(err, "file.write")
		// }

	}
}
