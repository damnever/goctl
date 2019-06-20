package defaults

import "time"

// IntIfZero returns the value deflt if actual is zero, otherwise returns actual.
func IntIfZero(actual, deflt int) int {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Int8IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Int8IfZero(actual, deflt int8) int8 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Int16IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Int16IfZero(actual, deflt int16) int16 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Int32IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Int32IfZero(actual, deflt int32) int32 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Int64IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Int64IfZero(actual, deflt int64) int64 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// UintIfZero returns the value deflt if actual is zero, otherwise returns actual.
func UintIfZero(actual, deflt uint) uint {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Uint8IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Uint8IfZero(actual, deflt uint8) uint8 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Uint16IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Uint16IfZero(actual, deflt uint16) uint16 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Uint32IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Uint32IfZero(actual, deflt uint32) uint32 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Uint64IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Uint64IfZero(actual, deflt uint64) uint64 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Float32IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Float32IfZero(actual, deflt float32) float32 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// Float64IfZero returns the value deflt if actual is zero, otherwise returns actual.
func Float64IfZero(actual, deflt float64) float64 {
	if actual == 0 {
		return deflt
	}
	return actual
}

// TimeIfZero returns the value deflt if actual is empty, otherwise returns actual.
func StringIfEmpty(actual, deflt string) string {
	if actual == "" {
		return deflt
	}
	return actual
}

// DurationIfZero returns the value deflt if actual is zero, otherwise returns actual.
func DurationIfZero(actual, deflt time.Duration) time.Duration {
	if actual == 0 {
		return deflt
	}
	return actual
}

// TimeIfZero returns the value deflt if actual is zero, otherwise returns actual.
func TimeIfZero(actual, deflt time.Time) time.Time {
	if actual.IsZero() {
		return deflt
	}
	return actual
}
