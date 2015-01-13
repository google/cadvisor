package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	netlink "github.com/remyoudompheng/go-netlink"
	"os"
	"syscall"
)

func MakeProcConnectorMsg() netlink.ConnMessage {
	var msg netlink.ConnMessage
	msg.Header.Type = syscall.NLMSG_DONE
	msg.Header.Flags = 0
	msg.Header.Pid = uint32(os.Getpid())
	msg.ConnHdr.Id = netlink.ConnMsgid{Idx: netlink.CN_IDX_PROC, Val: netlink.CN_VAL_PROC}
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.LittleEndian, int32(netlink.PROC_CN_MCAST_LISTEN))
	msg.Data = buf.Bytes()
	return msg
}

func get_events(s *netlink.NetlinkConn) {
	msg := MakeProcConnectorMsg()
	netlink.WriteMessage(s, &msg)
	for {
		resp, er := netlink.ReadMessage(s)
		if er != nil {
			fmt.Println(er)
			break
		}

		cnmsg, er := netlink.ParseConnMessage(resp)
		if er != nil {
			fmt.Println(er)
			break
		}

		msg_s, er := json.MarshalIndent(cnmsg, "", "  ")
		fmt.Printf("%s\n", msg_s)
		if resp.Header.Type == syscall.NLMSG_DONE {
			return
		}
	}
}

func main() {
	s, er := netlink.DialNetlink("conn", netlink.CN_IDX_PROC)
	if er != nil {
		fmt.Println(er)
		return
	}
	for {
		get_events(s)
	}
}
