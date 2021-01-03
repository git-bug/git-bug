/*

	This Source Code Form is subject to the terms of the Mozilla Public
	License, v. 2.0. If a copy of the MPL was not distributed with this file,
	You can obtain one at http://mozilla.org/MPL/2.0/.

	Copyright (c) 2013, Armon Dadgar armon.dadgar@gmail.com
	Copyright (c) 2013, Mitchell Hashimoto mitchell.hashimoto@gmail.com

	Alternatively, the contents of this file may be used under the terms
	of the GNU General Public License Version 3 or later, as described below:

	This file is free software: you may copy, redistribute and/or modify
	it under the terms of the GNU General Public License as published by the
	Free Software Foundation, either version 3 of the License, or (at your
	option) any later version.

	This file is distributed in the hope that it will be useful, but
	WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General
	Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program. If not, see http://www.gnu.org/licenses/.

*/

// Note: this code originally originate from Hashicorp's Serf but has been changed since to fit git-bug's need.

// Note: this Lamport clock implementation is different than the algorithms you can find, notably Wikipedia or the
//       original Serf implementation. The reason is lie to what constitute an event in this distributed system.
//       Commonly, events happen when messages are sent or received, whereas in git-bug events happen when some data is
//       written, but *not* when read. This is why Witness set the time to the max seen value instead of max seen value +1.
//       See https://cs.stackexchange.com/a/133730/129795

package lamport

import (
	"sync/atomic"
)

var _ Clock = &MemClock{}

// MemClock is a thread safe implementation of a lamport clock. It
// uses efficient atomic operations for all of its functions, falling back
// to a heavy lock only if there are enough CAS failures.
type MemClock struct {
	counter uint64
}

// NewMemClock create a new clock with the value 1.
// Value 0 is considered as invalid.
func NewMemClock() *MemClock {
	return &MemClock{
		counter: 1,
	}
}

// NewMemClockWithTime create a new clock with a value.
func NewMemClockWithTime(time uint64) *MemClock {
	return &MemClock{
		counter: time,
	}
}

// Time is used to return the current value of the lamport clock
func (mc *MemClock) Time() Time {
	return Time(atomic.LoadUint64(&mc.counter))
}

// Increment is used to return the value of the lamport clock and increment it afterwards
func (mc *MemClock) Increment() (Time, error) {
	return Time(atomic.AddUint64(&mc.counter, 1)), nil
}

// Witness is called to update our local clock if necessary after
// witnessing a clock value received from another process
func (mc *MemClock) Witness(v Time) error {
WITNESS:
	// If the other value is old, we do not need to do anything
	cur := atomic.LoadUint64(&mc.counter)
	other := uint64(v)
	if other <= cur {
		return nil
	}

	// Ensure that our local clock is at least one ahead.
	if !atomic.CompareAndSwapUint64(&mc.counter, cur, other) {
		// CAS: CompareAndSwap
		// The CAS failed, so we just retry. Eventually our CAS should
		// succeed or a future witness will pass us by and our witness
		// will end.
		goto WITNESS
	}

	return nil
}
