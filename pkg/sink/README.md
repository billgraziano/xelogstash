# Sink Notes
* These are mostly from before development
* Each sink should be a package.  But only `sampler` is :(

Names
-----
* Package is `sink`
* Interface is `sink.Sinker`

Logstash Notes
--------------
* Issues 
  - Logstash servers restart
  - DNS returns multiple servers
  - DNS updates which servers are active
* Want a pool of senders?  Or maybe a pool of connections?
* Want to round robin a pool of hosts?
* Connection pool - https://stackoverflow.com/questions/10308388/tcp-connection-pool 
* https://github.com/eternnoir/gncp
* https://github.com/fatih/pool 
