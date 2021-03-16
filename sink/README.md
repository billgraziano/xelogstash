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

Files
-----

* /sink/sink.go -- interface
* /sink/file/file_sink.go - a file sink
* /sink/logstash_http/
* /sink/logstash_tcp/
* /sink/elastic_http/


API
===
* `Write([]blah) (n int, error)`
    * array of bytes, strings?
* `Close() error`    
* ? -`Ping() error`
* ? `New()`

```
type Sinker interface {
     // Connect() error 
     Write(sting -- or bytes) (n, err)
     Close() error 
     
}

fileSink.Write()
  -- connect if needed
  -- write
```

Build array of Sinker 
-- loop through and call Write on each

LocalFile
---------
* .\events\events_20180730_00_computer_instance.log
* Single threaded writer per target

Logstash Notes
--------------
* Issues 
  - Logstash servers restart
  - DNS returns multiple servers
  - DNS updates which servers are active
* Want a pool of senders?  Or maybe a pool of connections?
* Want to round robin a pool of hosts?
* Simple connection pool - https://stackoverflow.com/questions/10308388/tcp-connection-pool 
* https://github.com/eternnoir/gncp
* https://github.com/fatih/pool 
