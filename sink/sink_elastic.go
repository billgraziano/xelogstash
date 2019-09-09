package sink

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/billgraziano/xelogstash/eshelper"
	"github.com/pkg/errors"
)

// ElasticSink writes to an ElasticSearch instance
type ElasticSink struct {
	Addresses         []string
	Username          string
	Password          string
	DefaultIndex      string
	EventIndexMap     map[string]string
	AutoCreateIndexes bool
	ProxyServer       string
	client            *elasticsearch.Client
	buf               *bytes.Buffer
}

// NewElasticSink returns a new ElasticSink
func NewElasticSink(addresses []string, proxy, username, password string) (*ElasticSink, error) {
	es := ElasticSink{
		Addresses:   addresses,
		Username:    username,
		Password:    password,
		ProxyServer: proxy,
	}
	client, err := eshelper.NewClient(addresses, proxy, username, password)
	if err != nil {
		return nil, errors.Wrap(err, "eshelper.newclient")
	}
	es.client = client
	es.buf = new(bytes.Buffer)
	return &es, nil
}

// Name returns the name of the sink
func (es *ElasticSink) Name() string {
	return fmt.Sprintf("elastic: %s", strings.Join(es.Addresses, ", "))
}

// Open tests the Elastic Client (id is ignored)
func (es *ElasticSink) Open(ignored string) error {
	_, err := es.client.Info()
	if err != nil {
		return errors.Wrap(err, "es.info")
	}

	if es.AutoCreateIndexes {
		esIndexes := make([]string, 0)
		if es.DefaultIndex != "" {
			esIndexes = append(esIndexes, es.DefaultIndex)
		}
		// App log indexes?
		// if es. != "" {
		// 	esIndexes = append(esIndexes, es.AppLogIndex)
		// }
		for _, ix := range es.EventIndexMap {
			esIndexes = append(esIndexes, ix)
		}

		err = eshelper.CreateIndexes(es.client, esIndexes)
		if err != nil {
			return errors.Wrap(err, "eshelper.createindexes")
		}
	}
	return nil
}

// Write the event to the buffer
func (es *ElasticSink) Write(name, event string) (int, error) {
	var esIndex string
	var ok bool
	esIndex, ok = es.EventIndexMap[name]
	if !ok {
		esIndex = es.DefaultIndex
	}
	meta := []byte(fmt.Sprintf(`{ "index" : { "_index" : "%s" } }%s`, esIndex, "\n"))
	espayload := []byte(event + "\n")
	es.buf.Grow(len(meta) + len(espayload))
	es.buf.Write(meta)
	es.buf.Write(espayload)
	return 0, nil
}

// Flush the buffer to ElasticSearch
func (es *ElasticSink) Flush() error {
	err := eshelper.WriteElasticBuffer(es.client, es.buf)
	if err != nil {
		return errors.Wrap(err, "eshelper.writeelasticbuffer")
	}
	es.buf.Reset()
	return nil
}

// Close the ElasticSink
func (es *ElasticSink) Close() error {
	return es.Flush()
}

// Clean is a noop to satisfy the interface
func (es *ElasticSink) Clean() error {
	return nil
}
