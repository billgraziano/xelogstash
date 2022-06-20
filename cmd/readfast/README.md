# READFAST

Readfast tries to read extended events as fast as possible.

It reads all events for one session and reports the number of events, duration, and events per second.

It can parse enough of the event to determine the time of the event.  It can also marshal the JSON.

It does not write the event anywhere.

## Performance

* On a small desktop using SQL Server Developer Edition
    * Read 964K events in 9 files at 28,762 per second
    * Parsing the XML reduced that to 16,472 per second
    * Parsing and processing with BlobLang reduced that to 15,248 per second
    * Parsing, BlobLang, and marshaling to JSON reduced that to 11,286 per second
    * EXE is 8 MB.  Adding BlobLang increased the EXE to 16 MB.  
    * RAM peaked at 21MB even with marshalling the JSON.  Most of this seemed to be the executable.
* Against a SQL Server across a LAN
    * Read 453K events in 10 files is 23K per second
    * Read, parsing, BlobLang, and marshaling is 4,650 per second
* Against a SQL Server across a 40ms ping WAN
    * Read one million events at 20K per second
    * Read and parse is 10,508 per second
    * Read, parse, BlobLang, and marshaling is 7,030 per second
