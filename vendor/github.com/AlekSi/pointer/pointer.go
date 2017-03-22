// Package pointer provides helpers to get pointers to values of build-in types.
package pointer // import "github.com/AlekSi/pointer"

import (
	"time"
)

func ToBool(b bool) *bool                   { return &b }
func ToByte(b byte) *byte                   { return &b }
func ToComplex128(c complex128) *complex128 { return &c }
func ToComplex64(c complex64) *complex64    { return &c }
func ToError(e error) *error                { return &e }
func ToFloat32(f float32) *float32          { return &f }
func ToFloat64(f float64) *float64          { return &f }
func ToInt(i int) *int                      { return &i }
func ToInt16(i int16) *int16                { return &i }
func ToInt32(i int32) *int32                { return &i }
func ToInt64(i int64) *int64                { return &i }
func ToInt8(i int8) *int8                   { return &i }
func ToRune(r rune) *rune                   { return &r }
func ToString(s string) *string             { return &s }
func ToTime(t time.Time) *time.Time         { return &t }
func ToUint(u uint) *uint                   { return &u }
func ToUint16(u uint16) *uint16             { return &u }
func ToUint32(u uint32) *uint32             { return &u }
func ToUint64(u uint64) *uint64             { return &u }
func ToUint8(u uint8) *uint8                { return &u }
func ToUintptr(u uintptr) *uintptr          { return &u }
