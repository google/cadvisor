package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	netlink "github.com/remyoudompheng/go-netlink"
	"github.com/remyoudompheng/go-netlink/genl"
	"os"
	"syscall"
)

func MakeCmdMessage() (msg netlink.GenericNetlinkMessage) {
	msg.Header.Type = 23
	msg.Header.Flags = syscall.NLM_F_REQUEST
	msg.GenHeader.Command = genl.TASKSTATS_CMD_GET
	msg.GenHeader.Version = genl.TASKSTATS_GENL_VERSION
	buf := bytes.NewBuffer([]byte{})
	netlink.PutAttribute(buf, genl.TASKSTATS_CMD_ATTR_PID, uint32(os.Getpid()))
	msg.Data = buf.Bytes()
	return msg
}

func TestTaskStats(s *netlink.NetlinkConn) {
	msg := MakeCmdMessage()
	netlink.WriteMessage(s, &msg)

	for {
		resp, _ := netlink.ReadMessage(s)
		parsedmsg, _ := genl.ParseGenlTaskstatsMsg(resp)
		switch m := parsedmsg.(type) {
		case nil:
			return
		case netlink.ErrorMessage:
			msg_s, _ := json.MarshalIndent(m, "", "  ")
			fmt.Printf("ErrorMsg = %s\n%s\n", msg_s, syscall.Errno(-m.Errno))
			break
		default:
			msg_s, _ := json.MarshalIndent(m, "", "  ")
			fmt.Printf("Taskstats = %s\n", msg_s)
		}
	}
}

func main() {
	s, _ := netlink.DialNetlink("generic", 0)
	TestTaskStats(s)
}
