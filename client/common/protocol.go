package common

import (

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
	ValueString Values = 0
	ValueUInteger64 Values = 1
	ValueByte Values = 2
)

// File that contains type serialization

// Each supported type in the Protocol supports the following schema:
// Value type: 1 byte
// Value length: 1 byte
// Value
func SerializeString(s string) []byte {
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

func DeserializeString(data []byte) string {
	received_string := string(data)

	return received_string
}

func DeserializeUInteger64(data []byte) uint64 {
	integer := binary.BigEndian.Uint64(data)

	return integer
}
func SerializeUInteger64(i uint64) []byte {
	length := 8

	buffer_len := length + 2

	buffer := make([]byte, buffer_len)

	// 2 additional bytes
	buffer[0] = byte(ValueUInteger64)
	buffer[1] = byte(length)

	binary.BigEndian.PutUint64(buffer[2:buffer_len], i)

	return buffer
}

func SerializeByte(b byte) []byte {
	length := 1

	buffer_len := length + 2

	buffer := make([]byte, buffer_len)

	// 2 additional bytes
	buffer[0] = byte(ValueByte)
	buffer[1] = byte(length)
	buffer[2] = b
	return buffer
}
