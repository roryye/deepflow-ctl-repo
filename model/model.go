package model

type HostVTapRebalanceResult struct {
	IP                string  `json:"IP"`
	AZ                string  `json:"AZ"`
	State             int     `json:"STATE"`
	BeforeVTapNum     int     `json:"BEFORE_VTAP_NUM"`
	AfterVTapNum      int     `json:"AFTER_VTAP_NUM"`
	SwitchVTapNum     int     `json:"SWITCH_VTAP_NUM"`
	BeforeVTapWeights float64 `json:"BEFORE_VTAP_WEIGHTS"`
	AfterVTapWeights  float64 `json:"AFTER_VTAP_WEIGHTS"`

	// debug data
	NewVTapToTraffic  map[string]int64 `json:"-"`
	DelVTapToTraffic  map[string]int64 `json:"-"`
	BeforeVTapTraffic int64            `json:"-"`
	AfterVTapTraffic  int64            `json:"-"`
}

type AZVTapRebalanceResult struct {
	TotalSwitchVTapNum int                        `json:"TOTAL_SWITCH_VTAP_NUM"`
	Details            []*HostVTapRebalanceResult `json:"DETAILS"`
}

type VTapRebalanceResult struct {
	TotalSwitchVTapNum int                        `json:"TOTAL_SWITCH_VTAP_NUM"`
	Details            []*HostVTapRebalanceResult `json:"DETAILS"`
}
