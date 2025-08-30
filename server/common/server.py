import socket
import logging
from . utils import Bet, store_bets
from . import protocol


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

        self._current_client: socket.socket = None

        self._killed = False

    def finalize(self):
        self._server_socket.close()
        if self._current_client != None:
            self._current_client.close()
        self._killed = True

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        # TODO: Modify this program to handle signal to graceful shutdown
        # the server
        # Despite only looping while this is alive, a signal could come at any time.
        # With that in mind, we must catch any potential OSError in here as well
        while not self._killed:
            try:
                self._current_client = self.__accept_new_connection()
                self.__handle_client_connection()
            except OSError as e:
                # If we catch an error, then most probably we received a signal that closed our sockets
                break

    # Size: Amount of bytes to read
    def __receive_bytes(self, size) -> bytes:
        buff: bytes  = b''
        remaining_size = size
        while remaining_size > 0:
            # Received bytes
            received = self._current_client.recv(size)
            buff = buff + received
            remaining_size -= len(received)

        return buff

    def __handle_client_connection(self):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads

            # We start of reading two bytes to check how much we should read
            initial_size = self.__receive_bytes(2)

            size = initial_size[1:2]
            size_i = int.from_bytes(size, byteorder='big', signed=True)

            # Now, we read all that data
            bet_bytes = self.__receive_bytes(size_i - 2)
            bet = self.__deserialize(bet_bytes)
            store_bets([bet])
            dni = bet.document
            number = bet.number
            logging.info(f'action: apuesta_almacenada | result: success | dni: {dni} | numero: {number}')


            # msg = self._current_client.recv(1024).rstrip().decode('utf-8')
            addr = self._current_client.getpeername()
            # logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {msg}')
            # TODO: Modify the send to avoid short-writes
            entire_message = initial_size + bet_bytes
            self._current_client.send("{}\n".format(entire_message).encode('utf-8'))
        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            self._current_client.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    def __deserialize(self, serialized_bet: bytes) -> Bet :
        rest = serialized_bet

        agency_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        agency = protocol.DeserializeString(rest[1:name_len_i])
        rest = rest[name_len_i:]

        string_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        first_name = protocol.DeserializeString(rest[1:name_len_i])
        rest = rest[name_len_i:]

        string_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        last_name = protocol.DeserializeString(rest[1:name_len_i])
        rest = rest[name_len_i:]

        string_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        document = protocol.DeserializeString(rest[1:name_len_i])
        rest = rest[name_len_i:]

        string_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        birthday = protocol.DeserializeString(rest[1:name_len_i])
        rest = rest[name_len_i:]

        integer_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        amount = protocol.DeserializeUInteger64(rest[1:name_len_i])
        rest = rest[name_len_i:]

        return Bet(agency, first_name, last_name, document, birthday, amount)

