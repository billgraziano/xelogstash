package eshelper

/*

"github.com/elastic/go-elasticsearch"
	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/estransport"

*/
import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/billgraziano/go-elasticsearch"
	"github.com/billgraziano/go-elasticsearch/esapi"
	"github.com/billgraziano/xelogstash/log"
	"github.com/pkg/errors"
)

// BulkResponse holds the response from Elastic
type BulkResponse struct {
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			ID     string `json:"_id"`
			Result string `json:"result"`
			Status int    `json:"status"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
				Cause  struct {
					Type   string `json:"type"`
					Reason string `json:"reason"`
				} `json:"caused_by"`
			} `json:"error"`
		} `json:"index"`
	} `json:"items"`
}

// NewClient creates a client given an http.Transport.  This is typically used for a proxy.
func NewClient(addresses []string, proxy, username, password string) (*elasticsearch.Client, error) {

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		log.Fatal(err)
	}
	t := http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: time.Second,
		DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS11,
		},
	}

	cfg := elasticsearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
		Transport: &t,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "elasticsearch.newclient")
	}

	_, err = es.Info()
	if err != nil {
		return nil, errors.Wrap(err, "es.info")
	}
	return es, nil
}

// NewDefaultClient returns a new elastic search client
func NewDefaultClient(addresses []string, username, password string) (*elasticsearch.Client, error) {
	es, err := NewClient(addresses, "", username, password)
	if err != nil {
		return nil, errors.Wrap(err, "eshelper.newclient")
	}

	return es, nil
}

// CreateIndexes creates all the needed indexes
func CreateIndexes(es *elasticsearch.Client, indexes []string) error {
	var (
		res *esapi.Response
		err error
	)

	for _, i := range indexes {
		res, err = es.Indices.Exists([]string{i})
		if err != nil {
			return errors.Wrap(err, "es.indices.exist")
		}
		if res.StatusCode == 200 {
			continue
		}
		res, err = es.Indices.Create(i)
		if err != nil {
			return errors.Wrap(err, "es.indices.create")
		}
		if res.IsError() {
			return errors.New(fmt.Sprintf("error creating elastic index [%s]: %s", i, res.String()))
		}
	}
	return nil
}

// WriteElasticBuffer sends the buffer to Elastic
func WriteElasticBuffer(esclient *elasticsearch.Client, buf *bytes.Buffer) error {
	if len(buf.Bytes()) == 0 {
		return nil
	}

	res, err := esclient.Bulk(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return errors.Wrap(err, "failure indexing batch")
	}
	defer res.Body.Close()
	if res.IsError() {
		log.Error(fmt.Sprintf("buffer bytes: %d", len(buf.Bytes())))
		log.Error(res.String())
		log.Error(string(buf.String()))
		return errors.New("res.iserror true writing to elastic")
	}
	buf.Reset()
	return nil
}
