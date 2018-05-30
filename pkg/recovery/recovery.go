package recovery

// Recoverer interface for recovering with different implementations
type Recoverer interface {
	Recover() error
}
