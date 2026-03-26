package sfid_test

import (
	"encoding/json"
	"fmt"

	"github.com/dpotapov/go-sfid"
)

func ExampleNewGenerator() {
	g := sfid.MustNewGenerator(42, 'd')
	id := g.New()

	fmt.Println(len(id.String()))
	fmt.Println(string(id.Prefix()))
	fmt.Println(id.Machine())
	fmt.Println(id.Counter())
	fmt.Println(id.Tick())

	// Output:
	// 13
	// d
	// 42
	// 0
	// 0
}

func ExampleParse() {
	id := sfid.MustParse("d000000000000")

	fmt.Println(id.Time().UTC().Format("2006-01-02T15:04:05.000Z07:00"))
	fmt.Println(string(id.Prefix()))
	fmt.Println(id.Machine())
	fmt.Println(id.Counter())
	fmt.Println(id.Tick())

	// Output:
	// 2025-01-01T00:00:00.000Z
	// d
	// 0
	// 0
	// 0
}

func ExampleID_MarshalJSON() {
	id := sfid.MustParse("d000000000000")
	data, _ := json.Marshal(id)
	fmt.Println(string(data))

	// Output:
	// "d000000000000"
}

type user struct{}

func (user) IDTag() string { return "u_" }

type account struct{}

type UserID = sfid.Typed[user]
type AccountID = sfid.Typed[account]

func ExampleTyped() {
	g := sfid.MustNewGenerator(42, 'd')
	userID := sfid.NewTyped[user](g)

	fmt.Println(len(userID.String()))
	fmt.Println(userID.String()[0:2])

	parsed, _ := sfid.ParseTyped[user](userID.String())
	fmt.Println(sfid.ID(parsed) == sfid.ID(userID))

	// Output:
	// 15
	// u_
	// true
}

func ExampleTyped_withoutTag() {
	g := sfid.MustNewGenerator(42, 'd')
	accountID := sfid.NewTyped[account](g)

	fmt.Println(len(accountID.String()))

	parsed, _ := sfid.ParseTyped[account](accountID.String())
	fmt.Println(sfid.ID(parsed) == sfid.ID(accountID))

	// Output:
	// 13
	// true
}
