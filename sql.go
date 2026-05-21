package sfid

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"math"
)

// Value implements driver.Valuer.
//
// Non-zero IDs encode as int64. Zero encodes as SQL NULL because the zero ID
// means absent/unset, and NULL is SQL's native representation of a missing value.
// Storing 0 instead would change FK and UNIQUE semantics (for example, multiple
// unset rows can share NULL under a UNIQUE constraint, but not 0). To persist
// zero as the numeric 0 in a NOT NULL column, pass int64(id) explicitly.
func (id ID) Value() (driver.Value, error) {
	if id == 0 {
		return nil, nil
	}
	return int64(id), nil
}

// Value implements driver.Valuer.
func (id Typed[T]) Value() (driver.Value, error) {
	return ID(id).Value()
}

// Scan implements sql.Scanner.
func (id *ID) Scan(src any) error {
	parsed, err := scanRawID(src)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// Scan implements sql.Scanner.
func (id *Typed[T]) Scan(src any) error {
	parsed, err := scanTypedID[T](src)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

func scanRawID(src any) (ID, error) {
	switch v := src.(type) {
	case nil:
		return 0, nil
	case int64:
		return idFromInt64(v)
	case int:
		return idFromInt64(int64(v))
	case int32:
		return idFromInt64(int64(v))
	case uint64:
		return idFromUint64(v)
	case string:
		return Parse(v)
	case []byte:
		if len(v) == 8 {
			return idFromUint64(binary.BigEndian.Uint64(v))
		}
		return Parse(string(v))
	default:
		return 0, fmt.Errorf("sfid: scan %T: %w", src, ErrInvalidScanSource)
	}
}

func scanTypedID[T any](src any) (Typed[T], error) {
	switch v := src.(type) {
	case nil:
		return 0, nil
	case int64:
		id, err := idFromInt64(v)
		if err != nil {
			return 0, err
		}
		return Wrap[T](id), nil
	case int:
		id, err := idFromInt64(int64(v))
		if err != nil {
			return 0, err
		}
		return Wrap[T](id), nil
	case int32:
		id, err := idFromInt64(int64(v))
		if err != nil {
			return 0, err
		}
		return Wrap[T](id), nil
	case uint64:
		id, err := idFromUint64(v)
		if err != nil {
			return 0, err
		}
		return Wrap[T](id), nil
	case string:
		return ParseTyped[T](v)
	case []byte:
		if len(v) == 8 {
			id, err := idFromUint64(binary.BigEndian.Uint64(v))
			if err != nil {
				return 0, err
			}
			return Wrap[T](id), nil
		}
		return ParseTyped[T](string(v))
	default:
		return 0, fmt.Errorf("sfid: scan %T: %w", src, ErrInvalidScanSource)
	}
}

func idFromInt64(v int64) (ID, error) {
	if v < 0 {
		return 0, fmt.Errorf("sfid: scan int64 %d: %w", v, ErrInvalidScanValue)
	}
	return ID(v), nil
}

func idFromUint64(v uint64) (ID, error) {
	if v > math.MaxInt64 {
		return 0, fmt.Errorf("sfid: scan uint64 %d: %w", v, ErrInvalidScanValue)
	}
	return ID(int64(v)), nil
}
