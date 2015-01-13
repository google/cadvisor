package netlink

/*
#include <sys/socket.h>
#include <linux/rtnetlink.h>
*/
import "C"

type LinkStats C.struct_rtnl_link_stats
type LinkStats64 C.struct_rtnl_link_stats64

type IfMap C.struct_rtnl_link_ifmap

type LinkCacheInfo C.struct_ifla_cacheinfo
type AddrCacheInfo C.struct_ifa_cacheinfo
