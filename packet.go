package mordhaurcon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

func buildPacketFromPayload(payload *payload) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})

	// Write payload data into buffer using LittleEndian as specified in the
	// Source RCON specification.
	binary.Write(buffer, binary.LittleEndian, payload.getSize())
	binary.Write(buffer, binary.LittleEndian, payload.ID)
	binary.Write(buffer, binary.LittleEndian, payload.Type)
	binary.Write(buffer, binary.LittleEndian, payload.Body)
	binary.Write(buffer, binary.LittleEndian, [2]byte{}) // write null bytes

	if buffer.Len() >= payloadMaxSize {
		return nil, fmt.Errorf("Payload too large. Max size: %d", payloadMaxSize)
	}

	return buffer.Bytes(), nil
}

func buildPayloadFromPacket(socket net.Conn) (*payload, error) {
	var packetSize int32
	var packetID int32
	var packetType int32

	// Read header bytes
	err := binary.Read(socket, binary.LittleEndian, &packetSize)
	if err != nil {
		return nil, err
	}

	err = binary.Read(socket, binary.LittleEndian, &packetID)
	if err != nil {
		return nil, err
	}

	err = binary.Read(socket, binary.LittleEndian, &packetType)
	if err != nil {
		return nil, err
	}

	packetBodyLen := packetSize - (payloadIDBytes + payloadTypeBytes)

	if packetBodyLen < 1 {
		return nil, fmt.Errorf("Empty packet body received")
	}

	// Create byte slice to read the body into
	packetBody := make([]byte, packetBodyLen)

	_, err = io.ReadFull(socket, packetBody)
	if err != nil {
		return nil, err
	}

	// Trim unneeded bytes
	packetBody = bytes.Trim(packetBody, "\x00")
	packetBody = bytes.Trim(packetBody, "\n")

	payload := &payload{
		ID:   packetID,
		Type: packetType,
		Body: packetBody,
	}

	return payload, nil
}
