package recovery

// Recoverer offers the ability to recover
// from security violation or boot failure
type Recoverer interface {
	Recover()
}
