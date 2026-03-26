package sfid

import (
	"testing"
)

func BenchmarkSFIDNew(b *testing.B) {
	g := MustNewGenerator(1, 'd')

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = g.New()
	}
}

func BenchmarkSFIDString(b *testing.B) {
	id := MustNewGenerator(1, 'd').New()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = id.String()
	}
}

func BenchmarkSFIDParse(b *testing.B) {
	s := MustNewGenerator(1, 'd').NewString()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = Parse(s)
	}
}
