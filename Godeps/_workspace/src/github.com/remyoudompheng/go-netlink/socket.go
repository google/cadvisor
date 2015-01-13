package netlink

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"syscall"
)

var netlinkFamilies = map[uint16]string{
	syscall.NETLINK_GENERIC:   "generic",
	syscall.NETLINK_ROUTE:     "route",
	syscall.NETLINK_CONNECTOR: "conn",
}

type NetlinkConn struct {
	fd     int
	family uint16
	addr   syscall.SockaddrNetlink
	rbuf   *bufio.Reader
}

type NetlinkAddr struct {
	family uint16
}

func (addr NetlinkAddr) Network() string {
	return netlinkFamilies[addr.family]
}

func (addr NetlinkAddr) String() string {
	return "netlink:" + netlinkFamilies[addr.family]
}

func DialNetlink(family string, mask uint32) (conn *NetlinkConn, er error) {
	var (
		fd       int
		errno    error
		familyno uint16
	)
	switch family {
	case "generic":
		familyno = syscall.NETLINK_GENERIC
	case "route":
		familyno = syscall.NETLINK_ROUTE
	case "conn":
		familyno = syscall.NETLINK_CONNECTOR
	default:
		fd = 0
		er = fmt.Errorf("Invalid netlink family: %s", family)
		return nil, er
	}
	// Create socket
	fd, errno = syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, int(familyno))
	if errno != nil {
		er = os.NewSyscallError("socket", errno)
		return nil, fmt.Errorf("Cannot create netlink socket: %s", er)
	}
	conn = new(NetlinkConn)
	conn.fd = fd
	conn.family = familyno
	conn.addr.Family = syscall.AF_NETLINK
	conn.addr.Pid = 0
	conn.addr.Groups = mask
	conn.rbuf = bufio.NewReader(conn)
	errno = syscall.Bind(fd, &conn.addr)
	if errno != nil {
		er = os.NewSyscallError("bind", errno)
		syscall.Close(fd)
		return nil, fmt.Errorf("Cannot bind netlink socket: %s", er)
	}
	return conn, nil
}

// net.Conn interface implementation
func (s NetlinkConn) Read(b []byte) (n int, err error) {
	nr, _, e := syscall.Recvfrom(s.fd, b, 0)
	return nr, os.NewSyscallError("recvfrom", e)
}

func (s NetlinkConn) Write(b []byte) (n int, err error) {
	e := syscall.Sendto(s.fd, b, 0, &s.addr)
	return len(b), os.NewSyscallError("sendto", e)
}

func (s NetlinkConn) Close() error {
	e := syscall.Close(s.fd)
	return os.NewSyscallError("close", e)
}

func (s NetlinkConn) LocalAddr() net.Addr {
	return NetlinkAddr{s.family}
}

func (s NetlinkConn) RemoteAddr() net.Addr {
	return NetlinkAddr{s.family}
}

// from <linux/socket.h>
const (
	SOL_NETLINK = 270
)

// Joins a multicast group
func (s NetlinkConn) JoinGroup(grp int) error {
	errno := syscall.SetsockoptInt(s.fd, SOL_NETLINK, syscall.NETLINK_ADD_MEMBERSHIP, grp)
	if errno != nil {
		return os.NewSyscallError("setsockopt", errno)
	}
	return nil
}

// Leaves a multicast group
func (s NetlinkConn) LeaveGroup(grp int) error {
	errno := syscall.SetsockoptInt(s.fd, SOL_NETLINK, syscall.NETLINK_DROP_MEMBERSHIP, grp)
	if errno != nil {
		return os.NewSyscallError("setsockopt", errno)
	}
	return nil
}
