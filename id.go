package sfid

import (
	"encoding/json"
	"fmt"
	"time"
)

// ID is a compact 64-bit distributed identifier.
type ID int64

// Parse parses a canonical lowercase base32hex sfid string.
//
// The empty string decodes to the zero ID. All non-zero IDs must use the
// canonical 13-character lowercase form.
func Parse(s string) (ID, error) {
	if s == "" {
		return 0, nil
	}

	v, err := decodeString(s)
	if err != nil {
		return 0, err
	}

	return ID(v), nil
}

// MustParse parses s and panics if it is not a valid sfid string.
func MustParse(s string) ID {
	id, err := Parse(s)
	if err != nil {
		panic(err)
	}

	return id
}

// String returns the canonical lowercase base32hex form of the ID.
//
// The zero ID encodes as the empty string. All non-zero IDs encode as exactly
// 13 lowercase base32hex characters.
func (id ID) String() string {
	if id == 0 {
		return ""
	}

	return encodeID(uint64(id))
}

// Prefix returns the visible lowercase prefix character in the range a-h.
func (id ID) Prefix() byte {
	return visiblePrefix(id.prefixNibble())
}

// Machine returns the 10-bit machine ID.
func (id ID) Machine() uint16 {
	return uint16((uint64(id) >> machineShift) & machineMask)
}

// Counter returns the 10-bit per-bucket counter.
func (id ID) Counter() uint16 {
	return uint16(uint64(id) & counterMask)
}

// Tick returns the 1-bit alternate timeline marker.
func (id ID) Tick() uint8 {
	return uint8((uint64(id) >> tickShift) & tickMask)
}

// Time returns the timestamp bucket as a UTC time rounded down to 4 milliseconds.
func (id ID) Time() time.Time {
	return timeForBucket(id.bucket())
}

// MarshalText implements encoding.TextMarshaler.
func (id ID) MarshalText() ([]byte, error) {
	return appendTextID(nil, uint64(id)), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (id *ID) UnmarshalText(text []byte) error {
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

// MarshalJSON implements json.Marshaler.
func (id ID) MarshalJSON() ([]byte, error) {
	if id == 0 {
		return []byte(`""`), nil
	}

	return appendQuotedEncodedID(make([]byte, 0, encodedLength+2), uint64(id)), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *ID) UnmarshalJSON(src []byte) error {
	s, ok := parseJSONString(src)
	if !ok {
		var decoded string
		if err := json.Unmarshal(src, &decoded); err != nil {
			return err
		}
		s = decoded
	}

	parsed, err := Parse(s)
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

func (id ID) prefixNibble() uint8 {
	return uint8((uint64(id) >> prefixShift) & prefixMask)
}

func (id ID) bucket() int64 {
	return int64((uint64(id) >> timeShift) & timeMask)
}

func packID(prefix uint8, bucket int64, tick uint8, machine, counter uint16) ID {
	if bucket < 0 || bucket > maxTimestamp {
		panic(fmt.Sprintf("sfid: bucket %d out of range", bucket))
	}

	v := (uint64(prefix&uint8(prefixMask)) << prefixShift) |
		((uint64(bucket) & timeMask) << timeShift) |
		((uint64(tick) & tickMask) << tickShift) |
		((uint64(machine) & machineMask) << machineShift) |
		(uint64(counter) & counterMask)

	return ID(int64(v))
}
