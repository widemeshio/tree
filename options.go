package tree

// Options contains task options
type Options struct {
	Logger Logger // nop unless set
}

// GetLogger returns a logger instance or default
func (opts Options) GetLogger() Logger {
	if v := opts.Logger; v != nil {
		return v
	}
	return nil
}
