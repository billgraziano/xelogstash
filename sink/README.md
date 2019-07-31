* Maybe the package should be `sink`.  Short and to the point.

Names
=====
* Destinations, Senders, Sinks, Targets
* Package is `sink`
* Interface is `sink.Writer`

Clients
=======
* `LogstashTCP`
* `LogstashHTTP` - Flag to enable bulk (or JSON lines)
* `ElasticHTTP` (ElasticBulk)

/sink/sink.go -- interface
/sink/file/file_sink.go - a file sink
/sink/logstash_http/
/sink/logstash_tcp/
/sink/elastic_http/

API
===
* `Write([]blah) (n int, error)`
    * array of bytes, strings?
* `Close() error`    
* ? -`Ping() error`
* ? `New()`

type Sinker interface {
     // Connect() error 
     Write(sting -- or bytes) (n, err)
     Close() error 
     
}

fileSink.Write()
  -- connect if needed
  -- write


Build array of Sinker 
-- loop through and call Write on each

LocalFile
---------
* .\events\events_20180730_00_computer_instance.log
* Single threaded writer per target

