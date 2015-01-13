package nl80211

import (
	"fmt"
	netlink "github.com/remyoudompheng/go-netlink"
	"github.com/remyoudompheng/go-netlink/genl"
)

var (
	NL80211_ID      uint16
	NL80211_VERSION = uint8(0)
)

func init() {
	ids, er := genl.GetFamilyIDs()
	if er != nil {
		panic(fmt.Sprintf("initialization error, no genl family nl80211: %s", er))
	}
	NL80211_ID = ids["nl80211"]
}

func Make80211Cmd(flags uint16, cmd uint8, arg interface{}) (msg netlink.GenericNetlinkMessage) {
	msg.Header.Type = NL80211_ID
	msg.Header.Flags = flags
	msg.GenHeader.Command = cmd
	msg.GenHeader.Version = NL80211_VERSION
	return msg
}
