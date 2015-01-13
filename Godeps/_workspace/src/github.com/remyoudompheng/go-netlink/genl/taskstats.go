package genl

import (
	"bytes"
	"encoding/binary"
	"github.com/remyoudompheng/go-netlink"
	"syscall"
)

const (
	TASKSTATS_GENL_NAME    = "TASKSTATS"
	TASKSTATS_GENL_VERSION = 0x1
)

type GenlTaskstatsMessage struct {
	Header    syscall.NlMsghdr   // 16 bytes
	GenHeader netlink.GenlMsghdr // 4 bytes
	Pid       uint32             `netlink:"1" type:"fixed"` // TASKSTATS_TYPE_PID
	TGid      uint32             `netlink:"2" type:"fixed"` // TASKSTATS_TYPE_TGID
	Stats     TaskStats          `netlink:"3" type:"fixed"` // TASKSTATS_TYPE_STATS
	AggrPid   bool               `netlink:"4" type:"none"`  // TASKSTATS_TYPE_AGGR_PID
	AggrTGid  bool               `netlink:"5" type:"none"`  // TASKSTATS_TYPE_AGGR_TGID
}

func ParseGenlTaskstatsMsg(msg syscall.NetlinkMessage) (netlink.ParsedNetlinkMessage, error) {
	m := new(GenlTaskstatsMessage)
	m.Header = msg.Header
	switch m.Header.Type {
	case syscall.NLMSG_DONE:
		return nil, nil
	case syscall.NLMSG_ERROR:
		return netlink.ParseErrorMessage(msg), nil
	}
	buf := bytes.NewBuffer(msg.Data)
	binary.Read(buf, netlink.SystemEndianness, &m.GenHeader)
	er := netlink.ReadManyAttributes(buf, m)
	return m, er
}
