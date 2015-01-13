#!/bin/bash

echo "#include <linux/nl80211.h>"
sed -r '
  s/^\s*([A-Z_]*)NL80211_/$\1/
  s/enum [a-z0-9_]*/enum/
  s/struct nl/struct $nl/
  s/^#define.*//
  s/.*__LINUX_NL80211_H.*//
  ' /usr/include/linux/nl80211.h

