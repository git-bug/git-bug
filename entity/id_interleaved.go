package entity

import (
	"strings"
)

// CombineIds compute a merged Id holding information from both the primary Id
// and the secondary Id.
//
// This allow to later find efficiently a secondary element because we can access
// the primary one directly instead of searching for a primary that has a
// secondary matching the Id.
//
// An example usage is Comment in a Bug. The interleaved Id will hold part of the
// Bug Id and part of the Comment Id.
//
// To allow the use of an arbitrary length prefix of this Id, Ids from primary
// and secondary are interleaved with this irregular pattern to give the
// best chance to find the secondary even with a 7 character prefix.
//
// Format is: PSPSPSPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPPSPPPP
//
// A complete interleaved Id hold 50 characters for the primary and 14 for the
// secondary, which give a key space of 36^50 for the primary (~6 * 10^77) and
// 36^14 for the secondary (~6 * 10^21). This asymmetry assume a reasonable number
// of secondary within a primary Entity, while still allowing for a vast key space
// for the primary (that is, a globally merged database) with a low risk of collision.
//
// Here is the breakdown of several common prefix length:
//
// 5:    3P, 2S
// 7:    4P, 3S
// 10:   6P, 4S
// 16:  11P, 5S
func CombineIds(primary Id, secondary Id) Id {
	var id strings.Builder

	for i := 0; i < idLength; i++ {
		switch {
		default:
			id.WriteByte(primary[0])
			primary = primary[1:]
		case i == 1, i == 3, i == 5, i == 9, i >= 10 && i%5 == 4:
			id.WriteByte(secondary[0])
			secondary = secondary[1:]
		}
	}

	return Id(id.String())
}

// SeparateIds extract primary and secondary prefix from an arbitrary length prefix
// of an Id created with CombineIds.
func SeparateIds(prefix string) (primaryPrefix string, secondaryPrefix string) {
	var primary strings.Builder
	var secondary strings.Builder

	for i, r := range prefix {
		switch {
		default:
			primary.WriteRune(r)
		case i == 1, i == 3, i == 5, i == 9, i >= 10 && i%5 == 4:
			secondary.WriteRune(r)
		}
	}

	return primary.String(), secondary.String()
}
