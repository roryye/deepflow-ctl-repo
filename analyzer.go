package main

import (
	"github.com/deepflowio/deepflow/server/controller/common"
	"github.com/deepflowio/deepflow/server/controller/db/mysql"
	"gorm.io/gorm"
)

// //go:generate mockgen -source=analyzer.go -destination=./mocks/mock_analyzer.go -package=mocks DB
type DB interface {
	Get() error
}

type DBInfo struct {
	AZs             []mysql.AZ
	Analyzers       []mysql.Analyzer
	AZAnalyzerConns []mysql.AZAnalyzerConnection
	VTaps           []mysql.VTap

	// get query data
	Controllers       []mysql.Controller
	AZControllerConns []mysql.AZControllerConnection
}

type AnalyzerInfo struct {
	onlyWeight                bool
	dbInfo                    *DBInfo
	regionToVTapNameToTraffic map[string]map[string]int64

	db    DB
	query Querier
}

func NewAnalyzerInfo(onlyWeight bool) *AnalyzerInfo {
	return &AnalyzerInfo{
		onlyWeight: onlyWeight,
		dbInfo:     &DBInfo{},
		query: &Query{
			onlyWeight: onlyWeight,
		},
	}
}

func (r *DBInfo) Get(db *gorm.DB) error {
	if err := mysql.Db.Find(&r.AZs).Error; err != nil {
		return err
	}
	if err := mysql.Db.Find(&r.Analyzers).Error; err != nil {
		return err
	}
	if err := mysql.Db.Find(&r.AZAnalyzerConns).Error; err != nil {
		return err
	}
	if err := mysql.Db.Where("type != ?", common.VTAP_TYPE_TUNNEL_DECAPSULATION).Find(&r.VTaps).Error; err != nil {
		return err
	}

	if err := mysql.Db.Find(&r.Controllers).Error; err != nil {
		return err
	}
	if err := mysql.Db.Find(&r.AZControllerConns).Error; err != nil {
		return err
	}
	return nil
}

func GetAZToAnalyzers(azAnalyzerConns []mysql.AZAnalyzerConnection, regionToAZLcuuids map[string][]string,
	ipToAnalyzer map[string]*mysql.Analyzer) map[string][]*mysql.Analyzer {

	azToAnalyzers := make(map[string][]*mysql.Analyzer)
	for _, conn := range azAnalyzerConns {
		if conn.AZ == "ALL" {
			if azLcuuids, ok := regionToAZLcuuids[conn.Region]; ok {
				for _, azLcuuid := range azLcuuids {
					if analyzer, ok := ipToAnalyzer[conn.AnalyzerIP]; ok {
						azToAnalyzers[azLcuuid] = append(
							azToAnalyzers[azLcuuid], analyzer,
						)
					}
				}
			}
		} else {
			if analyzer, ok := ipToAnalyzer[conn.AnalyzerIP]; ok {
				azToAnalyzers[conn.AZ] = append(azToAnalyzers[conn.AZ], analyzer)
			}
		}
	}
	return azToAnalyzers
}
