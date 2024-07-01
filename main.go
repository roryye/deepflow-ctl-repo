package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"gitlab.yunshan.net/weiqiang/deepflow-ctl-traffic/mysql"
	drivermysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	IMAGE_MAX_COUNT = 20
)

var (
	BranchRegex   = regexp.MustCompile(`Branch:[\s](.*)`)
	RevCountRegex = regexp.MustCompile(`RevCount:[\s](.*)`)
	CommitIDRegex = regexp.MustCompile(`CommitId:[\s](.*)`)

	log = logging.MustGetLogger("main")

	gormDB *gorm.DB

	user     = flag.String("user", "", "mysql user")
	password = flag.String("password", "", "mysql password")
	ip       = flag.String("ip", "", "mysql ip")
	port     = flag.String("port", "", "mysql port")
	apiport  = flag.Int("api-port", 8321, "api port")

	apiport  = flag.Int("api-port", 8321, "api port")
	sizeInMB = flag.Int("m", 0, "size of storage image in MB")

	arch, image, versionImage, k8sImage string
)

var format = logging.MustStringFormatter(
	`%{time:2006-01-02 15:04:05.000} %{shortfile} %{level:.4s} %{message}`,
)

func init() {
	flag.StringVar(&arch, "arch", "", "arch of deepflow-agent")
	flag.StringVar(&image, "image", "", "deepflow-agent image to upload")
	flag.StringVar(&versionImage, "version-image", "", "deepflow-agent image to get branch, rev_count and commit_id")
	flag.StringVar(&k8sImage, "k8s-image", "", "deepflow-agent Kubernetes image: if k8s-image is not empty, the image flag will be ignored.")

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter)

	flag.Parse()
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/deepflow?charset=utf8&parseTime=True&loc=Local&timeout=10s",
		*user, *password, *ip, *port)

	var err error
	gormDB, err = gorm.Open(drivermysql.New(drivermysql.Config{
		DSN:                       dsn,   // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{
		Logger: logger.New(
			l.New(os.Stdout, "\r\n", l.LstdFlags), // io writer
			logger.Config{
				SlowThreshold: time.Nanosecond, // 设置慢查询阈值为极短时间，以便实际上禁用慢查询日志
				LogLevel:      logger.Silent,   // 禁用日志记录
				Colorful:      false,           // 禁用彩色输出
			}),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
	})
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()

	r := gin.Default()
	r.POST("/v1/vtap-repo/", createVtapRepo)

	go func() {
		if err := r.Run(fmt.Sprintf(":%d", *apiport)); err != nil {
			fmt.Printf("Failed to run server: %v\n", err)
		}
	}()

	if err := createRepoAgent(); err != nil {
		log.Error()
	}
}

func createRepoAgent() error {
	execImage := image
	if versionImage != "" || len(k8sImage) != 0 {
		execImage = versionImage
	}
	agentOutput, err := getAgentOutput(execImage)
	if err != nil {
		return err
	}
	branch, revCount, commitID := getAgentInfo(agentOutput)

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	if len(k8sImage) != 0 {
		bodyWriter.WriteField("NAME", k8sImage)
	} else {
		bodyWriter.WriteField("NAME", path.Base(image))
	}
	bodyWriter.WriteField("ARCH", arch)
	bodyWriter.WriteField("BRANCH", branch)
	bodyWriter.WriteField("REV_COUNT", revCount)
	bodyWriter.WriteField("COMMIT_ID", commitID)
	bodyWriter.WriteField("K8S_IMAGE", k8sImage)
	osStr := "Linux"
	if strings.HasSuffix(image, ".exe") {
		osStr = "Windows"
	}
	bodyWriter.WriteField("OS", osStr)

	contentType := bodyWriter.FormDataContentType()
	if len(image) > 0 {
		fileWriter, err := bodyWriter.CreateFormFile("IMAGE", path.Base(image))
		f, err := os.Open(image)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err = io.Copy(fileWriter, f); err != nil {
			return err
		}
	}
	bodyWriter.Close()

	url := fmt.Sprintf("http://127.0.0.1:%d/v1/vtap-repo/", *apiport)
	err = CURLPostFormData(url, contentType, bodyBuf)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func CURLPostFormData(url, contentType string, body *bytes.Buffer) error {

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		log.Error(err)
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json, text/plain")
	req.Header.Set("X-User-Id", "1")
	req.Header.Set("X-User-Type", "1")
	req.Close = true

	return parseResponse(req)
}

func parseResponse(req *http.Request) error {
	// TODO: 通过配置文件获取API超时时间
	client := &http.Client{Timeout: time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New(fmt.Sprintf("curl (%s) failed, (%v)", req.URL, err))
	}

	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.New(fmt.Sprintf("read (%s) body failed, (%v)", req.URL, err))
	}
	log.Infof("response: %s", string(respBytes))
	return nil
}

func getAgentOutput(image string) (string, error) {
	if !path.IsAbs(image) {
		image = "./" + image
	}
	command := image + " -v"
	output, err := exec.Command("/usr/bin/bash", "-c", command).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command(%v) failed; result: %v, error:%v", command, string(output), err)
	}

	return string(output), nil
}

func getAgentInfo(s string) (branch, revCount, commitID string) {
	branchSubmatch := BranchRegex.FindStringSubmatch(s)
	if len(branchSubmatch) > 1 {
		branch = branchSubmatch[1]
	}
	revCountSubmatch := RevCountRegex.FindStringSubmatch(s)
	if len(revCountSubmatch) > 1 {
		revCount = revCountSubmatch[1]
	}
	commitIDSubmatch := CommitIDRegex.FindStringSubmatch(s)
	if len(commitIDSubmatch) > 1 {
		commitID = commitIDSubmatch[1]
	}
	return
}

func createVtapRepo(c *gin.Context) {
	vtapRepo := &mysql.VTapRepo{
		Name:     c.PostForm("NAME"),
		Arch:     c.PostForm("ARCH"),
		Branch:   c.PostForm("BRANCH"),
		RevCount: c.PostForm("REV_COUNT"),
		CommitID: c.PostForm("COMMIT_ID"),
		OS:       c.PostForm("OS"),
		K8sImage: c.PostForm("K8S_IMAGE"),
	}

	// get binary file
	if len(vtapRepo.K8sImage) == 0 {
		file, fileHeader, err := c.Request.FormFile("IMAGE")
		if err != nil {
			log.Error(err)
			return
		}
		defer file.Close()

		vtapRepo.Image = make([]byte, fileHeader.Size)
		_, err = file.Read(vtapRepo.Image)
		if err != nil {
			log.Error(err)
			return
		}
	}

	err := CreateVtapRepo(1, vtapRepo)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("create image successful")
}

func CreateVtapRepo(orgID int, vtapRepoCreate *mysql.VTapRepo) error {
	db := gormDB
	imageData := vtapRepoCreate.Image
	var vtapRepoFirst mysql.VTapRepo
	if err := db.Where("name = ?", vtapRepoCreate.Name).First(&vtapRepoFirst).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			err := fmt.Errorf("fail to query vtap_repo by name(%s), error: %s", vtapRepoCreate.Name, err)
			log.Error(err)
			return err
		}

		var count int64
		db.Model(&mysql.VTapRepo{}).Count(&count)
		if count >= IMAGE_MAX_COUNT {
			return fmt.Errorf("the number of image can not exceed %d", IMAGE_MAX_COUNT)
		}
		if sizeInMB != nil && *sizeInMB != 0 {
			vtapRepoCreate.Image = nil
		}
		if err = db.Create(&vtapRepoCreate).Error; err != nil {
			log.Error(err)
			return err
		}
		if vtapRepoCreate.Image != nil {
			return nil
		}

		imageSize := len(imageData)
		log.Infof("imageSize: %v", imageSize)
		chunkSize := *sizeInMB * 1024 * 1024
		for i := 0; i < imageSize; i += chunkSize {
			end := i + chunkSize
			if end > imageSize {
				end = imageSize
			}
			chunk := imageData[i:end]
			log.Infof("update image (%v:%v)", i, end)
			updateQuery := "UPDATE vtap_repo SET image = CONCAT(image, ?) WHERE name = ?"
			if err := db.Exec(updateQuery, chunk, vtapRepoCreate.Name).Error; err != nil {
				log.Errorf("index(%v) update error: %v", i, err)
				return err
			}
		}

		return nil
	}

	// update by name
	if err := db.Model(&mysql.VTapRepo{}).Where("name = ?", vtapRepoCreate.Name).
		Updates(vtapRepoCreate).Error; err != nil {
		log.Error(err)
		return err
	}
	return nil
}
