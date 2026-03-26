package sfid

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// IDTag describes the optional external tag used by Typed IDs.
//
// The returned value should be a lowercase ASCII token made of digits,
// lowercase letters, and underscores. It is used verbatim as the string
// prefix for Typed IDs. For example, "u_" formats IDs as "u_<id>", while
// "user_" formats them as "user_<id>".
//
// Implementations are often domain types such as user or org. They do not
// contribute any runtime data to the ID itself; they only let Go distinguish
// one typed ID family from another and associate each family with a type tag.
type IDTag interface {
	IDTag() string
}

// Typed adds type-safe semantics in Go code and can optionally add a string
// prefix to the text form of an ID.
//
// Typed does not alter the binary sfid layout. If T implements IDTag and
// returns a non-empty tag, Typed changes the string, text, and JSON forms by
// prepending that tag. Otherwise it uses the same text form as the underlying
// sfid.ID.
//
// The type parameter T is usually a domain type. For example, Typed[user] and
// Typed[org] are distinct Go types even though both use the same underlying
// sfid.ID representation.
type Typed[T any] ID

// Wrap wraps an ID with an external typed string prefix.
func Wrap[T any](id ID) Typed[T] {
	return Typed[T](id)
}

// NewTyped generates a new ID from g and wraps it with the typed string prefix.
func NewTyped[T any](g *Generator) Typed[T] {
	return Typed[T](g.New())
}

// ParseTyped parses a prefixed typed string such as "u_d000000000000".
//
// The empty string decodes to the zero typed ID.
func ParseTyped[T any](s string) (Typed[T], error) {
	if s == "" {
		return Typed[T](0), nil
	}

	want := typedPrefix[T]()
	if want != "" && !strings.HasPrefix(s, want) {
		return Typed[T](0), fmt.Errorf("%w: got %q, want prefix %q", ErrTypedPrefixMismatch, s, want)
	}

	id, err := Parse(s[len(want):])
	if err != nil {
		return Typed[T](0), err
	}

	return Wrap[T](id), nil
}

// MustParseTyped parses a typed string and panics on error.
func MustParseTyped[T any](s string) Typed[T] {
	id, err := ParseTyped[T](s)
	if err != nil {
		panic(err)
	}

	return id
}

// String returns the canonical typed string form "<prefix><id>".
//
// The zero typed ID encodes as the empty string.
func (id Typed[T]) String() string {
	if id == 0 {
		return ""
	}

	return typedPrefix[T]() + ID(id).String()
}

// Prefix returns the visible sfid prefix character from the underlying ID.
func (id Typed[T]) Prefix() byte {
	return ID(id).Prefix()
}

// Machine returns the machine ID from the underlying sfid.
func (id Typed[T]) Machine() uint16 {
	return ID(id).Machine()
}

// Counter returns the per-bucket counter from the underlying sfid.
func (id Typed[T]) Counter() uint16 {
	return ID(id).Counter()
}

// Tick returns the regression timeline bit from the underlying sfid.
func (id Typed[T]) Tick() uint8 {
	return ID(id).Tick()
}

// Time returns the time carried by the underlying sfid.
func (id Typed[T]) Time() time.Time {
	return ID(id).Time()
}

// MarshalText implements encoding.TextMarshaler.
func (id Typed[T]) MarshalText() ([]byte, error) {
	return appendTypedText(nil, typedPrefix[T](), uint64(id)), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (id *Typed[T]) UnmarshalText(text []byte) error {
	parsed, err := ParseTyped[T](string(text))
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

// MarshalJSON implements json.Marshaler.
func (id Typed[T]) MarshalJSON() ([]byte, error) {
	if id == 0 {
		return []byte(`""`), nil
	}

	prefix := typedPrefix[T]()
	return appendQuotedTyped(make([]byte, 0, len(prefix)+encodedLength+2), prefix, uint64(id)), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *Typed[T]) UnmarshalJSON(src []byte) error {
	s, ok := parseJSONString(src)
	if !ok {
		var decoded string
		if err := json.Unmarshal(src, &decoded); err != nil {
			return err
		}
		s = decoded
	}

	parsed, err := ParseTyped[T](s)
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

func typedPrefix[T any]() string {
	var zero T
	tagger, ok := any(zero).(IDTag)
	if !ok {
		return ""
	}

	prefix := tagger.IDTag()
	if !validTypedPrefix(prefix) {
		panic(fmt.Sprintf("sfid: invalid type tag %q", prefix))
	}

	return prefix
}

func validTypedPrefix(prefix string) bool {
	if prefix == "" {
		return true
	}

	for i := 0; i < len(prefix); i++ {
		ch := prefix[i]
		isDigit := ch >= '0' && ch <= '9'
		isLower := ch >= 'a' && ch <= 'z'
		if !isDigit && !isLower && ch != '_' {
			return false
		}
	}

	return true
}
