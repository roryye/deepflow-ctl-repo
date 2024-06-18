package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	trafficmysql "gitlab.yunshan.net/weiqiang/deepflow-ctl-traffic/mysql"

	"github.com/op/go-logging"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	log = logging.MustGetLogger("main")

	// dsn = "root:security421@tcp(10.233.28.187:30130)/deepflow?charset=utf8&parseTime=True&loc=Local&timeout=10s"
	gormDB *gorm.DB

	user     = flag.String("user", "", "mysql user")
	password = flag.String("password", "", "mysql password")
	ip       = flag.String("ip", "", "mysql ip")
	port     = flag.String("port", "", "mysql port")

	t        customTime
	duration = flag.Int("duration", 86400, "agent traffic duration, default: one day")
)

var format = logging.MustStringFormatter(
	`%{time:2006-01-02 15:04:05.000} %{shortfile} %{level:.4s} %{message}`,
)

func init() {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter)

	flag.Parse()
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/deepflow?charset=utf8&parseTime=True&loc=Local&timeout=10s",
		*user, *password, *ip, *port)

	var err error
	gormDB, err = gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
	})
	if err != nil {
		panic(err)
	}
}

type customTime struct {
	time.Time
}

func (ct *customTime) Set(value string) error {
	parsedTime, err := time.Parse("2006-01-02 15:04:05", value)
	if err != nil {
		return err
	}
	ct.Time = parsedTime
	return nil
}

func main() {
	flag.Var(&t, "time", "specify the time (format: 'YYYY-MM-DD HH:MM:SS')")
	flag.Parse()

	data, err := NewAnalyzerInfo(false).RebalanceAnalyzerByTraffic(&trafficmysql.DB{DB: gormDB, ORGID: 1}, true, t.Time, *duration)
	if err != nil {
		log.Error(err)
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("%s", string(b))
}
