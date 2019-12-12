package main

import (
	"os"
	"time"

	"github.com/billgraziano/xelogstash/log"
	"github.com/billgraziano/xelogstash/logstash"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

// This utility was primarily written to test against logz.io
// They require @timestamp and a field named "message"
// It sends successfully but disappers without those
func main() {
	var err error
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetLevel(log.DEBUG)

	log.Info("starting...")
	var opts struct {
		Logstash string `long:"logstash" description:"Logstash host in host:port format (required)"`
		Token    string `long:"token" description:"Token for logz.io testing (optional)"`
		Name     string `long:"name" description:"field name for the token"`
		Message  string `long:"message" description:"Content of the message body (Optional)"`
	}

	var parser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	_, err = parser.Parse()
	if err != nil {
		log.Error(errors.Wrap(err, "flags.Parse"))
		os.Exit(1)
	}

	if opts.Logstash == "" {
		log.Error("logstash host is empty")
		os.Exit(1)
	}

	if opts.Name == "" {
		opts.Name = "token"
		log.Info("token name defaulting to 'token'")
	} else {
		log.Info("token name:", opts.Name)
	}

	log.Info("logstash:", opts.Logstash)
	//log.Info("token:", opts.Token)

	var ls *logstash.Logstash
	ls, err = logstash.NewHost(opts.Logstash, 180)
	if err != nil {
		log.Fatal("logstash.newhost:", err)
	}

	lr := logstash.NewRecord()

	lr["@timestamp"] = time.Now()
	if opts.Message == "" {
		lr["message"] = "message body"
	} else {
		lr["message"] = opts.Message
	}

	if opts.Token != "" {
		lr[opts.Name] = opts.Token
	}

	rs, err := lr.ToJSON()
	if err != nil {
		log.Fatal("lr.tojson:", err)
	}

	log.Info(rs)

	err = ls.Writeln(rs)
	if err != nil {
		log.Fatal("ls.writeln:", err)
	}
}
