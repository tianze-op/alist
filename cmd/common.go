package cmd

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/alist-org/alist/v3/internal2/bootstrap"
	"github.com/alist-org/alist/v3/internal2/bootstrap/data"
	"github.com/alist-org/alist/v3/internal2/db"
	"github.com/alist-org/alist/v3/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func Init() {
	bootstrap.InitConfig()
	bootstrap.Log()
	bootstrap.InitDB()
	data.InitData()
	bootstrap.InitIndex()
}

func Release() {
	db.Close()
}

var pid = -1
var pidFile string

func initDaemon() {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exPath := filepath.Dir(ex)
	_ = os.MkdirAll(filepath.Join(exPath, "daemon"), 0700)
	pidFile = filepath.Join(exPath, "daemon/pid")
	if utils.Exists(pidFile) {
		bytes, err := os.ReadFile(pidFile)
		if err != nil {
			log.Fatal("failed to read pid file", err)
		}
		id, err := strconv.Atoi(string(bytes))
		if err != nil {
			log.Fatal("failed to parse pid data", err)
		}
		pid = id
	}
}
