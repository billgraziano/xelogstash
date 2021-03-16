package ls2

// Close a logstash connection
func (ls *Logstash) Close() error {
	ls.Lock()
	defer ls.Unlock()
	return ls.close()
}

func (ls *Logstash) close() error {
	var err error
	if ls.Connection != nil {
		err = ls.Connection.Close()
		ls.Connection = nil
		return err
	}
	return nil
}
