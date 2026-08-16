//go:build linux

// Package common holds shared helpers for guest backend implementations
// and the parent guest facade.
//
// Stages and blocks must use the Guest handle for disk I/O — they must not
// import this package for guest filesystem access.
package common
