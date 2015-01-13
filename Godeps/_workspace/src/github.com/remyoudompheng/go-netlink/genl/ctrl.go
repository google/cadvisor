package genl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	. "github.com/remyoudompheng/go-netlink"
	"os"
	"reflect"
	"syscall"
)

// Control messages for Generic Netlink interface

type CtrlOp struct {
	ID    uint32 `netlink:"1" type:"fixed"` // CTRL_ATTR_OP_ID
	Flags uint32 `netlink:"2" type:"fixed"` // CTRL_ATTR_OP_FLAGS
}

type CtrlMcastGroup struct {
	Name string `netlink:"1" type:"string"` // CTRL_ATTR_MCAST_GRP_NAME
	ID   uint32 `netlink:"2" type:"fixed"`  // CTRL_ATTR_MCAST_GRP_ID
}

type GenlCtrlMessage struct {
	Header      syscall.NlMsghdr // 16 bytes
	GenHeader   GenlMsghdr       // 4 bytes
	FamilyID    uint16           `netlink:"1" type:"fixed"`      // CTRL_ATTR_FAMILY_ID
	FamilyName  string           `netlink:"2" type:"string"`     // CTRL_ATTR_FAMILY_NAME
	Version     uint32           `netlink:"3" type:"fixed"`      // CTRL_ATTR_VERSION
	HdrSize     uint32           `netlink:"4" type:"fixed"`      // CTRL_ATTR_HDR_SIZE
	MaxAttr     uint32           `netlink:"5" type:"fixed"`      // CTRL_ATTR_MAXATTR
	Ops         []CtrlOp         `netlink:"6" type:"nestedlist"` // CTRL_ATTR_OPS
	McastGroups []CtrlMcastGroup `netlink:"7" type:"nestedlist"` // CTRL_ATTR_MCAST_GROUPS
}

const (
	CTRL_CMD_UNSPEC = iota
	CTRL_CMD_NEWFAMILY
	CTRL_CMD_DELFAMILY
	CTRL_CMD_GETFAMILY
	CTRL_CMD_NEWOPS
	CTRL_CMD_DELOPS
	CTRL_CMD_GETOPS
	CTRL_CMD_NEWMCAST_GRP
	CTRL_CMD_DELMCAST_GRP
	CTRL_CMD_GETMCAST_GRP
)

const (
	CTRL_ATTR_UNSPEC = iota
	CTRL_ATTR_FAMILY_ID
	CTRL_ATTR_FAMILY_NAME
	CTRL_ATTR_VERSION
	CTRL_ATTR_HDRSIZE
	CTRL_ATTR_MAXATTR
	CTRL_ATTR_OPS
	CTRL_ATTR_MCAST_GROUPS
)

const (
	GENL_ID_CTRL      = syscall.NLMSG_MIN_TYPE
	GENL_VERSION_CTRL = 0x1
)

func MakeGenCtrlCmd(cmd uint8) (msg GenericNetlinkMessage) {
	msg.Header.Type = GENL_ID_CTRL
	msg.Header.Flags = syscall.NLM_F_REQUEST | syscall.NLM_F_DUMP
	msg.GenHeader.Command = cmd
	msg.GenHeader.Version = GENL_VERSION_CTRL
	return msg
}

func ParseGenlFamilyMessage(msg syscall.NetlinkMessage) (ParsedNetlinkMessage, error) {
	m := new(GenlCtrlMessage)
	m.Header = msg.Header
	switch m.Header.Type {
	case syscall.NLMSG_DONE:
		return nil, nil
	case syscall.NLMSG_ERROR:
		return ParseErrorMessage(msg), nil
	}
	buf := bytes.NewBuffer(msg.Data)
	binary.Read(buf, SystemEndianness, &m.GenHeader)

	// read Family attributes
	er := ReadManyAttributes(buf, m)
	return m, er
}

func GetFamilyIDs() (ids map[string]uint16, er error) {
	s, _ := DialNetlink("generic", 0)
	msg := MakeGenCtrlCmd(CTRL_CMD_GETFAMILY)
	WriteMessage(s, &msg)
	ids = make(map[string]uint16, 4)
	for {
		resp, er := ReadMessage(s)
		if er != nil {
			return nil, fmt.Errorf("error while receiving netlink message: %s", er)
		}
		parsedmsg, er := ParseGenlFamilyMessage(resp)
		if er != nil {
			return nil, fmt.Errorf("error while parsing netlink message: %s", er)
		}

		switch m := parsedmsg.(type) {
		case nil:
			return ids, nil
		case *ErrorMessage:
			return nil, fmt.Errorf("could not retrieve genl family info: %s",
				os.NewSyscallError("netlink", syscall.Errno(-m.Errno)))
		case *GenlCtrlMessage:
			ids[m.FamilyName] = m.FamilyID
		default:
			return nil, fmt.Errorf("unable to handle message of wrong type %s", reflect.TypeOf(m))
		}
	}
	panic("unreachable")
}
