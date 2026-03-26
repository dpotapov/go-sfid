package sfid

import (
	"encoding/json"
	"testing"
)

func TestIDMarshalTextRoundTrip(t *testing.T) {
	id := packID(0x0d, 42, 1, 512, 7)

	text, err := id.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}

	var decoded ID
	if err := decoded.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if decoded != id {
		t.Fatalf("round trip = %v, want %v", decoded, id)
	}
}

func TestZeroIDMarshalTextRoundTrip(t *testing.T) {
	var id ID

	text, err := id.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}

	if got := string(text); got != "" {
		t.Fatalf("MarshalText() = %q, want empty string", got)
	}

	var decoded ID
	if err := decoded.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if decoded != 0 {
		t.Fatalf("round trip = %v, want zero ID", decoded)
	}
}

func TestIDMarshalJSONRoundTrip(t *testing.T) {
	id := packID(0x02, 99, 0, 7, 31)

	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if got, want := string(data), `"`+id.String()+`"`; got != want {
		t.Fatalf("json.Marshal() = %q, want %q", got, want)
	}

	var decoded ID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != id {
		t.Fatalf("round trip = %v, want %v", decoded, id)
	}
}

func TestZeroIDMarshalJSONRoundTrip(t *testing.T) {
	var id ID

	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if got, want := string(data), `""`; got != want {
		t.Fatalf("json.Marshal() = %q, want %q", got, want)
	}

	var decoded ID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != 0 {
		t.Fatalf("round trip = %v, want zero ID", decoded)
	}
}

func TestIDUnmarshalJSONRejectsNonString(t *testing.T) {
	var id ID
	if err := json.Unmarshal([]byte(`123`), &id); err == nil {
		t.Fatal("expected unmarshal error for non-string JSON")
	}
}
