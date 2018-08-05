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

package util

import (
	"testing"
)

func TestLamportClock(t *testing.T) {
	l := &LamportClock{}

	if l.Time() != 0 {
		t.Fatalf("bad time value")
	}

	if l.Increment() != 1 {
		t.Fatalf("bad time value")
	}

	if l.Time() != 1 {
		t.Fatalf("bad time value")
	}

	l.Witness(41)

	if l.Time() != 42 {
		t.Fatalf("bad time value")
	}

	l.Witness(41)

	if l.Time() != 42 {
		t.Fatalf("bad time value")
	}

	l.Witness(30)

	if l.Time() != 42 {
		t.Fatalf("bad time value")
	}
}
