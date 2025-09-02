import socket
import logging
from . utils import Bet, store_bets, load_bets, has_won
from . import protocol


class Server:
    def __init__(self, port, listen_backlog, expected_clients: int):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

        self._expected_clients = expected_clients

        # self._current_clients: list[socket.socket] = []
        self._current_client = 0
        self._client_by_agente = {}

        self._killed = False

    def finalize(self):
        self._server_socket.close()
        for client in self._client_by_agente:
            client.close()
        self._killed = True

    def _receive_clients(self):
        while self._current_client < self._expected_clients:
            self.__accept_new_connection()
            self.__handle_client_connection()
            self._current_client += 1

    def _handle_lottery(self):
        bets = load_bets()
        winners = []
        for bet in bets:
            if has_won(bet):
                winners.append(bet)

        winners_packages = self._serialize_winners(winners)
        self._send_winners(winners_packages)

        self._current_client = 0

    def _serialize_winners(self, winners: list) -> dict:
        winners_serialized_by_agency = {}
        for winner in winners:
            if winners_serialized_by_agency.get(winner.agency) == None:
                winners_serialized_by_agency[winner.agency] = []
            winners_serialized_by_agency[winner.agency].append(self.__serialize_bet(winner))
        # TODO: Serialize bets

        packages_by_agency = {}
        for agency, _ in winners_serialized_by_agency.items():
            packages_by_agency[agency] = []

        header_i = 0
        header = header_i.to_bytes(1, byteorder='big')

        # for agency, winners in winners_serialized_by_agency.items():
        for agency in self._client_by_agente.keys():
            total_size = 0
            data_part = b''
            if winners_serialized_by_agency.get(agency) != None:
                winners = winners_serialized_by_agency[agency]
                for winner in winners:
                    total_size += len(winner)
                    data_part += winner

            full_package = header + protocol.SerializeUInteger64(total_size) + data_part

            packages_by_agency[agency] = full_package

        return packages_by_agency

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
                self._receive_clients()
                self._handle_lottery()
            except OSError as e:
                # If we catch an error, then most probably we received a signal that closed our sockets
                break

    # Size: Amount of bytes to read
    def __receive_bytes(self, size, client_index) -> bytes:
        buff: bytes  = b''
        remaining_size = size
        while remaining_size > 0:
            # Received bytes
            received = self._client_by_agente[client_index].recv(size)
            buff = buff + received
            remaining_size -= len(received)

        return buff

    # Size: Amount of bytes to read
    def __send_bytes(self, data, client_index):
        size = len(data)
        remaining_size = size
        while remaining_size > 0:
            # Received bytes
            sent_data = self._client_by_agente[client_index].send(data)
            remaining_size -= sent_data


    def __handle_client_connection(self):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        while True:
            try:
                # TODO: Modify the receive to avoid short-reads

                # We start of reading two bytes to check how much we should read
                # Primer byte indicador de apuestas
                # Siguiente es un integer empaquetado:
                # # 1 byte indicador
                initial_type = self.__receive_bytes(1, self._current_client)
                initial_indicator = protocol.DeserializeUInteger8(initial_type)
                if initial_indicator == 2:
                    logging.info(f'action: apuesta_finalizadas | result: success | status: finished ')
                    break

                # # 1 byte longitud
                # # 8 bytes datos
                # 1 + 1 + 1 + 8 = 11
                initial_size = self.__receive_bytes(10, self._current_client)

                size = protocol.DeserializeUInteger64(initial_size[3:11])

                # Now, we read all that data
                bets_batch_bytes = self.__receive_bytes(size, self._current_client)
                try:
                    bets = self.__deserialize_batches(bets_batch_bytes)
                    store_bets(bets)
                    logging.info(f'action: apuesta_recibida | result: success | cantidad: {len(bets)}')
                    ok = bytes(1)
                    self.__send_bytes(ok, self._current_client)
                except:
                    logging.info(f'action: apuesta_recibida | result: fail')
                    ok = bytes(2)
                    self.__send_bytes(ok, self._current_client)

            except OSError as e:
                logging.error("action: receive_message | result: fail | error: {e}")


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

        client_id_byte = self.__receive_bytes(2, self._current_client)
        length_id = int(client_id_byte[1:2])

        client_id = self.__receive_bytes(length_id, self._current_client)
        client_id = int(protocol.DeserializeString(client_id))

        self._client_by_agente[client_id] = c

        return

    # Bet
    # Size
    # Datos
    def __deserialize_batches(self, bet_batches: bytes) -> list[Bet]:
        bets = []
        batch_len = len(bet_batches)
        current_byte = 0

        while current_byte < batch_len:
            bet_indicator = bet_batches[current_byte:current_byte + 1]

            size_b = bet_batches[current_byte + 1:current_byte + 2]
            size_i = int.from_bytes(size_b, byteorder='big', signed=True) # 1

            current_bet = self.__deserialize_bet(bet_batches[current_byte:current_byte + size_i + 2])
            bets.append(current_bet)
            current_byte += size_i + 2

        return bets

    def __deserialize_bet(self, serialized_bet: bytes) -> Bet :
        rest = serialized_bet

        bet_indicator = rest[0:1]
        bet_size_b = rest[1:2]
        bet_size = int.from_bytes(bet_size_b, byteorder='big', signed=True) # 1
        rest = rest[2:]

        string_type = rest[0:1] # 0
        name_len = rest[1:2]    # 1
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True) # 1
        agency = protocol.DeserializeString(rest[2: 2 + name_len_i])
        # if int(agency) not in self._client_by_agente:
        #     self._client_by_agente[int(agency)] = self._current_clients[self._current_client]

        rest = rest[name_len_i + 2:]

        string_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        first_name = protocol.DeserializeString(rest[2: 2 + name_len_i ])
        rest = rest[name_len_i + 2:]

        string_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        last_name = protocol.DeserializeString(rest[2: 2 + name_len_i ])
        rest = rest[name_len_i + 2:]

        string_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        document = protocol.DeserializeString(rest[2: 2 + name_len_i ])
        rest = rest[name_len_i + 2:]

        string_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        birthday = protocol.DeserializeString(rest[2: 2 + name_len_i ])
        rest = rest[name_len_i + 2:]

        integer_type = rest[0:1]
        name_len = rest[1:2]
        name_len_i = int.from_bytes(name_len, byteorder='big', signed=True)
        amount = protocol.DeserializeUInteger64(rest[2: 2 + name_len_i ])
        rest = rest[name_len_i + 2:]

        return Bet(agency, first_name, last_name, document, birthday, amount)


    def __serialize_bet(self, bet: Bet) -> bytes:
        bet_id = protocol.SerializeString(str(bet.agency))
        bet_name = protocol.SerializeString(str(bet.first_name))
        bet_surname = protocol.SerializeString(str(bet.last_name))
        bet_document = protocol.SerializeString(str(bet.document))
        bet_birthday = protocol.SerializeString(str(bet.birthdate))
        bet_amount = protocol.SerializeUInteger64(bet.number)

        bet_data = bet_id + bet_name + bet_surname + bet_document + bet_birthday + bet_amount

        bet_identifier_i = 0
        bet_identifier = bet_identifier_i.to_bytes(1, byteorder='big')
        length_i = len(bet_data)
        length = length_i.to_bytes(1, byteorder='big')
        header = bet_identifier + length

        package = header + bet_data

        return package

    def _send_winners(self, winners: dict):
        for agency, package in winners.items():
            # self.__send_bytes(package, agency - 1)
            self._client_by_agente[agency].send(package)
