package interrupt

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// CleanerFunc is a function to be executed when an interrupt trigger
type CleanerFunc func() error

// CancelFunc, if called, will disable the associated cleaner.
// This allow to create temporary cleaner. Be mindful though to not
// create too much of them as they are just disabled, not removed from
// memory.
type CancelFunc func()

type wrapper struct {
	f        CleanerFunc
	disabled bool
}

var mu sync.Mutex
var cleaners []*wrapper
var handlerCreated = false

// RegisterCleaner is responsible for registering a cleaner function.
// When a function is registered, the Signal watcher is started in a goroutine.
func RegisterCleaner(cleaner CleanerFunc) CancelFunc {
	mu.Lock()
	defer mu.Unlock()

	w := &wrapper{f: cleaner}
	cancel := func() { w.disabled = true }

	// prepend to later execute then in reverse order
	cleaners = append([]*wrapper{w}, cleaners...)

	if handlerCreated {
		return cancel
	}

	handlerCreated = true
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-ch
		// Prevent un-terminated ^C character in terminal
		fmt.Println()
		errl := clean()
		for _, err := range errl {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}()

	return cancel
}

// clean invokes all registered cleanup functions, and returns a list of errors, if they exist.
func clean() (errorList []error) {
	mu.Lock()
	defer mu.Unlock()

	for _, cleaner := range cleaners {
		if cleaner.disabled {
			continue
		}
		err := cleaner.f()
		if err != nil {
			errorList = append(errorList, err)
		}
	}
	cleaners = []*wrapper{}
	return
}
