package main

import (
	"fmt"

	"gotiktokdownloader/backend/app"
	"gotiktokdownloader/backend/logging"
)

func main() {
	logger := logging.New(logging.DEBUG)
	a := app.New(logger, func(event string, data any) {
		logger.Infof("emit %s: %v", event, data)
	})
	fmt.Println("GoTikTokDownloader Phase 1 backend OK:", a.GetAppInfo())
	fmt.Println(a.PingEvent("hello"))
}
