package sfid

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"errors"
	"testing"
)

type user struct{}

func (user) IDTag() string { return "u_" }

type org struct{}

func (org) IDTag() string { return "org_" }

type account struct{}

type UserID = Typed[user]
type AccountID = Typed[account]

var (
	_ encoding.TextMarshaler   = UserID(0)
	_ encoding.TextUnmarshaler = (*UserID)(nil)
	_ json.Marshaler           = UserID(0)
	_ json.Unmarshaler         = (*UserID)(nil)
	_ encoding.TextMarshaler   = AccountID(0)
	_ encoding.TextUnmarshaler = (*AccountID)(nil)
	_ json.Marshaler           = AccountID(0)
	_ json.Unmarshaler         = (*AccountID)(nil)
	_ driver.Valuer            = UserID(0)
	_ sql.Scanner              = (*UserID)(nil)
)

func TestWrapTypedString(t *testing.T) {
	id := MustParse("d000000000000")
	typed := Wrap[user](id)

	if got, want := typed.String(), "u_d000000000000"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}

	if ID(typed) != id {
		t.Fatalf("wrapped ID = %v, want %v", ID(typed), id)
	}
}

func TestNewTyped(t *testing.T) {
	g := MustNewGenerator(42, 'd')
	typed := NewTyped[user](g)

	if typed == 0 {
		t.Fatal("NewTyped() returned zero ID")
	}

	if got := typed.String(); len(got) != len("u_d000000000000") {
		t.Fatalf("len(String()) = %d, want %d", len(got), len("u_d000000000000"))
	}

	if got := typed.String()[:2]; got != "u_" {
		t.Fatalf("String() prefix = %q, want %q", got, "u_")
	}

	if got := typed.Machine(); got != 42 {
		t.Fatalf("Machine() = %d, want 42", got)
	}
}

func TestParseTyped(t *testing.T) {
	got, err := ParseTyped[user]("u_d000000000000")
	if err != nil {
		t.Fatalf("ParseTyped() error = %v", err)
	}

	if ID(got) != MustParse("d000000000000") {
		t.Fatalf("ParseTyped() = %v, want %v", ID(got), MustParse("d000000000000"))
	}
}

func TestParseTypedRejectsPrefixMismatch(t *testing.T) {
	_, err := ParseTyped[user]("org_d000000000000")
	if !errors.Is(err, ErrTypedPrefixMismatch) {
		t.Fatalf("ParseTyped() error = %v, want ErrTypedPrefixMismatch", err)
	}
}

func TestTypedTextJSONRoundTrip(t *testing.T) {
	typed := Wrap[org](MustParse("d000000000000"))

	text, err := typed.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}

	var fromText Typed[org]
	if err := fromText.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if fromText != typed {
		t.Fatalf("text round trip = %v, want %v", fromText, typed)
	}

	data, err := json.Marshal(typed)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if got, want := string(data), `"org_d000000000000"`; got != want {
		t.Fatalf("json.Marshal() = %q, want %q", got, want)
	}

	var fromJSON Typed[org]
	if err := json.Unmarshal(data, &fromJSON); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if fromJSON != typed {
		t.Fatalf("json round trip = %v, want %v", fromJSON, typed)
	}
}

func TestZeroTypedStringAndParsing(t *testing.T) {
	var typed Typed[user]

	if got := typed.String(); got != "" {
		t.Fatalf("String() = %q, want empty string", got)
	}

	parsed, err := ParseTyped[user]("")
	if err != nil {
		t.Fatalf("ParseTyped(\"\") error = %v", err)
	}

	if parsed != 0 {
		t.Fatalf("ParseTyped(\"\") = %v, want zero typed ID", parsed)
	}
}

func TestMustParseTypedPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustParseTyped should panic for invalid input")
		}
	}()

	_ = MustParseTyped[user]("x_invalid")
}

func TestTypedAliasMethodsAndConversion(t *testing.T) {
	base := MustParse("d000000000000")
	uid := UserID(base)

	if got, want := uid.String(), "u_d000000000000"; got != want {
		t.Fatalf("UserID.String() = %q, want %q", got, want)
	}

	text, err := uid.MarshalText()
	if err != nil {
		t.Fatalf("UserID.MarshalText() error = %v", err)
	}
	if got, want := string(text), "u_d000000000000"; got != want {
		t.Fatalf("UserID.MarshalText() = %q, want %q", got, want)
	}

	data, err := json.Marshal(uid)
	if err != nil {
		t.Fatalf("json.Marshal(UserID) error = %v", err)
	}
	if got, want := string(data), `"u_d000000000000"`; got != want {
		t.Fatalf("json.Marshal(UserID) = %q, want %q", got, want)
	}

	if got, want := int64(uid), int64(base); got != want {
		t.Fatalf("int64(UserID) = %d, want %d", got, want)
	}

	if got, want := ID(uid), base; got != want {
		t.Fatalf("ID(UserID) = %v, want %v", got, want)
	}

	if got, want := uid.Time(), base.Time(); got != want {
		t.Fatalf("UserID.Time() = %v, want %v", got, want)
	}
	if got, want := uid.Machine(), base.Machine(); got != want {
		t.Fatalf("UserID.Machine() = %d, want %d", got, want)
	}
	if got, want := uid.Counter(), base.Counter(); got != want {
		t.Fatalf("UserID.Counter() = %d, want %d", got, want)
	}
	if got, want := uid.Tick(), base.Tick(); got != want {
		t.Fatalf("UserID.Tick() = %d, want %d", got, want)
	}
}

func TestTypedWithoutTagUsesBaseStringForm(t *testing.T) {
	base := MustParse("d000000000000")
	aid := AccountID(base)

	if got, want := aid.String(), base.String(); got != want {
		t.Fatalf("AccountID.String() = %q, want %q", got, want)
	}

	text, err := aid.MarshalText()
	if err != nil {
		t.Fatalf("AccountID.MarshalText() error = %v", err)
	}
	if got, want := string(text), base.String(); got != want {
		t.Fatalf("AccountID.MarshalText() = %q, want %q", got, want)
	}

	data, err := json.Marshal(aid)
	if err != nil {
		t.Fatalf("json.Marshal(AccountID) error = %v", err)
	}
	if got, want := string(data), `"d000000000000"`; got != want {
		t.Fatalf("json.Marshal(AccountID) = %q, want %q", got, want)
	}

	parsed, err := ParseTyped[account](base.String())
	if err != nil {
		t.Fatalf("ParseTyped[account]() error = %v", err)
	}
	if got, want := ID(parsed), base; got != want {
		t.Fatalf("ParseTyped[account]() = %v, want %v", got, want)
	}
}
