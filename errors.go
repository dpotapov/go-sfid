package sfid

import "errors"

var (
	// ErrInvalidMachine reports a machine ID outside the 0..1023 range.
	ErrInvalidMachine = errors.New("sfid: invalid machine id")

	// ErrInvalidPrefix reports an invalid binary prefix nibble or visible prefix char.
	ErrInvalidPrefix = errors.New("sfid: invalid prefix")

	// ErrInvalidLength reports a non-empty text form that is not exactly 13 characters long.
	ErrInvalidLength = errors.New("sfid: invalid encoded length")

	// ErrInvalidCharacter reports an invalid character in the encoded text form.
	ErrInvalidCharacter = errors.New("sfid: invalid encoded character")

	// ErrTimeBeforeEpoch reports a timestamp before 2025-01-01T00:00:00Z.
	ErrTimeBeforeEpoch = errors.New("sfid: timestamp before epoch")

	// ErrTimeOverflow reports a timestamp beyond the representable 39-bit range.
	ErrTimeOverflow = errors.New("sfid: timestamp overflow")

	// ErrTypedPrefixMismatch reports a mismatch between a typed string prefix and its target type.
	ErrTypedPrefixMismatch = errors.New("sfid: typed prefix mismatch")

	// ErrInvalidScanSource reports an unsupported database/sql scan source type.
	ErrInvalidScanSource = errors.New("sfid: invalid scan source")

	// ErrInvalidScanValue reports a numeric scan source outside the valid ID range.
	ErrInvalidScanValue = errors.New("sfid: invalid scan value")
)
