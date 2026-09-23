package lamport

// Time is the value of a Clock.
type Time uint64

// Clock is a Lamport logical clock
type Clock interface {
	// Time is used to return the current value of the lamport clock.
	// It can fail, as an implementation may have to read the value from a
	// storage shared with other processes.
	Time() (Time, error)
	// Increment is used to return the value of the lamport clock and increment it afterwards
	Increment() (Time, error)
	// Witness is called to update our local clock if necessary after
	// witnessing a clock value received from another process
	Witness(time Time) error
}
