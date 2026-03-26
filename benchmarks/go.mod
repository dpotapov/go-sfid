module github.com/dpotapov/go-sfid/benchmarks

go 1.25

require (
	github.com/bwmarrin/snowflake v0.3.0
	github.com/dpotapov/go-sfid v0.0.0
	github.com/google/uuid v1.6.0
	github.com/muyo/sno v1.2.1
	github.com/oklog/ulid/v2 v2.1.1
	github.com/rs/xid v1.6.0
	github.com/sony/sonyflake v1.3.0
)

replace github.com/dpotapov/go-sfid => ..
