package common

import (
	"fmt"

	"bufio"
	"net"
	"time"
	"os"
	"strconv"

	"github.com/op/go-logging"

	"github.com/pkg/errors"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
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
	ClientBet ClientValues = iota
)

// A Bet is serialized as the following:
// A byte indicating that it's a bet
// Its length
// All the rest of the elements serialized
func (b *Bet) serialize() []byte {
	name := SerializeString(b.name)
	surname := SerializeString(b.surname)
	document := SerializeString(b.document)
	birthday := SerializeString(b.birthday)
	amount := SerializeUInteger64(b.amount)

	fields := append(name, surname...)
	fields = append(fields, document...)
	fields = append(fields, birthday...)
	fields = append(fields, amount...)

	length := len(fields)

	buffer_len := length + 2
	buffer := make([]byte, buffer_len)

	buffer[0] = byte(ClientBet)
	buffer[1] = byte(buffer_len)
	for i := 0; i < length; i++ {
		current_byte := fields[i]
		buffer[i + 2] = current_byte
	}

	fmt.Printf("Bet\n")
	fmt.Printf("%v\n", buffer)
	fmt.Printf("%#v\n", buffer)
	fmt.Printf("%+v\n", buffer)

	return buffer
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
	config ClientConfig
	conn   net.Conn
	killed bool

	bet    Bet
}

// Stop the client before hand.
func (c *Client) Close() {
     socket_err :=  c.conn.Close()
	if socket_err == nil {
	   log.Debug("action: close_socket | result: success | client_id: %v", c.config.ID);
     } else {
	   log.Debug("action: close_socket | result: failure | client_id: %v", c.config.ID);
	}
	c.killed = true
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig, bet Bet) *Client {
	client := &Client{
		config: config,
		killed: false,

		bet: bet,
	}
	return client
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

func (c *Client) sendToServer() error {
	bet := c.bet.serialize()
	length := len(bet)

	var sent = 0
	var err error
	for offset := 0 ; offset < length ; offset += sent {
		sent, err = c.conn.Write(bet[offset:])
		if err != nil {
			return err
		}
	}

	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// If the client is killed, break out of the loop inmediately
		if c.killed {
			break
		}
		// Create the connection the server in every loop iteration. Send an
		c.createClientSocket()

		// TODO: Modify the send to avoid short-write
		c.sendToServer()

		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		c.conn.Close()

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		// Wait a time between sending one message and the next one
		time.Sleep(c.config.LoopPeriod)

	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
