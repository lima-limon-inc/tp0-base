package common

import (
	// "bufio"
	// "fmt"
	"net"

	"encoding/binary"
	// "time"
	// "os"
	// "strconv"

	// "github.com/op/go-logging"

	// "github.com/pkg/errors"
)

type Values uint8


// Using the Value* prefix to avoid collissions
const (
	ValueString Values = iota
	ValueInteger
)

// File that contains type serialization

// Each supported type in the Protocol supports the following schema:
// Value type: 1 byte
// Value length: 1 byte
// Value

func SendString(s string) []byte {
	bytes := []byte(s)
	length := len(bytes)

	buffer_len := length + 2
	buffer := make([]byte, buffer_len)

	// 2 additional bytes
	buffer[0] = byte(ValueString)
	buffer[1] = byte(length)

	for i := 0; i < length; i++ {
		current_byte := bytes[i]
		buffer[i + 2] = current_byte
	}

	return buffer
}

func SendInteger(i uint64) []byte {
	length := 8

	buffer_len := length + 2

	buffer := make([]byte, buffer_len)

	// 2 additional bytes
	buffer[0] = byte(ValueString)
	buffer[1] = byte(length)

	binary.BigEndian.PutUint64(buffer[2:buffer_len], i)

	return buffer
}
