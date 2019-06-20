// Package defaults provides a list of functions to
// eval an alternative default value by conditions.
//
// The patterns for numbers just like normal if conditions,
// such as "N>=20&&N<90", N is the placholder for actual value,
// NOTE: the bit operation is not supported.
//
// The pattern for string uses "regexp" package.
//
// NOTE: if the pattern is invalid, the related function will
// PANIC directly.
package defaults
