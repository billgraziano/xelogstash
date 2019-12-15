package sink

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/billgraziano/xelogstash/eshelper"
	"github.com/elastic/go-elasticsearch/v7"
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
	//buf               *bytes.Buffer
	mu sync.RWMutex
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
	//es.buf = new(bytes.Buffer)
	return &es, nil
}

// Name returns the name of the sink
func (es *ElasticSink) Name() string {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return fmt.Sprintf("elastic: %s", strings.Join(es.Addresses, ", "))
}

// Open tests the Elastic Client (id is ignored)
func (es *ElasticSink) Open(ignored string) error {
	es.mu.Lock()
	defer es.mu.Unlock()

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
// func (es *ElasticSink) Write(name, event string) (int, error) {
// 	var esIndex string
// 	var ok bool
// 	var n int
// 	var err error

// 	es.mu.Lock()
// 	defer es.mu.Unlock()
// 	esIndex, ok = es.EventIndexMap[name]
// 	if !ok {
// 		esIndex = es.DefaultIndex
// 	}
// 	meta := []byte(fmt.Sprintf(`{ "index" : { "_index" : "%s" } }%s`, esIndex, "\n"))
// 	espayload := []byte(event + "\n")
// 	es.buf.Grow(len(meta) + len(espayload))
// 	n, err = es.buf.Write(meta)
// 	if err != nil {
// 		return n, errors.Wrap(err, "es.buf.write-meta")
// 	}
// 	es.buf.Write(espayload)
// 	if err != nil {
// 		return n, errors.Wrap(err, "es.buf.write-payload")
// 	}
// 	return 0, nil
// }

// Write the event to Elastic
func (es *ElasticSink) Write(name, event string) (int, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var esIndex string
	var ok bool
	var n int
	var err error

	esIndex, ok = es.EventIndexMap[name]
	if !ok {
		esIndex = es.DefaultIndex
	}

	meta := []byte(fmt.Sprintf(`{ "index" : { "_index" : "%s" } }%s`, esIndex, "\n"))
	espayload := []byte(event + "\n")
	var b bytes.Buffer
	b.Grow(len(meta) + len(espayload))
	n, err = b.Write(meta)
	if err != nil {
		return n, errors.Wrap(err, "es.buf.write-meta")
	}
	b.Write(espayload)
	if err != nil {
		return n, errors.Wrap(err, "es.buf.write-payload")
	}

	err = eshelper.WriteElasticBuffer(es.client, &b)
	if err != nil {
		return 0, errors.Wrap(err, "eshelper.writeelasticbuffer")
	}
	return 0, nil
}

// Flush the buffer to ElasticSearch
func (es *ElasticSink) Flush() error {
	return nil
	// es.mu.Lock()
	// defer es.mu.Unlock()

	// err := eshelper.WriteElasticBuffer(es.client, es.buf)
	// if err != nil {
	// 	return errors.Wrap(err, "eshelper.writeelasticbuffer")
	// }
	// es.buf.Reset()
	// return nil
}

func (es *ElasticSink) flush() error {
	// err := eshelper.WriteElasticBuffer(es.client, es.buf)
	// if err != nil {
	// 	return errors.Wrap(err, "eshelper.writeelasticbuffer")
	// }
	// es.buf.Reset()
	return nil
}

// Close the ElasticSink
func (es *ElasticSink) Close() error {
	return nil
	//return es.Flush()
}

// Clean is a noop to satisfy the interface
func (es *ElasticSink) Clean() error {
	return nil
}

// Reopen is a noop at this point
func (es *ElasticSink) Reopen() error {
	return nil
}
