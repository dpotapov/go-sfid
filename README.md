# go-sfid

`go-sfid` is an opinionated 64-bit distributed ID library for Go.

It is designed for compact, fixed-width IDs that are pleasant to use in URLs, TUIs, CLIs, logs, and JSON while still preserving time-major sortability under normal forward clock progression.

Install:

```bash
go get -u github.com/dpotapov/go-sfid
```

Minimal example:

```go
package main

import (
	"fmt"

	"github.com/dpotapov/go-sfid"
)

func main() {
	g := sfid.MustNewGenerator(42, 'd')
	fmt.Println(g.NewString())
}
```

## Why sfid

`sfid` is built around two practical constraints:

- in a database, the natural representation should be a 64-bit integer
- in text, the same ID should stay short, fixed-width, lowercase, and easy to scan

Time-ordered IDs keep new values near each other in B-tree indexes instead of scattering inserts across random pages. That improves index locality and clustering, which is one of the main practical advantages of ordered IDs over random identifiers.

The core shape is simple:

- 64-bit layout
- 13-character lowercase text form for every non-zero ID
- time-major ordering under normal forward clock progression
- one generator identity: machine ID plus a 3-bit prefix

This design deliberately chooses readability, compactness, and fixed-width text over maximum per-node throughput. A single generator can emit up to about `256,000` IDs per second before it starts waiting for the next 4 ms bucket, which is usually a comfortable ceiling for domain objects, aggregates, entities, and similar business records. It is a poor fit for very high-rate streams such as per-request IDs for busy edge services, log-event IDs, metrics samples, tracing spans, market-data ticks, or other telemetry/event pipelines that may need millions of IDs per second from one process.

### Trade-offs at a glance

| Scheme | Text | Example | DB form | Order / locality | Throughput | Lifespan | Configuration |
| --- | ---: | --- | --- | --- | --- | --- | --- |
| `sfid` | 13 | `d30bnjf5ak010` | 64-bit integer | yes; text and numeric order match within a fixed prefix | 256 IDs/ms per generator | `2025-01-01` to `2094-09-07` | machine ID + 3-bit prefix |
| snowflake | up to 19 | `1984329134502197248` | 64-bit integer | yes; time-major | 4096 IDs/ms per node | about 69 years from the chosen epoch | usually worker / node configuration |
| sonyflake | up to 20 | `548240135131480293` | 64-bit integer | yes; time-major | 256 IDs/10 ms per machine | about 174 years from the chosen start time | machine ID + optional start time / machine function |
| `xid` | 20 | `9m4e2mr0ui3e8a215n4g` | 12-byte value | yes; k-ordered | 16,777 IDs/ms per host/process | `1970` to `2106` | none |
| `sno` | 16 | `2f8x6q9w3k7n4d2m` | 10-byte value | yes; k-sortable | 16,384 IDs/ms per partition+meta | `2010` to `2079` | optional partition / sequence configuration |
| `ulid` | 26 | `01ARZ3NDEKTSV4RRFFQ69G5FAV` | 16-byte value | yes; lexicographically sortable | implementation-defined within a millisecond | until year `10889` | none |
| `uuid` | 36 | `550e8400-e29b-41d4-a716-446655440000` | 16-byte value | version-dependent; random UUIDs do not have locality | implementation-defined | version-dependent | none |

These throughput figures are per generator, node, or process for the standard format, not cluster totals.

## Binary layout

```text
63 62        60 59                  21 20 19        10 9         0
+--+------------+----------------------+--+------------+-----------+
|0 | prefix (3) | time bucket (39)     |t | machine(10)| counter(10)|
+--+------------+----------------------+--+------------+-----------+
```

- Bit `63`: reserved, always zero
- Bits `62..60`: 3-bit prefix
- Bits `59..21`: 39-bit timestamp bucket
- Bit `20`: tick bit
- Bits `19..10`: 10-bit machine ID
- Bits `9..0`: 10-bit counter

The timestamp bucket is:

```text
floor((unixMillis - epochMillis) / 4)
```

The epoch is `2025-01-01T00:00:00Z`, the timestamp resolution is `4 ms`, the machine ID range is `0..1023`, and the counter range per bucket is `0..1023`.

With a 39-bit bucket field at 4 ms resolution, `sfid` covers timestamps from `2025-01-01T00:00:00Z` through `2094-09-07T15:47:35.551Z`.

## Text layout

`sfid` uses a fixed-width text form with:

- a visible prefix alphabet: `abcdefgh`
- a lowercase base32hex payload alphabet: `0123456789abcdefghijklmnopqrstuv`

The 12-character payload uses lowercase base32hex:

```text
0123456789abcdefghijklmnopqrstuv
```

The zero ID encodes as the empty string. Every non-zero text form is 13 characters wide, uses lowercase only, and sorts lexicographically the same way the underlying 64-bit integer sorts.

Conceptually, the encoded string can be read as:

```text
p tttttttt mm cc
```

- `p`: visible prefix character
- `tttttttt`: timestamp block, where the lowest bit of that block is the tick bit
- `mm`: machine block
- `cc`: counter block

This grouping is conceptual only. The canonical string has no separators.

The empty string is reserved for the zero value.

Parsing is strict: non-zero IDs must use the canonical 13-character lowercase form.

This visual shape is useful even without decoding the full ID. When you look at a batch of IDs, the `mm` tail helps you quickly see which generator produced them, and the `cc` tail shows whether a generator is issuing more than one ID in the same 4 ms bucket. For example, if many IDs from the same `mm` end with counters other than `00`, that is a quick visual hint that this generator is running at a relatively high local rate.

### Why 13 characters

Fixed-width encodings work best with alphabets whose size is a power of two. For a 64-bit value, the practical options are:

- base16: 16 characters
- base32: 13 characters
- base64: 11 characters

`sfid` chooses base32 because 5 bits per character keeps the text short without introducing case sensitivity or punctuation. The signed-safe layout reserves the top bit, then uses a 3-bit visible prefix and a remaining 60-bit payload. That payload falls into `40 + 10 + 10` bits exactly: 40 bits for time plus tick, 10 for machine, and 10 for counter. The 60-bit payload encodes into exactly 12 base32 characters, plus 1 visible prefix character for a total width of 13.

### Why the first character is limited to `a-h`

The first character is the visible 3-bit prefix. `sfid` uses a fixed prefix alphabet:

```text
a b c d e f g h
```

The remaining 12 characters encode the 60-bit payload in lowercase base32hex.

## Prefix behavior

The 3-bit prefix exists because `sfid` reserves the sign bit so IDs fit naturally in a signed `int64`, then keeps the remaining layout base32-friendly. Time plus tick takes `40` bits, machine takes `10`, and counter takes `10`. That leaves `3` bits for a small user-defined prefix while keeping the remaining 60-bit payload aligned to 12 base32 characters.

That gives one visible leading character for lightweight application metadata without changing the width of the ID.

The prefix is configured when the generator is created:

```go
g := sfid.MustNewGenerator(42, 'd')
```

You may pass either:

- a raw nibble in the range `0..7`
- a visible lowercase prefix char in the range `a..h`

`ID.Prefix()` returns the visible lowercase prefix character in `a..h`.

`ID(0)` is a special case in the string layer: it encodes as `""`.

Example prefix meanings you might document in your own application:

- environment tags such as `a` = prod, `g` = staging, `d` = dev
- security classes such as `a` = public, `c` = classified, `h` = top-secret
- `c`..`h` = version families or shard families, depending on your application
- combine prefix and machine ID if you need a wider shard or partition space

Those are examples only. `sfid` does not assign any built-in semantics to prefixes.

## Tick bit and clock regression behavior

The tick bit exists to keep generation practical when the local clock moves backward relative to the last emitted bucket.

This part of the design is inspired by the `muyo/sno` project's tick-tock regression handling, adapted here to `sfid`'s 64-bit layout and API.

Each generator tracks:

- the last emitted bucket
- the current tick value
- the last used bucket on each tick timeline

Behavior:

1. When time moves forward, generation stays on the current tick timeline.
2. When time stays in the same bucket, the counter increments.
3. When the clock moves backward relative to the last emitted bucket, the generator flips the tick bit and attempts to continue on the alternate timeline.
4. If that alternate timeline has already used the requested bucket or a later bucket, the generator waits for a fresh safe bucket instead of reusing a prior `(bucket, tick)` combination.

This preserves uniqueness without stubbing out the regression path.

### Caveats

- `sfid` preserves uniqueness across bounded backward clock movement, but it does not promise strict monotonic ordering across arbitrary regressions.
- Under normal forward clock progression from a single generator, generated IDs are monotonic in both numeric and lexicographic order.
- If a single generator saturates all `1024` counters in one 4 ms bucket, it waits for the next bucket.

That last point means a single generator tops out at roughly `256,000` IDs per second before intentional waiting kicks in.

## Public API

Core API:

```go
type ID int64

type Generator struct { /* unexported fields */ }

func NewGenerator(machine uint16, prefix byte) (*Generator, error)
func MustNewGenerator(machine uint16, prefix byte) *Generator
func (g *Generator) New() ID
func (g *Generator) NewString() string

func Parse(s string) (ID, error)
func MustParse(s string) ID
```

`ID` methods:

```go
func (id ID) String() string
func (id ID) Prefix() byte
func (id ID) Machine() uint16
func (id ID) Counter() uint16
func (id ID) Tick() uint8
func (id ID) Time() time.Time
func (id ID) MarshalText() ([]byte, error)
func (id *ID) UnmarshalText([]byte) error
func (id ID) MarshalJSON() ([]byte, error)
func (id *ID) UnmarshalJSON([]byte) error
```

Raw conversion stays plain Go: `id := sfid.ID(v)` and `v := int64(id)`.

## Usage

### Create a generator and mint IDs

```go
package main

import (
	"fmt"

	"github.com/dpotapov/go-sfid"
)

func main() {
	g := sfid.MustNewGenerator(42, 'd')

	id := g.New()
	fmt.Println(id.String())
	fmt.Println(string(id.Prefix()))
	fmt.Println(id.Machine())
	fmt.Println(id.Counter())
	fmt.Println(id.Tick())
	fmt.Println(id.Time().UTC())
}
```

### Parse and inspect

```go
id := sfid.MustParse("d000000000000")

fmt.Println(id.Time().UTC()) // 2025-01-01 00:00:00 +0000 UTC
fmt.Println(id.Machine())    // 0
fmt.Println(id.Counter())    // 0
fmt.Println(id.Tick())       // 0
```

### JSON and text marshaling

`sfid` marshals as the canonical lowercase string:

```go
id := sfid.MustParse("d000000000000")

data, _ := json.Marshal(id)
fmt.Println(string(data))

var decoded sfid.ID
_ = json.Unmarshal(data, &decoded)
```

Output:

```text
"d000000000000"
```

The zero value marshals as `""` in both text and JSON forms.

### Typed string prefixes

`sfid.Typed[T]` adds type-safe semantics in Go code. It can also add an optional string prefix in text, JSON, and text-marshaled forms. It does not alter the underlying 64-bit ID format.

Type aliases are usually the nicest way to bind a business object type to `sfid.Typed[T]`, because the alias keeps the methods defined by `sfid.Typed[T]`:

```go
type SomeID = sfid.Typed[ObjectType]
```

Define a domain type for the business object you want to identify:

```go
type user struct{}

func (user) IDTag() string { return "u_" }

type UserID = sfid.Typed[user]
```

`user` is not part of the ID data. It is the domain type that names the typed-ID family. If that type implements `IDTag()`, the returned value becomes the exact string prefix for that family. A second family would look the same:

```go
type org struct{}

func (org) IDTag() string { return "o_" }

type OrgID = sfid.Typed[org]
```

In other words: the type parameter gives the compiler a distinct type identity, and `IDTag()` returns the exact string prefix that belongs to that family.

`sfid.Typed[T]` is a typed scalar whose underlying representation is still `sfid.ID`, so converting between the two is just a plain Go conversion such as `sfid.ID(uid)`.

If the domain type does not implement `IDTag()`, `sfid.Typed[T]` still gives you a distinct Go type, but it uses the same string form as plain `sfid.ID`.

Generate and format:

```go
g := sfid.MustNewGenerator(42, 'd')
uid := sfid.NewTyped[user](g)

fmt.Println(uid.String())
```

Parse and validate:

```go
uid, err := sfid.ParseTyped[user]("u_d30bnjf5ak010")
if err != nil {
	panic(err)
}

fmt.Println(sfid.ID(uid).String())
```

`sfid` does not add separators to typed prefixes. If you want `u_<id>`, return `u_`. If you want `user_<id>`, return `user_`.

If you do not want a string prefix at all, just omit `IDTag()`:

```go
type account struct{}

type AccountID = sfid.Typed[account]
```

Then `AccountID` keeps type-safe semantics in Go code while using the same text form as plain `sfid.ID`.

For consistency with `sfid.ID`, a zero typed ID also encodes as `""` rather than `u_`.

Single-letter typed prefixes are especially useful in logs because they produce fixed-width 15-character tokens such as `u_d30bnjf5ak010` or `o_d30bnjf5ak010`. That makes it practical for external tooling to scan rendered logs, identify typed IDs, parse them, and enrich the event with database context.

```go
uid := sfid.NewTyped[user](g)

slog.Info("operation started",
	"action", "checkout.begin",
	"actor", uid,
)
```

If a log processor sees a token like `u_d30bnjf5ak010`, it can parse it as a typed user ID, use the `u_` prefix as a type hint, and attach additional context such as the user's name or email after a lookup.

## Lowercase-only parsing

`sfid` is intentionally strict:

- output is always lowercase
- parsing expects lowercase only
- uppercase input is rejected rather than normalized

That keeps the wire format canonical and avoids silently accepting multiple textual representations of the same ID.

## Concurrency notes

- A `Generator` is safe for concurrent use by many goroutines.
- The generator coordinates access with a mutex. Correctness and testability take priority over lock-free cleverness.
- No duplicates are emitted by a correctly configured generator.
- A `(machine, prefix)` pair should be treated as a unique generator identity within a collision domain.

## Ordering behavior

- For a fixed prefix, numeric order and string order are the same.
- Across different prefixes, prefix order dominates because the prefix occupies the high 3 visible bits below the reserved sign bit.
- Under normal forward clock progression from one generator, IDs are time-major and monotonic.
- Under clock regressions, `sfid` preserves uniqueness first. Ordering across those jumps is intentionally caveated and should not be overclaimed.

## Benchmarks

Benchmarks live in the repository and can be reproduced with:

```bash
go test -bench=. -benchmem ./...
(cd benchmarks && go test -bench=. -benchmem ./...)
```

The root module contains the `sfid` benchmarks only. Cross-library comparison benchmarks live under [`benchmarks/`](benchmarks/) so the main library does not depend on benchmark-only packages.

Measured locally on March 25, 2026 with Go `1.26.1` on `darwin/arm64` (`Apple M3 Pro`).

### Summary

| Workload | `sfid` | `sno` | `xid` | snowflake | sonyflake | `ulid` | `uuid` |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Generate | `3896 ns/op`, `0 allocs` | `60.95 ns/op`, `0 allocs` | `36.43 ns/op`, `0 allocs` | `244.2 ns/op`, `0 allocs` | `39084 ns/op`, `0 allocs` | `43.58 ns/op`, `1 alloc` | `229.9 ns/op`, `1 alloc` |
| String | `6.844 ns/op`, `0 allocs` | `5.013 ns/op`, `0 allocs` | `7.519 ns/op`, `0 allocs` | `25.79 ns/op`, `1 alloc` | `21.61 ns/op`, `1 alloc` | `10.34 ns/op`, `0 allocs` | `27.56 ns/op`, `1 alloc` |
| Parse | `6.467 ns/op`, `0 allocs` | `6.361 ns/op`, `0 allocs` | `12.05 ns/op`, `0 allocs` | `26.87 ns/op`, `0 allocs` | `22.81 ns/op`, `0 allocs` | `13.44 ns/op`, `0 allocs` | `18.18 ns/op`, `0 allocs` |

The generation result for `sfid` reflects its deliberate throughput ceiling on a single generator: once one 4 ms bucket consumes all `1024` counters, generation waits for the next bucket. The string and parse paths stay allocation-free and very fast.

The comparison suite in [`benchmarks/`](benchmarks/) contains the snowflake, sonyflake, `sno`, `xid`, `ulid`, and `uuid` measurements shown above.

## References and attributions

- [Snowflake ID](https://en.wikipedia.org/wiki/Snowflake_ID): conceptual family and historical background
- [`muyo/sno`](https://github.com/muyo/sno): inspiration for the tick-tock regression idea and benchmark style
- [`rs/xid`](https://github.com/rs/xid): inspiration for API ergonomics and practical developer experience

`sfid` borrows ideas, not code. It is a separate 64-bit format with its own layout, encoding, constraints, and tradeoffs.

## License

MIT. See [LICENSE](LICENSE).
