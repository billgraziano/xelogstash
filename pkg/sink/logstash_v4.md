

* Need
    * Context
    * Logger


* Shared Lock
* Write
* release lock
* if OK, exit

* get exclusive lock
* defer relesae 
* do write // if someone else fixed it
* if ok, exit

* loop
    * backoff or delay (context)
    * Close the connection - log error only
    * Make new connection - log error only
    * if no error, try the write, log any errors
    * If no errors, break out of the loop


Open (Connect)
====
* Lock & defer Unlock
* close()
* open()


open()
------
* Loop
    * resolve address
    * open
    * set connection
    * back off (ctx)


Close()
=======
lock & defer

close()
-------


Write() & Maybe RecycleWrite()
==============================
* RLock()
* Write
* If OK, RUnlock(), return

* Lock()
* defer Unlock()

* Write
* If OK, return

* Loop
    * close()
    * open()
    * write()
    * backoff()

write()
-------
* set deadline
* write

