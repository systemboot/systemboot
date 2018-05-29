package recovery

// Powerer offers the ability to recover
// from security violation or boot failure
type Powerer interface {
	PowerCycle()
}
