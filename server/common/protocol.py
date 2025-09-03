import os

# Received like so:
# 1 byte for length
# N bytes for data
def DeserializeString(bytes_string: bytes) -> str:
    inner_string = bytes_string.decode()

    return inner_string

def SerializeString(string: str) -> bytes:
    str_bytes = string.encode('utf-8')

    str_indicator = b'0'
    length = len(str_bytes).to_bytes(1, byteorder='big')
    header = str_indicator + length

    package = header + str_bytes

    return package

def DeserializeUInteger64(bytes_integer: bytes) -> tuple[int, bytes]:
    integer_indicator = bytes_integer[0:1]
    integer_indicator_i = int.from_bytes(integer_indicator, byteorder='big', signed=True)
    if integer_indicator_i != 1:
        print(f"Tried to deserialize as uint64, non uint64 type {integer_indicator_i}")
        os.abort()

    integer_len = bytes_integer[1:2]
    integer_len_i = int.from_bytes(integer_len, byteorder='big', signed=True)
    if integer_len_i != 8:
        print(f"Tried to deserialize as uint64 that's longer than 8 bytes: {integer_len_i}")
        os.abort()

    inner_int = int.from_bytes(bytes_integer[2: 2 + integer_len_i], byteorder='big', signed=True)

    remaining_bytes = bytes_integer[integer_len_i + 2:]

    return inner_int, remaining_bytes

def SerializeUInteger64(integer: int) -> bytes:
    integer_bytes = integer.to_bytes(8, byteorder='big')

    int_indicator_i = 1
    int_indicator = int_indicator_i.to_bytes(1, byteorder='big')

    length = len(integer_bytes).to_bytes(1, byteorder='big')
    header = int_indicator + length

    package = header + integer_bytes

    return package

def DeserializeUInteger8(bytes_integer: bytes) -> int:
    inner_int = int.from_bytes(bytes_integer, byteorder='big', signed=True)

    return inner_int
