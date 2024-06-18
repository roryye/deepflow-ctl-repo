package mysql

import "gorm.io/gorm"

type DB struct {
	*gorm.DB
	ORGID int
	Name  string
}
