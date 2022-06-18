package xe

// SetExtraUnits adds duration_sec, cpu_time_sec, writes_mb,
// logical_reads_mb, and physical_reads_mb
func (e *Event) SetExtraUnits() {
	dur, ok := e.GetIntFromString("duration")
	if ok {
		e.Set("duration_sec", dur/1000000) // microseconds to seconds
	}
	cpu, ok := e.GetIntFromString("cpu_time")
	if ok {
		e.Set("cpu_time_sec", cpu/1000000) // microseconds to seconds
	}
	lr, ok := e.GetIntFromString("logical_reads")
	if ok {
		e.Set("logical_reads_mb", lr*8192/(1024*1024)) // pages to bytes
	}
	pr, ok := e.GetIntFromString("physical_reads")
	if ok {
		e.Set("physical_reads_mb", pr*8192/(1024*1024)) // pages to bytes
	}
	wr, ok := e.GetIntFromString("writes")
	if ok {
		e.Set("writes_mb", wr*8192/(1024*1024)) // pages to bytes
	}
}
