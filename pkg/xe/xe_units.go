package xe

// SetExtraUnits adds duration_sec, cpu_time_sec, writes_mb,
// logical_reads_mb, and physical_reads_mb
func (e *Event) SetExtraUnits() {
	dur, ok := e.GetIntFromString("duration")
	if ok {
		if dur >= 1000000 {
			e.Set("duration_sec", dur/1000000) // microseconds to seconds
		}
	}
	cpu, ok := e.GetIntFromString("cpu_time")
	if ok {
		if cpu >= 1000000 {
			e.Set("cpu_time_sec", cpu/1000000) // microseconds to seconds
		}
	}
	lr, ok := e.GetIntFromString("logical_reads")
	if ok {
		if lr >= 128 {
			e.Set("logical_reads_mb", lr/128) // pages to bytes
		}
	}
	pr, ok := e.GetIntFromString("physical_reads")
	if ok {
		if pr >= 128 {
			e.Set("physical_reads_mb", pr/128) // pages to bytes
		}
	}
	wr, ok := e.GetIntFromString("writes")
	if ok {
		if wr >= 128 {
			e.Set("writes_mb", wr/128) // pages to bytes
		}
	}
}
