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


API
===
* `Write([]blah) (n int, error)`
    * array of bytes, strings?
* `Ping() error`
* `New()`

