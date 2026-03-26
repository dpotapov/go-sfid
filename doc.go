/*
Package sfid provides sortable 64-bit IDs for distributed systems with a
compact string representation.

An sfid ID uses the following fixed layout, from most-significant bit to
least-significant bit:

	reserved[1] | prefix[3] | time[39] | tick[1] | machine[10] | counter[10]

The timestamp stores 4 millisecond buckets relative to the epoch
2025-01-01T00:00:00Z, which gives about 69.7 years of range. The tick bit lets
a generator survive bounded backward clock movement by switching to an
alternate timeline instead of immediately reusing a bucket it has already
emitted on. The machine and counter fields keep the API small while still
allowing concurrent, distributed generation.

The canonical text form is a fixed-width, lowercase base32hex string using the
alphabet 0-9a-v.

The zero ID encodes as an empty string. Every non-zero encoded ID is exactly 13
characters long. The first character is the visible prefix in the range a-h,
and the remaining 12 characters encode the remaining 60 bits in lowercase
base32hex. Example IDs look like a000000000000 or d30bnjf5ak010.

SFID keeps its public API intentionally small: a Generator, a 64-bit ID type,
strict parsing, and an optional typed string wrapper for conventions such as
u_<id>. Typed prefixes are used exactly as configured in the string layer and
do not alter the binary ID layout.
*/
package sfid
