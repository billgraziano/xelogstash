/sink/logstash
==============
* Accepts a logger and a context



/logstash
=========
* Accepts a logger and a context


* Get shared lock
* Try to write
* release shared lock
* If write was Ok, exit

* Get Lock
* Try to write (in case someone fixed it)
* If OK, release lock and exit
* Loop to try and fix it
    * Rebuild the connection
    * Try to write
    * If Ok, relase lock and exit
    * Delay an increasing amount

