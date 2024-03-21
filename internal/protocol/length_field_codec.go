package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/panjf2000/gnet/v2"
)

var ErrIncompletePacket = errors.New("incomplete packet")
var ErrCheckFailPacket = errors.New("check fail packet")
var ErrInvalidMagicNumber = errors.New("invalid magic number")
var ErrTooLargeBody = errors.New("body too large")

const ConnectedAck = "server connected\n"

const (
	bodySizeHexNum = 4
	bodySize       = 4096
)

var magicNumberBytes []byte = []byte{10, 12, 10, 15}
var magicNumberSize = 4

// LengthFieldBasedFrameCodec Protocol format:
//
// * 0           4                       4
// * +-----------+-----------------------+
// * |   magic   |       body len        |
// * +-----------+-----------+-----------+
// * |                                   |
// * +                                   +
// * |           body bytes (max 4096)   |
// * +                                   +
// * |            ... ...                |
// * +-----------------------------------+
type LengthFieldBasedFrameCodec struct{}

// Encode encodes the provided buffer into a byte slice.
//
// The encoded data consists of the following parts:
//
// * Magic number (4 bit): A fixed value identifying the protocol.
// * Body length (12 bit): The length of the body in bytes.
// * Body (max 4096 bit): The actual data to be sent.
// * CRC-16 checksum (4 bit): A checksum of the magic number, body length, and body bytes.
//
// Parameter(s):
//
//	buf []byte: The data to be encoded.
//
// Return type(s):
//
//	[]byte, error: The encoded data and/or an error if something went wrong.
func (codec LengthFieldBasedFrameCodec) Encode(buf []byte) ([]byte, error) {
	// if len(buf) > bodySize {
	// return nil, ErrTooLargeBody
	// }
	bodyOffset := magicNumberSize + bodySizeHexNum
	msgLen := bodyOffset + len(buf)
	data := make([]byte, msgLen)
	copy(data, magicNumberBytes)

	bodyLen := IntToBytes(len(buf))
	copy(data[bodyOffset-len(bodyLen):bodyOffset], []byte(bodyLen))
	copy(data[bodyOffset:msgLen], buf)
	return data, nil
}

// Decode decodes the data received from the connection.
//
// It reads the first bodyOffset bytes from the connection and verifies
// that they contain the correct magic number and body length.
//
// It then reads the entire message (body length + magic number + CRC-16 checksum)
// from the connection and discards the magic number and CRC-16 checksum.
//
// c - gnet.Conn
// []byte, error
func (codec *LengthFieldBasedFrameCodec) Decode(c gnet.Conn) ([]byte, error) {
	bodyOffset := magicNumberSize + bodySizeHexNum
	// Read the first bodyOffset bytes from the connection
	buf, _ := c.Peek(bodyOffset)
	if len(buf) < bodyOffset {
		// The buffer doesn't contain a complete packet
		return nil, ErrIncompletePacket
	}

	// Check the magic number
	if !bytes.Equal(magicNumberBytes, buf[:magicNumberSize]) {
		return nil, ErrInvalidMagicNumber
	}

	// Extract the body length
	bodyLen := BytesToInt(buf[magicNumberSize:bodyOffset])

	// Calculate the full message length
	msgLen := bodyOffset + bodyLen

	// Check if the connection contains a complete packet
	if c.InboundBuffered() < msgLen {
		return nil, ErrIncompletePacket
	}
	// Read the entire message from the connection
	buf, _ = c.Peek(msgLen)
	_, _ = c.Discard(msgLen)

	// Return the decoded data (without the magic number and CRC-16 checksum)
	return buf[bodyOffset : bodyOffset+bodyLen], nil
}

// Unpack unpacks the buf byte slice and returns the extracted data or an error.
//
// buf []byte - the byte slice to be unpacked.
//
// Returns:
// []byte - the extracted data
// error  - an error if the unpacking failed
func (codec LengthFieldBasedFrameCodec) Unpack(buf []byte) ([]byte, error) {
	// The minimum length of a valid packet is the size of the magic number (4 bytes)
	// + the size of the body length field (12 bit)
	// + the size of the CRC-16 checksum (4 bit)
	bodyOffset := magicNumberSize + bodySize
	if len(buf) < bodyOffset {
		return nil, ErrIncompletePacket
	}

	// Check the magic number
	if !bytes.Equal(magicNumberBytes, buf[:magicNumberSize]) {
		return nil, ErrInvalidMagicNumber
	}

	// Extract the body length
	bodyLen := 4

	// Calculate the full message length
	msgLen := bodyOffset + bodyLen

	// Check if the buf contains a complete packet
	if len(buf) < msgLen {
		return nil, ErrIncompletePacket
	}

	// Check the CRC-16 checksum
	crc16 := buf[bodyOffset+bodyLen : msgLen]
	if crc16 == nil {
		return nil, ErrCheckFailPacket
	}

	// Return the extracted data
	return buf[bodyOffset : bodyOffset+bodyLen], nil
}

func (codec *LengthFieldBasedFrameCodec) DecodeReader(r *bufio.Reader) ([]byte, error) {
	bodyOffset := magicNumberSize + bodySizeHexNum
	// Read the first bodyOffset bytes from the connection
	buf := make([]byte, bodyOffset)
	_, err := r.Read(buf)
	if err != nil {
		return nil, err
	}

	if len(buf) < bodyOffset {
		// The buffer doesn't contain a complete packet
		return nil, ErrIncompletePacket
	}

	// Check the magic number
	if !bytes.Equal(magicNumberBytes, buf[:magicNumberSize]) {
		return nil, ErrInvalidMagicNumber
	}

	// Extract the body length
	bodyLen := BytesToInt(buf[magicNumberSize:bodyOffset])

	buf = make([]byte, bodyLen)

	// Read the entire message from the connection
	_, _ = r.Read(buf)

	// Return the decoded data (without the magic number and CRC-16 checksum)
	return buf, nil
}

func IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func BytesToInt(bs []byte) int {
	var i int32
	buf := bytes.NewBuffer(bs)
	binary.Read(buf, binary.BigEndian, &i)
	return int(i)
}
