package defaults

import (
	"fmt"
	"time"
)

func Example() {
	{
		var a int
		fmt.Println(IntIfZero(a, 1))
	}
	{
		var a string
		fmt.Println(StringIfEmpty(a, "hello"))
	}
	{
		var a float64 = 0.1
		fmt.Println(Float64IfNotMatch(a, 0.8, "N>=0.2&&N<0.9"))
	}
	{
		now := time.Now().UTC()
		deflt := time.Date(2000, time.January, 1, 1, 1, 1, 0, time.UTC)
		fmt.Println(TimeIfNotMatchFunc(now, deflt, func(t time.Time) bool { return t.Before(deflt) }))
	}

	// Output:
	// 1
	// hello
	// 0.8
	// 2000-01-01 01:01:01 +0000 UTC
}
