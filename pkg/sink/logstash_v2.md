

/sink/logstash
==============
* Init - make X workers
* Write
    * Send down the channel

/logstash (maybe lswriter package?)
=========

* single threaded
* Needs a logger
* Needs a context cancel
* Writer has connection in it
* accepts context
* Has retry logic forever to keep trying
    * The retry uses a SELECT with Cancel and timer/ticker to handle context cancel
* On error
    * close the connection
    * nil the connection
    * try again

* Maye retry starts at 10 seconds and doubles up to 300 seconds (5 minutes)

Maybe WRITE does this:
-----------
```
loop forever {
    write 
    if no error { break }
    log.Error 
    close connection 
    throw away connection
    sleep 60 seconds (or retry) while checking context
    if cancel, log it and exit 
    make a new connection with new IP, etc.
}
```
	