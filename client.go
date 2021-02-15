package mordhaurcon

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const (
	serverDataAuth        = 3
	serverDataExecCommand = 2
)

var (
	defaultBroadcastHandler           = func(string) {}
	defaultBroadcastKeepAliveInterval = time.Minute * 60
)

type broadcastHandlerFunc func(string)
type disconnectHandlerFunc func(err error, expected bool)

// Client is the struct which facilitates all RCON client functionality.
// Clients should not be created manually, instead they should be created using NewClient.
type Client struct {
	address       string
	password      string
	mainConn      *net.TCPConn
	broadcastConn *net.TCPConn
	config        *ClientConfig
}

// ClientConfig holds configurable values for use by the RCON client.
// Host is required.
// Port is required.
// Password is required.
// SendheartBeatCommand is optional.
// HeartbeatCommandInterval is optional (Default: 60s).
// BroadcastHandler is optional.
type ClientConfig struct {
	Host                     string                // required
	Port                     int16                 // required
	Password                 string                // required
	SendHeartbeatCommand     bool                  // optional. default: false
	AttemptReconnect         bool                  // optional. default: false
	HeartbeatCommandInterval time.Duration         // optional. default: 30 seconds
	EnableBroadcasts         bool                  // optional
	BroadcastHandler         broadcastHandlerFunc  // optional
	DisconnectHandler        disconnectHandlerFunc // optional
}

// NewClient is used to properly create a new instance of Client.
// It takes in the address and port of the RCON server you wish to connect to
// as well as your RCON password.
func NewClient(config *ClientConfig) *Client {
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)

	client := &Client{
		address:  address,
		password: config.Password,
		config:   config,
	}

	// If client.config.HeartbeatCommandInterval is 0s, then assume a value wasn't provided and
	// set it to the default value.

	return client
}

// SetBroadcastHandler accepts a broadcastHandlerFunc and updates the client's internal broadcastHandler
// field to the one passed in. By default, broadcastHandler is null so this function must be used at least
// once to get access to broadcast messages.
//
// It should also be noted that not all messages will necessarily be broadcasts. For example, the "Alive" command
// used to keep the socket alive will also have it's output sent to the broadcastHandler. Because of this, it's
// important that you make sure you only process the data you wish with your own logic within your handler.
func (c *Client) SetBroadcastHandler(handler broadcastHandlerFunc) {
	c.config.BroadcastHandler = handler
}

// SetDisconnectHandler accepts a disconnectHandlerFunc and updates the client's internal disconnectHandler
// field to the value passed in. The disconnect handler is called when a socket disconnects.
func (c *Client) SetDisconnectHandler(handler disconnectHandlerFunc) {
	c.config.DisconnectHandler = handler
}

// SetSendHeartbeatCommand enables an occasional heartbeat command to be sent to the server to keep the broadcasting
// socket alive.
func (c *Client) SetSendHeartbeatCommand(enabled bool) {
	c.config.SendHeartbeatCommand = enabled
}

// SetHeartbeatCommandInterval sets the interval at which the client will send out a heartbeat command to the server
// to keep the broadcast socket alive. This is only done if heartbeat commands were enabled.
func (c *Client) SetHeartbeatCommandInterval(interval time.Duration) {
	c.config.HeartbeatCommandInterval = interval
}

// Connect tries to open a socket and authentciated to the RCON server specified during client setup.
// This socket is used exclusively for command executions. For broadcast listening, see ListenForBroadcasts().
// The default value is 30 seconds (30*time.Second).
func (c *Client) Connect() error {
	dialer := net.Dialer{Timeout: time.Second * 10}

	rawConn, err := dialer.Dial("tcp", c.address)
	if err != nil {
		return err
	}

	c.mainConn = rawConn.(*net.TCPConn)

	// Enable keepalive
	c.mainConn.SetKeepAlive(true)

	// Authenticate
	c.authenticate(c.mainConn)

	return nil
}

func (c *Client) Disconnect() error {
	if err := c.mainConn.Close(); err != nil {
		return err
	}

	if err := c.broadcastConn.Close(); err != nil {
		return err
	}

	if c.config.DisconnectHandler != nil {
		c.config.DisconnectHandler(nil, true)
	}

	return nil
}

// ExecCommand executes a command on the RCON server. It returns the response body from the server
// or an error if something went wrong.
func (c *Client) ExecCommand(command string) (string, error) {
	return c.execCommand(c.mainConn, command)
}

// ListenForBroadcasts is the function which kicks of broadcast listening. It opens a second socket to the
// RCON server meant specifically for listening for broadcasts and periodically runs a command to keep the
// connection alive.
func (c *Client) ListenForBroadcasts(broadcastTypes []string, errors chan error) {
	// Make sure broadcast listening is enabled
	if !c.config.EnableBroadcasts {
		return
	}

	// Open broadcast socket
	err := c.connectBroadcastListener(broadcastTypes)
	if err != nil {
		errors <- err
	}

	if c.config.SendHeartbeatCommand {
		c.startBroadcasterHeartBeat(errors)
	}

	// Start listening for broadcasts
	go func() {
		for {
			response, err := buildPayloadFromPacket(c.broadcastConn)
			if err != nil {
				if err == io.EOF || err == io.ErrClosedPipe {
					fmt.Println("Broadcast listener closed")

					if c.config.AttemptReconnect {
						fmt.Println("Attempting to reconnect...")

						// If EOF was read, then try reconnecting to the server.
						err := c.connectBroadcastListener(broadcastTypes)
						if err != nil {
							errors <- err
						}
					}

					if c.config.DisconnectHandler != nil {
						c.config.DisconnectHandler(err, false)
					}

					return
				} else {
					errors <- err
				}
			}

			if response == nil {
				continue
			}

			if response.isNotBroadcast() {
				continue
			}

			if c.config.BroadcastHandler != nil {
				c.config.BroadcastHandler(string(response.Body))
			}
		}
	}()
}

func (c *Client) startBroadcasterHeartBeat(errors chan error) {
	ticker := time.NewTicker(c.config.HeartbeatCommandInterval)
	done := make(chan bool)

	// Start broadcast listener keepalive routine
	go func() {
		for {
			select {
			case <-ticker.C:
				keepAlivePayload := newPayload(serverDataExecCommand, []byte("Alive"))
				keepAlivePacket, err := buildPacketFromPayload(keepAlivePayload)
				if err != nil {
					errors <- err
					return
				}

				_, err = c.broadcastConn.Write(keepAlivePacket)
				if err != nil {
					errors <- err
					return
				}
				break
			case <-done:
				ticker.Stop()
				close(done)
			}
		}
	}()
}

func (c *Client) authenticate(socket *net.TCPConn) error {
	payload := newPayload(serverDataAuth, []byte(c.password))

	_, err := sendPayload(socket, payload)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) execCommand(socket *net.TCPConn, command string) (string, error) {
	payload := newPayload(serverDataExecCommand, []byte(command))

	response, err := sendPayload(socket, payload)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(response.Body)), nil
}

func (c *Client) openBroadcastListenerSocket() error {
	// Dial out with a second connection specifically meant for receiving broadcasts.
	dialer := net.Dialer{Timeout: time.Second * 10}
	bcConn, err := dialer.Dial("tcp", c.address)
	if err != nil {
		return err
	}
	c.broadcastConn = bcConn.(*net.TCPConn)

	// Disable deadlines as we can't guarantee when we'll receive broadcasts
	c.broadcastConn.SetDeadline(time.Time{})
	c.broadcastConn.SetReadDeadline(time.Time{})
	c.broadcastConn.SetWriteDeadline(time.Time{})
	c.broadcastConn.SetKeepAlive(true)

	return nil
}

func (c *Client) connectBroadcastListener(broadcastTypes []string) error {
	// Dial out with a second connection specifically meant
	// for receiving broadcasts.
	err := c.openBroadcastListenerSocket()
	if err != nil {
		return err
	}

	// Authenticate
	err = c.authenticate(c.broadcastConn)
	if err != nil {
		return err
	}

	// Subscribe to broadcast types
	for _, broadcastType := range broadcastTypes {
		_, err := c.execCommand(c.broadcastConn, fmt.Sprintf("listen %s", broadcastType))
		if err != nil {
			return err
		}

		log.Printf("RCON client listening for %s broadcasts.\n", broadcastType)
	}

	return nil
}
