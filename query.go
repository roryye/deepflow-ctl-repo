package main

import "github.com/deepflowio/deepflow/server/controller/db/mysql"

type Querier interface {
	GetAgentDispatcher(db *mysql.DB, domainPrefix string, dataDuration int) (map[string]int64, error)
}
