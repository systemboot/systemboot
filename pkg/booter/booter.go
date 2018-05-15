package booter

import "log"

// Booter is an interface that defines custom boot types. Implementations can be
// like network boot, local boot, etc.
type Booter interface {
	Boot() error
	TypeName() string
}

// NullBooter is a dummy booter that does nothing. It is used when no other
// booter has been found
type NullBooter struct {
}

func (nb *NullBooter) TypeName() string {
	return "null"
}

func (nb *NullBooter) Boot() error {
	log.Printf("Null booter does nothing")
	return nil
}
