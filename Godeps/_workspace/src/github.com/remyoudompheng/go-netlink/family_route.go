package netlink

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
)

func MakeRouteMessage(proto, family int) (msg RawNetlinkMessage) {
	msg.Header.Type = uint16(proto)
	msg.Header.Flags = syscall.NLM_F_DUMP | syscall.NLM_F_REQUEST
	msg.Data = make([]byte, syscall.SizeofRtGenmsg)
	msg.Data[0] = uint8(family)
	return msg
}

const (
	DEVCONF_MAX     = 29
	IPSTATS_MIB_MAX = 31
	ICMP6_MIB_MAX   = 5
)

type Inet6ProtInfo struct {
	Flags      uint32        `netlink:"1" type:"fixed"` // IFLA_INET6_FLAGS
	Conf       [29]uint32    `netlink:"2" type:"fixed"` // IFLA_INET6_CONF (DEVCONF_MAX=29)
	Stats      [31]uint64    `netlink:"3" type:"fixed"` // IFLA_INET6_STATS
	Mcast      []byte        `netlink:"4" type:"bytes"` // IFLA_INET6_MCAST
	CacheInfo  LinkCacheInfo `netlink:"5" type:"fixed"` // IFLA_INET6_CACHEINFO
	Icmp6Stats [5]uint64     `netlink:"6" type:"fixed"` // IFLA_INET6_ICMP6STATS
}

type ProtInfo struct {
	Inet6Info Inet6ProtInfo `netlink:"28" type:"nested"` // AF_INET6
}

type InetData struct {
	Conf [25]uint32 `netlink:"1" type:"fixed"` // IFLA_INET_CONF
}

type AFSpec struct {
	Local []byte        `netlink:"1" type:"bytes"`   // AF_LOCAL
	Inet  InetData      `netlink:"2" type:"nested"`  // AF_INET
	Inet6 Inet6ProtInfo `netlink:"10" type:"nested"` // AF_INET6
}

// Link messages for family NETLINK_ROUTE
type RouteLinkMessage struct {
	Header syscall.NlMsghdr
	IfInfo syscall.IfInfomsg
	// attributes
	Address   net.HardwareAddr `netlink:"1" type:"bytes"`   // IFLA_ADDRESS
	BcastAddr net.HardwareAddr `netlink:"2" type:"bytes"`   // IFLA_BROADCAST
	Ifname    string           `netlink:"3" type:"string"`  // IFLA_IFNAME
	MTU       uint32           `netlink:"4" type:"fixed"`   // IFLA_MTU
	LinkType  int32            `netlink:"5" type:"fixed"`   // IFLA_LINK
	QDisc     string           `netlink:"6" type:"string"`  // IFLA_QDISC
	Stats     LinkStats        `netlink:"7" type:"fixed"`   // IFLA_STATS
	Master    uint32           `netlink:"10" type:"fixed"`  // IFLA_MASTER
	ProtInfo  []byte           `netlink:"12" type:"bytes"`  // IFLA_PROTINFO
	TxQLen    uint32           `netlink:"13" type:"fixed"`  // IFLA_TXQLEN
	Map       IfMap            `netlink:"14" type:"fixed"`  // IFLA_MAP
	Weight    uint32           `netlink:"15" type:"fixed"`  // IFLA_WEIGHT
	OperState uint8            `netlink:"16" type:"fixed"`  // IFLA_OPERSTATE
	LinkMode  uint8            `netlink:"17" type:"fixed"`  // IFLA_LINKMODE
	Ifalias   string           `netlink:"20" type:"string"` // IFLA_IFALIAS
	NumVF     uint32           `netlink:"21" type:"fixed"`  // IFLA_NUM_VF
	Stats64   LinkStats64      `netlink:"23" type:"fixed"`  // IFLA_STATS64
	AFSpec    AFSpec           `netlink:"26" type:"nested"` // IFLA_AF_SPEC
	Group     uint32           `netlink:"27" type:"fixed"`  // IFLA_GROUP
}

const (
	IFLA_NUM_VF  = 0x15
	IFLA_STATS64 = 0x17
	IFLA_AF_SPEC = 0x1a
	IFLA_GROUP   = 0x1b
)

func ParseRouteLinkMessage(msg syscall.NetlinkMessage) (ParsedNetlinkMessage, error) {
	m := new(RouteLinkMessage)
	m.Header = msg.Header
	buf := bytes.NewBuffer(msg.Data)
	binary.Read(buf, SystemEndianness, &m.IfInfo)
	// read link attributes
	er := ReadManyAttributes(buf, m)
	return m, er
}

// address messages
type RouteAddrMessage struct {
	Header syscall.NlMsghdr
	IfAddr syscall.IfAddrmsg
	// attributes
	Address   net.IP        `netlink:"1" type:"bytes"`  // IFA_ADDRESS
	Local     net.IP        `netlink:"2" type:"bytes"`  // IFA_LOCAL
	Label     string        `netlink:"3" type:"string"` // IFA_LABEL
	Broadcast net.IP        `netlink:"4" type:"bytes"`  // IFA_BROADCAST
	Anycast   net.IP        `netlink:"5" type:"bytes"`  // IFA_ANYCAST
	Cacheinfo AddrCacheInfo `netlink:"6" type:"fixed"`  // IFA_CACHEINFO
	Multicast net.IP        `netlink:"7" type:"bytes"`  // IFA_MULTICAST
}

// Parse address messages for family NETLINK_ROUTE
func ParseRouteAddrMessage(msg syscall.NetlinkMessage) (ParsedNetlinkMessage, error) {
	m := new(RouteAddrMessage)
	m.Header = msg.Header
	buf := bytes.NewBuffer(msg.Data)

	binary.Read(buf, SystemEndianness, &m.IfAddr)
	// read Address attributes
	er := ReadManyAttributes(buf, m)
	return m, er
}

func ParseRouteMessage(msg syscall.NetlinkMessage) (ParsedNetlinkMessage, error) {
	switch msg.Header.Type {
	case syscall.RTM_NEWADDR, syscall.RTM_GETADDR, syscall.RTM_DELADDR:
		return ParseRouteAddrMessage(msg)
	case syscall.RTM_NEWLINK, syscall.RTM_GETLINK, syscall.RTM_DELLINK:
		return ParseRouteLinkMessage(msg)
	}
	return nil, fmt.Errorf("Invalid Route netlink message type: %d", msg.Header.Type)
}
