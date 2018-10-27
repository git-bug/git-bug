package interrupt

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Cleaner type refers to a function with no inputs that returns an error
type Cleaner func() error

var cleaners []Cleaner
var active = false

// RegisterCleaner is responsible for registering a cleaner function. When a function is registered, the Signal watcher is started in a goroutine.
func RegisterCleaner(f ...Cleaner) {
	for _, fn := range f {
		cleaners = append([]Cleaner{fn}, cleaners...)
		if !active {
			active = true
			go func() {
				ch := make(chan os.Signal, 1)
				signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
				<-ch
				// Prevent un-terminated ^C character in terminal
				fmt.Println()
				errl := clean()
				for _, err := range errl {
					fmt.Println(err)
				}
				os.Exit(1)
			}()
		}
	}
}

// clean invokes all registered cleanup functions, and returns a list of errors, if they exist.
func clean() (errorlist []error) {
	for _, f := range cleaners {
		err := f()
		if err != nil {
			errorlist = append(errorlist, err)
		}
	}
	cleaners = []Cleaner{}
	return
}
