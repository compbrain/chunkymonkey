package proto

import (
	"encoding/binary"
	"io"
	"log"
	"math"
	"os"
	"reflect"

	. "chunkymonkey/types"
)

// Possible error values for reading and writing packets.
var (
	ErrorPacketNotPtr      = os.NewError("packet not passed as a pointer")
	ErrorPacketNil         = os.NewError("packet was passed by a nil pointer")
	ErrorStrLengthNegative = os.NewError("string length was negative")
	ErrorStrTooLong        = os.NewError("string was too long")
	ErrorInternal          = os.NewError("implementation problem with packetization")
)

// Packet definitions.

type PacketKeepAlive struct {
	Id int32
}

type PacketLogin struct {
	VersionOrEntityId int32
	Username          string
	MapSeed           RandomSeed
	GameMode          int32
	Dimension         DimensionId
	Difficulty        GameDifficulty
	WorldHeight       byte
	MaxPlayers        byte
}

type PacketHandshake struct {
	UsernameOrHash string
}

type PacketUseEntity struct {
	User      EntityId
	Target    EntityId
	LeftClick bool
}

type PacketPlayerPosition struct {
	X, Y1, Y2, Z AbsCoord
	OnGround     bool
}

// PacketSerializer reads and writes packets. It is not safe to use one
// simultaneously between multiple goroutines.
type PacketSerializer struct {
	// Scratch space to be able to encode up to 64bit values without allocating.
	scratch [8]byte
}

func (ps *PacketSerializer) readUint8(reader io.Reader) (v uint8, err os.Error) {
	if _, err = io.ReadFull(reader, ps.scratch[0:1]); err != nil {
		return
	}
	return ps.scratch[0], nil
}

func (ps *PacketSerializer) readInt8(reader io.Reader) (v int8, err os.Error) {
	uv, err := ps.readUint8(reader)
	return int8(uv), err
}

func (ps *PacketSerializer) ReadPacket(reader io.Reader, packet interface{}) (err os.Error) {
	// TODO Check packet is CanSettable? (if settable at the top, does that
	// follow for all its descendants?)
	value := reflect.ValueOf(packet)
	kind := value.Kind()
	if kind != reflect.Ptr {
		return ErrorPacketNotPtr
	} else if value.IsNil() {
		return ErrorPacketNil
	}

	return ps.readData(reader, reflect.Indirect(value))
}

func (ps *PacketSerializer) readData(reader io.Reader, value reflect.Value) (err os.Error) {
	kind := value.Kind()

	switch kind {
	case reflect.Struct:
		numField := value.NumField()
		for i := 0; i < numField; i++ {
			field := value.Field(i)
			if err = ps.readData(reader, field); err != nil {
				return
			}
		}

	case reflect.Bool:
		if _, err = io.ReadFull(reader, ps.scratch[0:1]); err != nil {
			return
		}
		value.SetBool(ps.scratch[0] != 0)

		// Integer types:

	case reflect.Int8:
		if _, err = io.ReadFull(reader, ps.scratch[0:1]); err != nil {
			return
		}
		value.SetInt(int64(ps.scratch[0]))
	case reflect.Int16:
		if _, err = io.ReadFull(reader, ps.scratch[0:2]); err != nil {
			return
		}
		value.SetInt(int64(binary.BigEndian.Uint16(ps.scratch[0:2])))
	case reflect.Int32:
		if _, err = io.ReadFull(reader, ps.scratch[0:4]); err != nil {
			return
		}
		value.SetInt(int64(binary.BigEndian.Uint32(ps.scratch[0:4])))
	case reflect.Int64:
		if _, err = io.ReadFull(reader, ps.scratch[0:8]); err != nil {
			return
		}
		value.SetInt(int64(binary.BigEndian.Uint64(ps.scratch[0:8])))
	case reflect.Uint8:
		if _, err = io.ReadFull(reader, ps.scratch[0:1]); err != nil {
			return
		}
		value.SetUint(uint64(ps.scratch[0]))
	case reflect.Uint16:
		if _, err = io.ReadFull(reader, ps.scratch[0:2]); err != nil {
			return
		}
		value.SetUint(uint64(binary.BigEndian.Uint16(ps.scratch[0:2])))
	case reflect.Uint32:
		if _, err = io.ReadFull(reader, ps.scratch[0:4]); err != nil {
			return
		}
		value.SetUint(uint64(binary.BigEndian.Uint32(ps.scratch[0:4])))
	case reflect.Uint64:
		if _, err = io.ReadFull(reader, ps.scratch[0:8]); err != nil {
			return
		}
		value.SetUint(binary.BigEndian.Uint64(ps.scratch[0:8]))

		// Floating point types:

	case reflect.Float32:
		if _, err = io.ReadFull(reader, ps.scratch[0:4]); err != nil {
			return
		}
		value.SetFloat(float64(math.Float32frombits(binary.BigEndian.Uint32(ps.scratch[0:4]))))

	case reflect.Float64:
		if _, err = io.ReadFull(reader, ps.scratch[0:8]); err != nil {
			return
		}
		value.SetFloat(math.Float64frombits(binary.BigEndian.Uint64(ps.scratch[0:8])))

	case reflect.String:
		// TODO Maybe the tag field could/should suggest a max length.
		if _, err = io.ReadFull(reader, ps.scratch[0:2]); err != nil {
			return
		}
		length := int16(binary.BigEndian.Uint16(ps.scratch[0:2]))
		if length < 0 {
			return ErrorStrLengthNegative
		}
		codepoints := make([]uint16, length)
		if err = binary.Read(reader, binary.BigEndian, codepoints); err != nil {
			return
		}
		value.SetString(encodeUtf8(codepoints))

	default:
		// TODO
		typ := value.Type()
		log.Printf("Unimplemented type in packet: %v", typ)
		return ErrorInternal
	}
	return
}

func (ps *PacketSerializer) WritePacket(writer io.Writer, packet interface{}) (err os.Error) {
	value := reflect.ValueOf(packet)
	kind := value.Kind()
	if kind == reflect.Ptr {
		value = reflect.Indirect(value)
	}

	return ps.writeData(writer, value)
}

func (ps *PacketSerializer) writeData(writer io.Writer, value reflect.Value) (err os.Error) {
	kind := value.Kind()

	switch kind {
	case reflect.Struct:
		numField := value.NumField()
		for i := 0; i < numField; i++ {
			field := value.Field(i)
			if err = ps.writeData(writer, field); err != nil {
				return
			}
		}

	case reflect.Bool:
		if value.Bool() {
			ps.scratch[0] = 1
		} else {
			ps.scratch[0] = 0
		}
		_, err = writer.Write(ps.scratch[0:1])

		// Integer types:

	case reflect.Int8:
		ps.scratch[0] = byte(value.Int())
		_, err = writer.Write(ps.scratch[0:1])
	case reflect.Int16:
		binary.BigEndian.PutUint16(ps.scratch[0:2], uint16(value.Int()))
		_, err = writer.Write(ps.scratch[0:2])
	case reflect.Int32:
		binary.BigEndian.PutUint32(ps.scratch[0:4], uint32(value.Int()))
		_, err = writer.Write(ps.scratch[0:4])
	case reflect.Int64:
		binary.BigEndian.PutUint64(ps.scratch[0:8], uint64(value.Int()))
		_, err = writer.Write(ps.scratch[0:8])
	case reflect.Uint8:
		ps.scratch[0] = byte(value.Uint())
		_, err = writer.Write(ps.scratch[0:1])
	case reflect.Uint16:
		binary.BigEndian.PutUint16(ps.scratch[0:2], uint16(value.Uint()))
		_, err = writer.Write(ps.scratch[0:2])
	case reflect.Uint32:
		binary.BigEndian.PutUint32(ps.scratch[0:4], uint32(value.Uint()))
		_, err = writer.Write(ps.scratch[0:4])
	case reflect.Uint64:
		binary.BigEndian.PutUint64(ps.scratch[0:8], value.Uint())
		_, err = writer.Write(ps.scratch[0:8])

		// Floating point types:

	case reflect.Float32:
		binary.BigEndian.PutUint32(ps.scratch[0:4], math.Float32bits(float32(value.Float())))
		_, err = writer.Write(ps.scratch[0:4])
	case reflect.Float64:
		binary.BigEndian.PutUint64(ps.scratch[0:8], math.Float64bits(value.Float()))
		_, err = writer.Write(ps.scratch[0:8])

	case reflect.String:
		lengthInt := value.Len()
		if lengthInt > math.MaxInt16 {
			return ErrorStrTooLong
		}
		binary.BigEndian.PutUint16(ps.scratch[0:2], uint16(lengthInt))
		if _, err = writer.Write(ps.scratch[0:2]); err != nil {
			return
		}
		codepoints := decodeUtf8(value.String())
		err = binary.Write(writer, binary.BigEndian, codepoints)

	default:
		// TODO
		typ := value.Type()
		log.Printf("Unimplemented type in packet: %v", typ)
		return ErrorInternal
	}

	return
}