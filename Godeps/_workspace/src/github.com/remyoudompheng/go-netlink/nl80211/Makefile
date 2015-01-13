include $(GOROOT)/src/Make.inc

TARG=netlink/nl80211
GOFILES=\
	commands.go\
	policies.go\
	nl80211_defs.go\

CLEANFILES+=nl80211_defs.go

include $(GOROOT)/src/Make.pkg

nl80211_defs.go: godefs_input.sh
	sh godefs_input.sh | godefs -gnl80211 - | gofmt > nl80211_defs.go

