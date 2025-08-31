# Received like so:
# 1 byte for length
# N bytes for data
def DeserializeString(bytes_string: bytes) -> str:
    inner_string = bytes_string.decode()

    return inner_string

def DeserializeUInteger64(bytes_integer: bytes) -> int:
    inner_int = int.from_bytes(bytes_integer, byteorder='big', signed=True)

    return inner_int

