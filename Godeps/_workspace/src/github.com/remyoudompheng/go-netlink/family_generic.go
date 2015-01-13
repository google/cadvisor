package netlink

import (
	"bytes"
	"encoding/binary"
	"syscall"
)

type GenlMsghdr struct {
	Command  uint8
	Version  uint8
	Reserved uint16
}

// Netlink messages are aligned to 4 bytes multiples
type GenericNetlinkMessage struct {
	Header    syscall.NlMsghdr // 16 bytes
	GenHeader GenlMsghdr       // 4 bytes
	Data      []byte
}

func (msg *GenericNetlinkMessage) toRawMsg() (rawmsg syscall.NetlinkMessage) {
	rawmsg.Header = msg.Header
	w := bytes.NewBuffer([]byte{})
	binary.Write(w, SystemEndianness, msg.GenHeader)
	w.Write(msg.Data)
	rawmsg.Data = w.Bytes()
	return rawmsg
}

func ParseGenlMessage(msg syscall.NetlinkMessage) (genmsg GenericNetlinkMessage, er error) {
	genmsg.Header = msg.Header
	buf := bytes.NewBuffer(msg.Data)
	binary.Read(buf, SystemEndianness, &genmsg.GenHeader)
	genmsg.Data = buf.Bytes()
	return genmsg, nil
}
