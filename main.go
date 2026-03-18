package main

import (
	"flag"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/hekaixin66-sketch/xiaohongshuritter/configs"
)

func main() {
	var (
		headless bool
		binPath  string
		port     string
	)
	flag.BoolVar(&headless, "headless", true, "run in headless mode")
	flag.StringVar(&binPath, "bin", "", "chrome binary path")
	flag.StringVar(&port, "port", ":18060", "listen port")
	flag.Parse()

	if len(binPath) == 0 {
		binPath = os.Getenv("ROD_BROWSER_BIN")
	}

	configs.InitHeadless(headless)
	configs.SetBinPath(binPath)

	xiaohongshuService, err := NewXiaohongshuService()
	if err != nil {
		logrus.Fatalf("failed to init service: %v", err)
	}

	appServer := NewAppServer(xiaohongshuService)
	if err := appServer.Start(port); err != nil {
		logrus.Fatalf("failed to run server: %v", err)
	}
}
