package netlink

import (
	"bytes"

	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"syscall"
)

func netlinkPadding(size int) int {
	partialChunk := size % syscall.NLMSG_ALIGNTO
	return (syscall.NLMSG_ALIGNTO - partialChunk) % syscall.NLMSG_ALIGNTO
}

func skipAlignedFromSlice(r *bytes.Buffer, dataLen int) error {
	r.Next(dataLen + netlinkPadding(dataLen))
	return nil
}

func strtoi(s string) int {
	i := 0
	for _, c := range s {
		i *= 10
		i += int(c) - '0'
	}
	return i
}

// Returns pointer to a field, and type information corresponding to a given numerical ID.
func getDestinationAndType(object interface{}, id uint16) (reflect.Value, string, error) {
	ptrType := reflect.TypeOf(object)
	// check the object is a pointer
	if ptrType.Kind() != reflect.Ptr {
		er := fmt.Errorf("getDestinationAndType() received"+
			" object of Kind %s, expected pointer!", ptrType.Kind())
		return reflect.ValueOf(nil), "", er
	}
	// check the indirected object is a struct
	objType := ptrType.Elem()
	if objType.Kind() != reflect.Struct {
		er := fmt.Errorf("getDestinationAndType() received"+
			" a pointer to %s, expected pointer to struct!", objType.Kind())
		return reflect.ValueOf(nil), "", er
	}
	// find appropriate field
	for i := 0; i < objType.NumField(); i++ {
		objField := objType.Field(i)
		if strtoi(objField.Tag.Get("netlink")) == int(id) {
			// found field
			type_s := objField.Tag.Get("type")
			// returns ValueOf((*object).field)
			fieldValue := reflect.Indirect(reflect.ValueOf(object)).Field(i)
			return fieldValue, type_s, nil
		}
	}
	er := fmt.Errorf("could not find field ID %d in object of type %s",
		id, objType)
	return reflect.ValueOf(nil), "", er
}

// Reads one attribute into a structure.
// dest must be a pointer to a struct.
func readAttribute(r *bytes.Buffer, dest interface{}) (er error) {
	var attr syscall.RtAttr
	er = binary.Read(r, SystemEndianness, &attr)
	if er != nil {
		return er
	}
	dataLen := int(attr.Len) - syscall.SizeofRtAttr
	value, type_spec, er := getDestinationAndType(dest, attr.Type)
	switch true {
	case er != nil:
		return er
	case type_spec == "none":
		value.Set(reflect.ValueOf(true))
	case type_spec == "fixed":
		// The payload is a binary struct
		if !value.CanAddr() {
			return fmt.Errorf("trying to read fixed-width data in a non addressable field!")
		}
		er = binary.Read(r, SystemEndianness, value.Addr().Interface())
	case type_spec == "bytes":
		// The payload is a raw sequence of bytes
		buf := make([]byte, dataLen)
		_, er = r.Read(buf[:])
		value.Set(reflect.ValueOf(buf))
	case type_spec == "string":
		// The payload is a NUL-terminated byte array
		if value.Type().Kind() != reflect.String {
			return fmt.Errorf("unable to fill field of type %s with string!", value.Type())
		}
		buf := make([]byte, dataLen)
		_, er = r.Read(buf[:])
		s := string(buf[:len(buf)-1])
		value.Set(reflect.ValueOf(s))
	case type_spec == "nested":
		// The payload is a seralized sequence of attributes
		// <header> (<header1> <attribute1> ... <header n> <attribute n>)
		if !value.CanAddr() {
			return fmt.Errorf("trying to read nested attributes to a non addressable field!")
		}
		buf := make([]byte, dataLen)
		_, er = r.Read(buf[:])
		er = ReadManyAttributes(bytes.NewBuffer(buf), value.Addr().Interface())
	case type_spec == "nestedlist":
		// The payload is a sequence of nested attributes, each of them carrying
		// a payload describing a struct with attributes
		// <header (4 bytes)> <payload>
		// where payload is
		// <header1> <nested attributes 1> ... <headern> <nested attributes n>
		buf := make([]byte, dataLen)
		_, er = r.Read(buf[:])
		er = readNestedAttributeList(bytes.NewBuffer(buf), value)
	default:
		return fmt.Errorf("Invalid format tag %s: expecting 'fixed', 'bytes', 'string', or 'nested'", type_spec)
	}
	r.Next(netlinkPadding(dataLen))
	return er
}

func ReadManyAttributes(r *bytes.Buffer, dest interface{}) (er error) {
	for {
		er := readAttribute(r, dest)
		switch er {
		case nil:
			break
		case io.EOF:
			return nil
		default:
			return er
		}
	}
	return nil
}

// Reads n nested attributes into the elements of an array
func readNestedAttributeList(r *bytes.Buffer, dest reflect.Value) (er error) {
	if dest.Type().Kind() != reflect.Slice {
		return fmt.Errorf("unable to fill field of type %s with list of nested attrs!", dest.Type())
	}
	for {
		var attr syscall.RtAttr
		er = binary.Read(r, SystemEndianness, &attr)
		switch er {
		case nil:
			break
		case io.EOF:
			return nil
		default:
			return er
		}
		dataLen := int(attr.Len) - syscall.SizeofRtAttr

		// Create buffer for nested attribute
		buf := make([]byte, dataLen)
		_, er = r.Read(buf[:])
		if er != nil {
			return er
		}

		// Read the value
		item := reflect.New(dest.Type().Elem())
		er = ReadManyAttributes(bytes.NewBuffer(buf), item.Interface())
		if er != nil {
			return er
		}

		// Append the value
		dest.Set(reflect.Append(dest, reflect.Indirect(item)))
	}
	return nil
}

func PutAttribute(w *bytes.Buffer, attrtype uint16, data interface{}) error {
	attr := Attr{Len: syscall.SizeofRtAttr, Type: attrtype}
	switch data := data.(type) {
	case []byte:
		attr.Len += uint16(len(data))
		binary.Write(w, SystemEndianness, attr)
		binary.Write(w, SystemEndianness, data)
	case string:
		attr.Len += uint16(len(data)) + 1
		binary.Write(w, SystemEndianness, attr)
		binary.Write(w, SystemEndianness, []byte(data))
		w.WriteByte(0)
	default:
		attr.Len += uint16(sizeof(data))
		binary.Write(w, SystemEndianness, attr)
		binary.Write(w, SystemEndianness, data)
	}
	for i := 0; i < netlinkPadding(int(attr.Len)); i++ {
		w.WriteByte(0)
	}
	return nil
}

func sizeof(data interface{}) int {
	var v reflect.Value
	switch d := reflect.ValueOf(data); d.Kind() {
	case reflect.Ptr:
		v = d.Elem()
	case reflect.Slice:
		v = d
	default:
		v = d
	}
	return binary.Size(v.Interface())
}
