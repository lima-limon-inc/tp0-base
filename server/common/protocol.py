# Received like so:
# 1 byte for length
# N bytes for data
def DeserializeString(bytes_string: bytes) -> str:
    inner_string = bytes_string.decode()

    return inner_string

def SerializeString(string: str) -> bytes:
    str_bytes = string.encode('utf-8')

    str_indicator = b'0'
    length = len(str_bytes).to_bytes()
    header = str_indicator + length

    package = header + str_bytes

    return package

def DeserializeUInteger64(bytes_integer: bytes) -> int:
    inner_int = int.from_bytes(bytes_integer, byteorder='big', signed=True)

    return inner_int

def SerializeUInteger64(integer: int) -> bytes:
    integer_bytes = integer.to_bytes(8, byteorder='big')

    int_indicator = b'1'
    length = len(integer_bytes).to_bytes()
    header = int_indicator + length

    package = header + integer_bytes

    return package

def DeserializeUInteger8(bytes_integer: bytes) -> int:
    inner_int = int.from_bytes(bytes_integer, byteorder='big', signed=True)

    return inner_int
