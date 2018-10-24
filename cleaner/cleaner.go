package cleaner

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type Cleaner func() error

var cleaners []t
var inactive bool

// Register a cleaner function. When a function is registered, the Signal watcher is started in a goroutine.
func Register(f t) {
	cleaners = append(cleaners, f)
	if !inactive {
		inactive = false
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
			<-ch
                        // Prevent un-terminated ^C character in terminal
			fmt.Println()
			clean()
			os.Exit(1)
		}()
	}
}

// Clean invokes all registered cleanup functions
func clean() {
	fmt.Println("Cleaning")
	for _, f := range cleaners {
		_ = f()
	}
	cleaners = []t{}
}
