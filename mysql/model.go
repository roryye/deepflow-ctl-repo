package mysql

import (
	"bytes"
	"compress/zlib"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("db.mysql")

type VTapRepo struct {
	ID        int             `gorm:"primaryKey;column:id;type:int;not null" json:"ID"`
	Name      string          `gorm:"column:name;type:char(64);not null" json:"NAME"`
	Arch      string          `gorm:"column:arch;type:varchar(256);default:''" json:"ARCH"`
	OS        string          `gorm:"column:os;type:varchar(256);default:''" json:"OS"`
	Branch    string          `gorm:"column:branch;type:varchar(256);default:''" json:"BRANCH"`
	RevCount  string          `gorm:"column:rev_count;type:varchar(256);default:''" json:"REV_COUNT"`
	CommitID  string          `gorm:"column:commit_id;type:varchar(256);default:''" json:"COMMIT_ID"`
	Image     compressedBytes `gorm:"column:image;type:logblob" json:"IMAGE"`
	K8sImage  string          `gorm:"column:k8s_image;type:varchar(512);default:''" json:"K8S_IMAGE"`
	CreatedAt time.Time       `gorm:"column:created_at;type:timestamp;not null;default:CURRENT_TIMESTAMP" json:"CREATED_AT"`
	UpdatedAt time.Time       `gorm:"column:updated_at;type:timestamp;not null;default:CURRENT_TIMESTAMP" json:"UPDATED_AT"`
}

type compressedBytes []byte

// Scan scan decompress value into compressedBytes, implements sql.Scanner interface
func (c *compressedBytes) Scan(value interface{}) error {
	// decompress
	compressedData, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("failed to decompress compressedImage value:", value))
	}

	var b bytes.Buffer
	b.Write(compressedData)
	r, err := zlib.NewReader(&b)
	if err != nil {
		return err
	}
	defer r.Close()

	originData, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	*c = originData
	return nil
}

// Value return compress value, implement driver.Valuer interface
func (c compressedBytes) Value() (driver.Value, error) {
	// compress
	t1 := time.Now()
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, err := w.Write(c)
	if err != nil {
		return nil, fmt.Errorf("failed to write compressed data: %v", err)
	}
	if err = w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zlib writer: %v", err)
	}
	log.Infof("compress time comsumed: %v", time.Since(t1))
	return b.String(), nil
}

func (VTapRepo) TableName() string {
	return "vtap_repo"
}
