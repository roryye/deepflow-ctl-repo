package main

import "gitlab.yunshan.net/weiqiang/deepflow-ctl-traffic/mysql"

type Querier interface {
	GetAgentDispatcher(db *mysql.DB, domainPrefix string, dataDuration int) (map[string]int64, error)
}
