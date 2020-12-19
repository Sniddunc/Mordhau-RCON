package mordhaurcon

import (
	"fmt"
	"log"
	"net"
	"regexp"
)

const (
	packetIDAuthFailed = -1
	payloadIDBytes     = 4
	payloadTypeBytes   = 4
	payloadNullBytes   = 2
	payloadMaxSize     = 2048
)

var (
	nonBroadcastPatterns = []*regexp.Regexp{}

	currentPayloadID int32 = 0
)

func init() {
	// Add known non-broadcast regexp patterns
	patterns := []string{
		"^Keeping client alive for another\\s[0-9]{1,6}\\sseconds$",
	}

	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			log.Fatalf("Invalid regexp. Error: %v\n", err)
		}

		nonBroadcastPatterns = append(nonBroadcastPatterns, compiled)
	}
}

type payload struct {
	ID   int32
	Type int32
	Body []byte
}

func newPayload(payloadType int, body []byte) *payload {
	currentPayloadID++

	return &payload{
		ID:   int32(currentPayloadID),
		Type: int32(payloadType),
		Body: []byte(body),
	}
}

func (p *payload) getSize() int32 {
	return int32(len(p.Body) + (payloadIDBytes + payloadTypeBytes + payloadNullBytes))
}

func (p *payload) isNotBroadcast() bool {
	for _, pattern := range nonBroadcastPatterns {

		// If payload body matches a known non-broadcast pattern, we can safely
		// assume that it's not a broadcast so we return true.
		if pattern.MatchString(string(p.Body)) {
			return true
		}
	}

	// If none of the known non-broadcast patterns were matches, return false.
	return false
}

func sendPayload(conn net.Conn, request *payload) (*payload, error) {
	packet, err := buildPacketFromPayload(request)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write(packet)
	if err != nil {
		return nil, err
	}

	response, err := buildPayloadFromPacket(conn)
	if err != nil {
		return nil, err
	}

	if response.ID == packetIDAuthFailed {
		return nil, fmt.Errorf("Authentication failed")
	}

	return response, nil
}
