package sfid

import (
	"bytes"
	"fmt"
	"time"
)

const (
	prefixBits  = 3
	timeBits    = 39
	tickBits    = 1
	machineBits = 10
	counterBits = 10

	counterShift = 0
	machineShift = counterShift + counterBits
	tickShift    = machineShift + machineBits
	timeShift    = tickShift + tickBits
	prefixShift  = timeShift + timeBits
	signShift    = prefixShift + prefixBits

	prefixMask  = uint64(1<<prefixBits - 1)
	timeMask    = uint64(1<<timeBits - 1)
	tickMask    = uint64(1<<tickBits - 1)
	machineMask = uint64(1<<machineBits - 1)
	counterMask = uint64(1<<counterBits - 1)
	payloadMask = uint64(1<<prefixShift - 1)

	maxMachine   = uint16(machineMask)
	maxCounter   = uint16(counterMask)
	maxTimestamp = int64(timeMask)

	encodedLength = 13
	bucketMillis  = int64(4)
)

const encodeAlphabet = "0123456789abcdefghijklmnopqrstuv"
const prefixAlphabet = "abcdefgh"

var (
	bucketDuration = 4 * time.Millisecond
	epochTime      = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	epochUnixMilli = epochTime.UnixMilli()
	decodeTable    = newDecodeTable()
	prefixTable    = newPrefixTable()
)

func newDecodeTable() [256]byte {
	var table [256]byte
	for i := range table {
		table[i] = 0xff
	}

	for i := range encodeAlphabet {
		table[encodeAlphabet[i]] = byte(i)
	}

	return table
}

func newPrefixTable() [256]byte {
	var table [256]byte
	for i := range table {
		table[i] = 0xff
	}

	for i := range prefixAlphabet {
		table[prefixAlphabet[i]] = byte(i)
	}

	return table
}

func encodeID(v uint64) string {
	var buf [encodedLength]byte

	buf[0] = prefixAlphabet[(v>>prefixShift)&prefixMask]
	v &= payloadMask

	for i := encodedLength - 1; i >= 1; i-- {
		buf[i] = encodeAlphabet[v&0x1f]
		v >>= 5
	}

	return string(buf[:])
}

func appendEncodedID(dst []byte, v uint64) []byte {
	var buf [encodedLength]byte

	buf[0] = prefixAlphabet[(v>>prefixShift)&prefixMask]
	v &= payloadMask

	for i := encodedLength - 1; i >= 1; i-- {
		buf[i] = encodeAlphabet[v&0x1f]
		v >>= 5
	}

	return append(dst, buf[:]...)
}

func appendQuotedEncodedID(dst []byte, v uint64) []byte {
	dst = append(dst, '"')
	dst = appendEncodedID(dst, v)
	dst = append(dst, '"')
	return dst
}

func appendTextID(dst []byte, v uint64) []byte {
	if v == 0 {
		return dst
	}

	return appendEncodedID(dst, v)
}

func appendTypedText(dst []byte, prefix string, v uint64) []byte {
	if v == 0 {
		return dst
	}

	dst = append(dst, prefix...)
	return appendEncodedID(dst, v)
}

func appendQuotedTyped(dst []byte, prefix string, v uint64) []byte {
	dst = append(dst, '"')
	dst = appendTypedText(dst, prefix, v)
	dst = append(dst, '"')
	return dst
}

func decodeString(s string) (uint64, error) {
	if len(s) != encodedLength {
		return 0, fmt.Errorf("%w: got %d, want %d", ErrInvalidLength, len(s), encodedLength)
	}

	first := prefixTable[s[0]]
	if first == 0xff {
		return 0, invalidCharacterError(s[0], 0)
	}

	v := uint64(first)
	for i := 1; i < encodedLength; i++ {
		digit := decodeTable[s[i]]
		if digit == 0xff {
			return 0, invalidCharacterError(s[i], i)
		}

		v = (v << 5) | uint64(digit)
	}

	return v, nil
}

func invalidCharacterError(ch byte, index int) error {
	return fmt.Errorf("%w: %q at index %d", ErrInvalidCharacter, ch, index)
}

func parseJSONString(src []byte) (string, bool) {
	if len(src) < 2 || src[0] != '"' || src[len(src)-1] != '"' {
		return "", false
	}

	body := src[1 : len(src)-1]
	if bytes.IndexByte(body, '\\') >= 0 {
		return "", false
	}

	return string(body), true
}

func normalizePrefix(prefix byte) (uint8, error) {
	switch {
	case prefix <= byte(prefixMask):
		return uint8(prefix), nil
	case prefix >= 'a' && prefix <= 'h':
		return uint8(prefix - 'a'), nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrInvalidPrefix, prefix)
	}
}

func visiblePrefix(prefix uint8) byte {
	return prefixAlphabet[prefix&uint8(prefixMask)]
}

func bucketFromTime(t time.Time) (int64, error) {
	return bucketFromUnixMilli(t.UnixMilli())
}

func bucketFromUnixMilli(unixMilli int64) (int64, error) {
	if unixMilli < epochUnixMilli {
		return 0, ErrTimeBeforeEpoch
	}

	bucket := (unixMilli - epochUnixMilli) / bucketMillis
	if bucket > maxTimestamp {
		return 0, ErrTimeOverflow
	}

	return bucket, nil
}

func timeForBucket(bucket int64) time.Time {
	return time.UnixMilli(epochUnixMilli + bucket*bucketMillis).UTC()
}

func durationUntilBucket(now time.Time, bucket int64) time.Duration {
	waitMillis := epochUnixMilli + bucket*bucketMillis - now.UnixMilli()
	if waitMillis <= 0 {
		return 0
	}

	return time.Duration(waitMillis) * time.Millisecond
}
