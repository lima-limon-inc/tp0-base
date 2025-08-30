# Received like so:
# 1 byte for length
# N bytes for data
def DeserializeString(string: bytes) -> str:
    length = string[0]
    bytes_string = string[1:length]

    inner_string = bytes_string.decode()

    return inner_string

# Received like so:
# 1 byte for length
# N bytes for data
def DeserializeUInteger64(string: bytes) -> int:
    length = string[0]
    bytes_int = string[1:length]

    inner_int = int.from_bytes(bytes_int, byteorder='big', signed=True)

    return inner_int
