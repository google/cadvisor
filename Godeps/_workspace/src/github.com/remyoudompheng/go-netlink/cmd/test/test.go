package main

import (
	"encoding/json"
	"fmt"
	"github.com/remyoudompheng/go-netlink"
	"github.com/remyoudompheng/go-netlink/genl"
	"syscall"
)

func TestRouteLink(s *netlink.NetlinkConn) {
	msg := netlink.MakeRouteMessage(syscall.RTM_GETLINK, syscall.AF_UNSPEC)
	netlink.WriteMessage(s, msg)
	for {
		resp, _ := netlink.ReadMessage(s)
		parsedmsg, er := netlink.ParseRouteMessage(resp)
		if parsedmsg == nil {
			break
		}
		msg_s, _ := json.MarshalIndent(parsedmsg, "", "  ")
		fmt.Printf("Errmsg: %#v\nLinkmsg = %s\n", er, msg_s)
	}
}

func TestRouteAddr(s *netlink.NetlinkConn) {
	msg := netlink.MakeRouteMessage(syscall.RTM_GETADDR, syscall.AF_UNSPEC)
	netlink.WriteMessage(s, msg)
	for {
		resp, _ := netlink.ReadMessage(s)
		parsedmsg, er := netlink.ParseRouteMessage(resp)
		if parsedmsg == nil {
			break
		}
		if er != nil {
			fmt.Println(resp)
		}
		msg_s, _ := json.MarshalIndent(parsedmsg, "", "  ")
		fmt.Printf("Errmsg: %#v\nAddrMsg = %s\n", er, msg_s)
	}
}

func TestGenericFamily(s *netlink.NetlinkConn) {
	msg := genl.MakeGenCtrlCmd(genl.CTRL_CMD_GETFAMILY)
	netlink.WriteMessage(s, &msg)

	for {
		resp, _ := netlink.ReadMessage(s)
		parsedmsg, _ := genl.ParseGenlFamilyMessage(resp)
		switch m := parsedmsg.(type) {
		case nil:
			return
		case netlink.ErrorMessage:
			msg_s, _ := json.MarshalIndent(m, "", "  ")
			fmt.Printf("ErrorMsg = %s\n%s\n", msg_s, syscall.Errno(-m.Errno))
			break
		default:
			msg_s, _ := json.MarshalIndent(m, "", "  ")
			fmt.Printf("GenlFamily = %s\n", msg_s)
		}
	}
}

func main() {
	s, _ := netlink.DialNetlink("route", 0)
	TestRouteLink(s)
	TestRouteAddr(s)

	// NETLINK_GENERIC tests
	s, _ = netlink.DialNetlink("generic", 0)
	fmt.Println("Testing generic family messages")
	TestGenericFamily(s)

	ids, er := genl.GetFamilyIDs()
	fmt.Printf("%#v\n%s\n", ids, er)
}
