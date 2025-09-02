package common

import (
	"io"

	"fmt"
	"net"
	"time"
	"os"
	"strconv"

	"github.com/op/go-logging"

	"github.com/pkg/errors"

	"encoding/csv"
)

var log = logging.MustGetLogger("log")

const (
	// Max batch size is 8 KILO bytes aka 8 thousand bytes
	MAX_BATCH_SIZE int  = 8000
)

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID                     string
	ServerAddress          string
	LoopAmount             int
	LoopPeriod             time.Duration
	MaxBetAmountInBatch    int
}

type Bet struct {
	name          string
	surname       string
	document      string
	birthday      string
	amount        uint64
}

type ClientValues uint8

// Using the Client* prefix to avoid collissions
const (
	ClientBet ClientValues = 0
	ClientBetBatch ClientValues = 1
	ClientBetEnd ClientValues = 2
)

// A Bet is serialized as the following:
// A byte indicating that it's a bet
// Its length
// All the rest of the elements serialized
func (b *Bet) serialize(ID string) []byte {
	id := SerializeString(ID)
	name := SerializeString(b.name)
	surname := SerializeString(b.surname)
	document := SerializeString(b.document)
	birthday := SerializeString(b.birthday)
	amount := SerializeUInteger64(b.amount)

	fields := append(id, name...)
	fields = append(fields, surname...)
	fields = append(fields, document...)
	fields = append(fields, birthday...)
	fields = append(fields, amount...)

	length := len(fields)

	buffer_len := length + 2
	buffer := make([]byte, buffer_len)

	buffer[0] = byte(ClientBet)
	buffer[1] = byte(length)
	for i := 0; i < length; i++ {
		current_byte := fields[i]
		buffer[i + 2] = current_byte
	}

	return buffer
}

func deserialize_bets(bets_b []byte, length int) []*Bet {
	var bets []*Bet;
	offset := 0

	for ; offset < length ; {
		bet_indicator := bets_b[offset]
		if bet_indicator != byte(ClientBet) {
			panic("Tried to deserialize as bet something that isnt a bet")
		}
		bet_size_b := bets_b[offset + 1]
		bet_size := int(uint8(bet_size_b))

		bet := deserialize(bets_b[offset:offset + bet_size])

		bets = append(bets, bet)

		offset += bet_size + 2
	}

	return bets
}

func deserialize(data []byte) *Bet {

	// String indicator
	string_identifier := 2
	id_size_pos := string_identifier + 1
	id_size_b := data[id_size_pos]
	id_size := int(id_size_b)
	// id_end := id_size_pos + 1 + id_size - 2
	// id := DeserializeString(data[id_size_pos + 1:id_end])

	// String indicator
	name_indicator_pos := string_identifier + id_size + 2
	name_size_pos := name_indicator_pos + 1
	name_size_b := data[name_size_pos]
	name_size := int(name_size_b)
	name_end := name_size_pos + 1 + name_size
	name := DeserializeString(data[name_size_pos + 1:name_end])

	// String indicator
	surname_indicator := name_indicator_pos + name_size + 2
	surname_size_pos := surname_indicator + 1
	surname_size_b := data[surname_size_pos]
	surname_size := int(surname_size_b)
	surname_end := surname_size_pos + 1 + surname_size
	surname := DeserializeString(data[surname_size_pos + 1:surname_end])

	// String indicator
	document_indicator := surname_indicator + surname_size + 2
	document_size_pos := document_indicator + 1
	document_size_b := data[document_size_pos]
	document_size := int(document_size_b)
	document_end := document_size_pos + 1 + document_size
	document := DeserializeString(data[document_size_pos + 1:document_end])

	// String indicator
	birthday_indicator := document_indicator + document_size + 2
	birthday_size_pos := birthday_indicator + 1
	birthday_size_b := data[birthday_size_pos]
	birthday_size := int(birthday_size_b)
	birthday_end := birthday_size_pos + 1 + birthday_size
	birthday := DeserializeString(data[birthday_size_pos + 1:birthday_end])

	// Integer indicator
	amount_indicator := birthday_indicator + birthday_size + 2
	amount_size_pos := amount_indicator + 1
	amount_size_b := data[amount_size_pos]
	amount_size := int(amount_size_b)
	amount_pos := amount_size_pos + 1
	amount_end := amount_pos + amount_size
	amount := DeserializeUInteger64(data[amount_indicator:amount_end])

	bet := &Bet {
			name: name,
			surname: surname,
			document: document,
			birthday: birthday,
			amount: amount,
	}

	return bet
}


// I avoided bet for simplicity's sake and because the environment variables are
// "guaranteed" to be passed.
func InitBet() (*Bet, error) {
	var all_exists bool = true
	name, name_exists := os.LookupEnv("NOMBRE")
	all_exists = all_exists && name_exists
	surname, surname_exists := os.LookupEnv("APELLIDO")
	all_exists = all_exists && surname_exists
	document, document_exists := os.LookupEnv("DOCUMENTO")
	all_exists = all_exists && document_exists
	birthday, birthday_exists := os.LookupEnv("NACIMIENTO")
	all_exists = all_exists && birthday_exists
	amount_str, amount_str_exists := os.LookupEnv("NUMERO")
	all_exists = all_exists && amount_str_exists

	// TODO: Display which variable was not defined
	if !all_exists {
		return nil, errors.New("Not all variables defined")
	}

	amount, err := strconv.Atoi(amount_str)
	if err != nil {
		// TODO: Display number in err message
		return nil, errors.Wrapf(err, "Failed to parse amount to int")
	}


	bet := &Bet {
		name: name,
		surname: surname ,
		document: document,
		birthday: birthday,
		amount: uint64(amount),
	}

	return bet, err
}

// Client Entity that encapsulates how
type Client struct {
	config      ClientConfig

	agencyReader  csv.Reader
	agencyFile    os.File

	conn        net.Conn
	killed      bool

	bet         Bet
}

// Stop the client before hand.
func (c *Client) Close() {
     socket_err :=  c.conn.Close()
	if socket_err == nil {
	   log.Debug("action: close_socket | result: success | client_id: %v", c.config.ID);
     } else {
	   log.Debug("action: close_socket | result: failure | client_id: %v", c.config.ID);
	}

	c.agencyFile.Close()

	c.killed = true
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) (*Client, error) {
	data_path := ".data/dataset/agency-" + config.ID + ".csv"
	agency_file, err := os.Open(data_path)
	if err != nil {
		return nil, err
	}
	agency_reader := csv.NewReader(agency_file);

	client := &Client{
		config: config,
		killed: false,
		agencyFile: *agency_file,
		agencyReader: *agency_reader,
	}
	return client, nil
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

func (c *Client) sendToServer(data []byte) error {
	length := len(data)


	var sent = 0
	var err error
	for offset := 0 ; offset < length ; offset += sent {
		sent, err = c.conn.Write(data[offset:])
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) endCommunication() {
}

func (c *Client) receiveWinners() ([]*Bet, error) {
	// 1 for indicator + 10 for uint64
	header, err := c.receiveMessage(1 + 10)
	if err != nil {
		return nil, err
	}
	length_b := header[1:11]
	length := DeserializeUInteger64(length_b)

	bets_b, err := c.receiveMessage(int(length))

	bets := deserialize_bets(bets_b, int(length))

	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v",
		len(bets),
	)

	return bets, nil
}

// Will send a batch of complete bets to the server
// The amount of bets will depend on the following equation:
// min(max_amount, 8kb of data)
// Initial bet is used when a bet was succesfully serialized, but was unable to fit in the previous batch
func (c *Client) createBatch(initial_bet *Bet) ([]byte, *Bet, error, bool) {
	buffer := make([]byte, MAX_BATCH_SIZE)

	var offset = 0
	var left_out_bet *Bet = nil
	var file_has_lines = true;

	if initial_bet != nil {
		serialized_bet := initial_bet.serialize(c.config.ID)
		copy(buffer, serialized_bet)
		offset += len(serialized_bet)
	}

	max_batches := c.config.MaxBetAmountInBatch;
	for current_batch := 0 ; current_batch < max_batches && file_has_lines == true; current_batch += 1 {
		record, err := c.agencyReader.Read()
		if err != nil {
			if err == io.EOF {
				file_has_lines = false
				break;
			} else {
				return nil, nil, err, true
			}
		}

		name          := record[0]
		surname       := record[1]
		document      := record[2]
		birthday      := record[3]
		amount, err   := strconv.ParseUint(record[4], 10, 64)

		bet := &Bet {
			name: name,
			surname: surname,
			document: document,
			birthday: birthday,
			amount: uint64(amount),
		}

		serialized_bet := bet.serialize(c.config.ID)
		if offset + len(serialized_bet) > MAX_BATCH_SIZE {
			// Cotemplates the case where a bet does not fit inside the current batch
			left_out_bet = bet
			break
		} else {
			// This is the case where it fits
			for i := 0; i < len(serialized_bet); i++ {
				buffer[offset + i] = serialized_bet[i]
			}
			offset += len(serialized_bet)
		}

	}

	batch := buffer[0:offset]
	packaged_batch := c.packageBets(batch)

	return packaged_batch, left_out_bet, nil, file_has_lines
}

/// 1 byte de indicador
/// 8 bytes de longitud
func (c *Client) packageBets(bets []byte) []byte {
	bets_batch_indicator := [1]byte {byte(uint8(ClientBetBatch))}
	bets_length := uint64(len(bets))
	bets_length_b := SerializeUInteger64(bets_length)

	bets_header := append(bets_batch_indicator[:], bets_length_b...)

	full_package := append(bets_header[:], bets...)

	return full_package
}

func (c *Client) receiveMessage(size int) ([]byte, error) {
	buffer := make([]byte, size)

	var received = 0
	var err error
	for offset := 0 ; offset < size ; offset += received {
		received, err = c.conn.Read(buffer[received:])
		if err != nil {
			break
		}
	}

	return buffer, err
}


// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	var initial_bet *Bet = nil
	var file_has_lines = true
	// TODO: No hacer que se reconecte en cada iteracion
	// Create the connection the server in every loop iteration. Send an
	c.createClientSocket()
	for ; file_has_lines == true; {
		// If the client is killed, break out of the loop inmediately
		if c.killed {
			break
		}

		bets, left_out_bet, err, still_has_lines := c.createBatch(initial_bet)
		file_has_lines = still_has_lines
		initial_bet = left_out_bet


		c.sendToServer(bets)


		// Now we receive two bytes representing 'ok'
		msg, err := c.receiveMessage(1)
		if err != nil {
			// TODO: Actualizar
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		exit_status := int8(msg[0])
		if exit_status != 0 {
			log.Errorf("action: apuestas_enviadas | result: fail | dni: %v | server_exit_status: %v",
				c.config.ID,
				exit_status,
			)
			return
		}



		log.Infof("action: apuestas_enviadas | result: success")

		// Wait a time between sending one message and the next one
		// time.Sleep(c.config.LoopPeriod)

	}
	// msg, err := bufio.NewReader(c.conn).ReadString('\n')
	end_indicator := [1]byte { byte(uint8(ClientBetEnd)) };
	c.sendToServer(end_indicator[:])

	c.receiveWinners()

	// time.Sleep(c.config.LoopPeriod)

	c.conn.Close()
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
