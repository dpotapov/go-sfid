package sfid

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"slices"
	"testing"
)

var (
	_ driver.Valuer = ID(0)
	_ sql.Scanner   = (*ID)(nil)
)

func TestZeroIDVector(t *testing.T) {
	t.Helper()

	id := ID(0)

	if got := id.String(); got != "" {
		t.Fatalf("String() = %q, want empty string", got)
	}

	if id != 0 {
		t.Fatal("zero ID should compare equal to 0")
	}

	if got := id.Prefix(); got != 'a' {
		t.Fatalf("Prefix() = %q, want %q", got, 'a')
	}

	if got := id.Machine(); got != 0 {
		t.Fatalf("Machine() = %d, want 0", got)
	}

	if got := id.Counter(); got != 0 {
		t.Fatalf("Counter() = %d, want 0", got)
	}

	if got := id.Tick(); got != 0 {
		t.Fatalf("Tick() = %d, want 0", got)
	}

	if got := id.Time(); !got.Equal(epochTime) {
		t.Fatalf("Time() = %v, want %v", got, epochTime)
	}

	roundTrip, err := Parse(id.String())
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if roundTrip != id {
		t.Fatalf("Parse(String()) = %v, want %v", roundTrip, id)
	}
}

func TestIDAccessors(t *testing.T) {
	id := packID(0x03, 12345, 1, 42, 7)

	if got := id.Prefix(); got != 'd' {
		t.Fatalf("Prefix() = %q, want %q", got, 'd')
	}

	if got := id.Machine(); got != 42 {
		t.Fatalf("Machine() = %d, want 42", got)
	}

	if got := id.Counter(); got != 7 {
		t.Fatalf("Counter() = %d, want 7", got)
	}

	if got := id.Tick(); got != 1 {
		t.Fatalf("Tick() = %d, want 1", got)
	}

	if got := id.Time(); !got.Equal(timeForBucket(12345)) {
		t.Fatalf("Time() = %v, want %v", got, timeForBucket(12345))
	}

	if got := ID(int64(id)); got != id {
		t.Fatalf("ID(int64(id)) = %v, want %v", got, id)
	}
}

func TestParseRoundTrip(t *testing.T) {
	ids := []ID{
		ID(0),
		packID(0x01, 1, 0, 1, 1),
		packID(0x07, 12345, 1, maxMachine, maxCounter),
		packID(0x07, maxTimestamp, 1, maxMachine, maxCounter),
	}

	for _, id := range ids {
		s := id.String()
		if id == 0 {
			if s != "" {
				t.Fatalf("zero String() = %q, want empty string", s)
			}
		} else if len(s) != encodedLength {
			t.Fatalf("len(String()) = %d, want %d", len(s), encodedLength)
		}

		for i := 0; i < len(s); i++ {
			ch := s[i]
			if i == 0 {
				if ch < 'a' || ch > 'h' {
					t.Fatalf("String() returned invalid prefix char %q", ch)
				}
				continue
			}

			isDigit := ch >= '0' && ch <= '9'
			isLower := ch >= 'a' && ch <= 'v'
			if !isDigit && !isLower {
				t.Fatalf("String() returned non-lowercase base32hex char %q", ch)
			}
		}

		parsed, err := Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", s, err)
		}

		if parsed != id {
			t.Fatalf("Parse(%q) = %v, want %v", s, parsed, id)
		}
	}
}

func TestEncodedFirstCharacterBounds(t *testing.T) {
	values := []ID{
		ID(0),
		packID(0x07, 0, 0, 0, 0),
		packID(0x07, maxTimestamp, 1, maxMachine, maxCounter),
	}

	for _, id := range values {
		if id == 0 {
			if got := id.String(); got != "" {
				t.Fatalf("zero String() = %q, want empty string", got)
			}
			continue
		}

		first := id.String()[0]
		if first < 'a' || first > 'h' {
			t.Fatalf("first char = %q, want in a..h", first)
		}
	}
}

func TestParseRejectsInvalidLength(t *testing.T) {
	cases := []string{
		"000000000000",
		"00000000000000",
	}

	for _, tc := range cases {
		_, err := Parse(tc)
		if !errors.Is(err, ErrInvalidLength) {
			t.Fatalf("Parse(%q) error = %v, want ErrInvalidLength", tc, err)
		}
	}
}

func TestParseEmptyStringReturnsZero(t *testing.T) {
	id, err := Parse("")
	if err != nil {
		t.Fatalf("Parse(\"\") error = %v", err)
	}

	if id != 0 {
		t.Fatalf("Parse(\"\") = %v, want zero ID", id)
	}
}

func TestParseRejectsInvalidCharacters(t *testing.T) {
	cases := []string{
		"i000000000000",
		"G000000000000",
		"d!00000000000",
		"d00000000000W",
	}

	for _, tc := range cases {
		_, err := Parse(tc)
		if !errors.Is(err, ErrInvalidCharacter) {
			t.Fatalf("Parse(%q) error = %v, want ErrInvalidCharacter", tc, err)
		}
	}
}

func TestLexicographicOrderMatchesNumericOrder(t *testing.T) {
	ids := []ID{
		ID(0),
		ID(1),
		ID(31),
		ID(32),
		packID(0x01, 1, 0, 0, 0),
		packID(0x01, 1, 0, 0, 1),
		packID(0x01, 2, 0, 0, 0),
		packID(0x07, maxTimestamp, 1, maxMachine, maxCounter),
	}

	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}

	sorted := slices.Clone(strs)
	slices.Sort(sorted)

	if !slices.Equal(strs, sorted) {
		t.Fatalf("lexicographic order = %v, want %v", sorted, strs)
	}
}

func TestMustParsePanics(t *testing.T) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("MustParse should panic for invalid input")
		}
	}()

	_ = MustParse("invalid")
}
