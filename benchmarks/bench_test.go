package benchmarks

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	sfid "github.com/dpotapov/go-sfid"
	"github.com/google/uuid"
	"github.com/muyo/sno"
	"github.com/oklog/ulid/v2"
	"github.com/rs/xid"
	"github.com/sony/sonyflake"
)

func BenchmarkCompareNew(b *testing.B) {
	snowflakeNode, err := snowflake.NewNode(1)
	if err != nil {
		b.Fatalf("snowflake.NewNode: %v", err)
	}
	sonyflakeGen := mustSonyflake(b)

	b.Run("sfid", func(b *testing.B) {
		g := sfid.MustNewGenerator(1, 'd')
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = g.New()
		}
	})

	b.Run("sno", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = sno.New(0)
		}
	})

	b.Run("xid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xid.New()
		}
	})

	b.Run("snowflake", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = snowflakeNode.Generate()
		}
	})

	b.Run("sonyflake", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = sonyflakeGen.NextID()
		}
	})

	b.Run("ulid", func(b *testing.B) {
		entropy := ulid.Monotonic(rand.New(rand.NewSource(1)), 0)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = ulid.New(ulid.Timestamp(time.Now().UTC()), entropy)
		}
	})

	b.Run("uuid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = uuid.New()
		}
	})
}

func BenchmarkCompareString(b *testing.B) {
	sfidID := sfid.MustNewGenerator(1, 'd').New()
	snoID := sno.New(0)
	xidID := xid.New()
	snowflakeNode, err := snowflake.NewNode(1)
	if err != nil {
		b.Fatalf("snowflake.NewNode: %v", err)
	}
	snowflakeID := snowflakeNode.Generate()
	sonyflakeGen := mustSonyflake(b)
	sonyflakeID, err := sonyflakeGen.NextID()
	if err != nil {
		b.Fatalf("sonyflake.NextID: %v", err)
	}
	entropy := ulid.Monotonic(rand.New(rand.NewSource(1)), 0)
	ulidID, _ := ulid.New(ulid.Timestamp(time.Now().UTC()), entropy)
	uuidID := uuid.New()

	b.Run("sfid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = sfidID.String()
		}
	})

	b.Run("sno", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = snoID.String()
		}
	})

	b.Run("xid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xidID.String()
		}
	})

	b.Run("snowflake", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = snowflakeID.String()
		}
	})

	b.Run("sonyflake", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = strconv.FormatUint(sonyflakeID, 10)
		}
	})

	b.Run("ulid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ulidID.String()
		}
	})

	b.Run("uuid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = uuidID.String()
		}
	})
}

func BenchmarkCompareParse(b *testing.B) {
	sfidString := sfid.MustNewGenerator(1, 'd').NewString()
	snoID := sno.New(0)
	snoString := snoID.String()
	xidString := xid.New().String()
	snowflakeNode, err := snowflake.NewNode(1)
	if err != nil {
		b.Fatalf("snowflake.NewNode: %v", err)
	}
	snowflakeString := snowflakeNode.Generate().String()
	sonyflakeGen := mustSonyflake(b)
	sonyflakeID, err := sonyflakeGen.NextID()
	if err != nil {
		b.Fatalf("sonyflake.NextID: %v", err)
	}
	sonyflakeString := strconv.FormatUint(sonyflakeID, 10)
	entropy := ulid.Monotonic(rand.New(rand.NewSource(1)), 0)
	ulidID, _ := ulid.New(ulid.Timestamp(time.Now().UTC()), entropy)
	ulidString := ulidID.String()
	uuidString := uuid.New().String()

	b.Run("sfid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = sfid.Parse(sfidString)
		}
	})

	b.Run("sno", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = sno.FromEncodedString(snoString)
		}
	})

	b.Run("xid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = xid.FromString(xidString)
		}
	})

	b.Run("snowflake", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = snowflake.ParseString(snowflakeString)
		}
	})

	b.Run("sonyflake", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = strconv.ParseUint(sonyflakeString, 10, 64)
		}
	})

	b.Run("ulid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = ulid.ParseStrict(ulidString)
		}
	})

	b.Run("uuid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = uuid.Parse(uuidString)
		}
	})
}

func mustSonyflake(b *testing.B) *sonyflake.Sonyflake {
	b.Helper()

	gen := sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		MachineID: func() (uint16, error) { return 1, nil },
		CheckMachineID: func(id uint16) bool {
			return id == 1
		},
	})
	if gen == nil {
		b.Fatal("sonyflake.NewSonyflake returned nil")
	}

	return gen
}
