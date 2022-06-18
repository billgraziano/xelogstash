
* The New constructor will have to accept a Context


* Close()
    * cancel all the workers
    * close the TCP connection

* Open
    * Close if open...
    * Create a new context from the passed one
    * Open the TCP connection
    * Init the channel
    * Launch X workers (default to the number of cores)

* Write
    * Write the event to the channel

* Worker
    * SELECT channel, cancel
    * Write the event
    * Log any errors

