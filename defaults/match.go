package defaults

import (
	"regexp"
	"time"

	"github.com/damnever/goctl/defaults/ebool"
)

// IntIfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func IntIfNotMatch(actual, deflt int, pattern string) int {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Int8IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Int8IfNotMatch(actual, deflt int8, pattern string) int8 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Int16IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Int16IfNotMatch(actual, deflt int16, pattern string) int16 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Int32IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Int32IfNotMatch(actual, deflt int32, pattern string) int32 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Int64IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Int64IfNotMatch(actual, deflt int64, pattern string) int64 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// UintIfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func UintIfNotMatch(actual, deflt uint, pattern string) uint {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Uint8IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Uint8IfNotMatch(actual, deflt uint8, pattern string) uint8 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Uint16IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Uint16IfNotMatch(actual, deflt uint16, pattern string) uint16 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Uint32IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Uint32IfNotMatch(actual, deflt uint32, pattern string) uint32 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Uint64IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Uint64IfNotMatch(actual, deflt uint64, pattern string) uint64 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Float32IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Float32IfNotMatch(actual, deflt float32, pattern string) float32 {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Float64IfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func Float64IfNotMatch(actual, deflt float64, pattern string) float64 {
	if evalBool(pattern, actual) {
		return actual
	}
	return deflt
}

// DurationIfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func DurationIfNotMatch(actual, deflt time.Duration, pattern string) time.Duration {
	if evalBool(pattern, float64(actual)) {
		return actual
	}
	return deflt
}

// Int16IfNotMatch returns the value deflt if actual isn't match with regexp pattern,
// otherwise returns actual.
// NOTE it will panic directly if the pattern is invalid.
func StringIfNotMatch(actual, deflt, pattern string) string {
	if regexp.MustCompile(pattern).MatchString(actual) {
		return actual
	}
	return deflt
}

// TimeIfNotMatch returns the value deflt if actual isn't match with pattern,
// otherwise returns actual.
// NOTE it is not implemented.
func TimeIfNotMatch(actual, deflt time.Time, pattern string) time.Time {
	panic("not implemented")
}

func evalBool(pattern string, arg float64) bool {
	e, err := ebool.New(pattern)
	if err != nil {
		panic(err)
	}
	ok, err := e.Eval(arg)
	if err != nil {
		panic(err)
	}
	return ok
}

// IntIfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func IntIfNotMatchFunc(actual, deflt int, validator func(int) bool) int {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Int8IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Int8IfNotMatchFunc(actual, deflt int8, validator func(int8) bool) int8 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Int16IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Int16IfNotMatchFunc(actual, deflt int16, validator func(int16) bool) int16 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Int32IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Int32IfNotMatchFunc(actual, deflt int32, validator func(int32) bool) int32 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Int64IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Int64IfNotMatchFunc(actual, deflt int64, validator func(int64) bool) int64 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// UintIfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func UintIfNotMatchFunc(actual, deflt uint, validator func(uint) bool) uint {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Uint8IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Uint8IfNotMatchFunc(actual, deflt uint8, validator func(uint8) bool) uint8 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Uint16IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Uint16IfNotMatchFunc(actual, deflt uint16, validator func(uint16) bool) uint16 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Uint32IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Uint32IfNotMatchFunc(actual, deflt uint32, validator func(uint32) bool) uint32 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Uint64IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Uint64IfNotMatchFunc(actual, deflt uint64, validator func(uint64) bool) uint64 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Float32IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Float32IfNotMatchFunc(actual, deflt float32, validator func(float32) bool) float32 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// Float64IfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func Float64IfNotMatchFunc(actual, deflt float64, validator func(float64) bool) float64 {
	if validator(actual) {
		return actual
	}
	return deflt
}

// DurationIfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func DurationIfNotMatchFunc(actual, deflt time.Duration, validator func(time.Duration) bool) time.Duration {
	if validator(actual) {
		return actual
	}
	return deflt
}

// StringIfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func StringIfNotMatchFunc(actual, deflt string, validator func(string) bool) string {
	if validator(actual) {
		return actual
	}
	return deflt
}

// TimeIfNotMatchFunc returns the value deflt if validator(actual) returns false,
// otherwise returns actual.
func TimeIfNotMatchFunc(actual, deflt time.Time, validator func(time.Time) bool) time.Time {
	if validator(actual) {
		return actual
	}
	return deflt
}
