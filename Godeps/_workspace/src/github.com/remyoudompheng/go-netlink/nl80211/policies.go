package nl80211

type BSS struct {
	BSSID          bool   `netlink:"1" type:"none"`   // BSS_BSSID
	Frequency      uint32 `netlink:"2" type:"fixed"`  // BSS_FREQUENCY
	TSF            uint64 `netlink:"3" type:"fixed"`  // BSS_TSF
	BeaconInterval uint16 `netlink:"4" type:"fixed"`  // BSS_BEACON_INTERVAL
	Capability     uint16 `netlink:"5" type:"fixed"`  // BSS_CAPABILITY
	InfoElements   bool   `netlink:"6" type:"none"`   // BSS_INFORMATION_ELEMENTS
	SignalMBM      uint32 `netlink:"7" type:"fixed"`  // BSS_SIGNAL_MBM
	SignalUnspec   uint8  `netlink:"8" type:"fixed"`  // BSS_SIGNAL_UNSPEC
	Status         uint32 `netlink:"9" type:"fixed"`  // BSS_STATUS
	SeenMsAgo      uint32 `netlink:"10" type:"fixed"` // BSS_SEEN_MS_AGO
	BeaconIES      bool   `netlink:"11" type:"none"`  // BSS_BEACON_IES
}

type StationInfo struct {
	InactiveTime  uint32   `netlink:"1" type:"fixed"`   // STA_INFO_INACTIVE_TIME 
	RxBytes       uint32   `netlink:"2" type:"fixed"`   // STA_INFO_RX_BYTES 
	TxBytes       uint32   `netlink:"3" type:"fixed"`   // STA_INFO_TX_BYTES 
	Llid          uint16   `netlink:"4" type:"fixed"`   // STA_INFO_LLID 
	Plid          uint16   `netlink:"5" type:"fixed"`   // STA_INFO_PLID 
	PLinkState    uint8    `netlink:"6" type:"fixed"`   // STA_INFO_PLINK_STATE 
	Signal        uint8    `netlink:"7" type:"fixed"`   // STA_INFO_SIGNAL 
	TxBitrate     RateInfo `netlink:"8" type:"nested"`  // STA_INFO_TX_BITRATE 
	RxPackets     uint32   `netlink:"9" type:"fixed"`   // STA_INFO_RX_PACKETS 
	TxPackets     uint32   `netlink:"10" type:"fixed"`  // STA_INFO_TX_PACKETS 
	TxRetries     uint32   `netlink:"11" type:"fixed"`  // STA_INFO_TX_RETRIES 
	TxFailed      uint32   `netlink:"12" type:"fixed"`  // STA_INFO_TX_FAILED 
	SignalAvg     uint8    `netlink:"13" type:"fixed"`  // STA_INFO_SIGNAL_AVG 
	RxBitrate     RateInfo `netlink:"14" type:"nested"` // STA_INFO_RX_BITRATE 
	BssParam      BSSParam `netlink:"15" type:"nested"` // STA_INFO_BSS_PARAM 
	ConnectedTime uint32   `netlink:"16" type:"fixed"`  // STA_INFO_CONNECTED_TIME 
}

type RateInfo struct {
	Bitrate           uint16 `netlink:"1" type:"fixed"` // RATE_INFO_BITRATE
	FlagsMcs          uint8  `netlink:"2" type:"fixed"` //RATE_INFO_FLAGS_MCS
	Flags_40MHz_width bool   `netlink:"3" type:"none"`  // RATE_INFO_FLAGS_40_MHZ_WIDTH
	FlagsShortGI      bool   `netlink:"4" type:"none"`  // RATE_INFO_FLAGS_SHORT_GI
}

type BSSParam struct {
	FlagsCTSProt       bool   `netlink:"1" type:"none"`  // BSS_PARAM_FLAGS_CTS_PROT
	FlagsShortPreamble bool   `netlink:"2" type:"none"`  // BSS_PARAM_FLAGS_SHORT_PREAMBLE 
	FlagsShortSlotTime bool   `netlink:"3" type:"none"`  // BSS_PARAM_FLAGS_SHORT_SLOT_TIME 
	DtimPeriod         uint8  `netlink:"4" type:"fixed"` // BSS_PARAM_FLAGS_DTIM_PERIOD 
	BeaconInterval     uint16 `netlink:"5" type:"fixed"` // BSS_PARAM_FLAGS_BEACON_INTERVAL  
}
