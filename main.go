package main

import (
	"log"
	"net/http"
	"ssh-manage/api"
	"ssh-manage/config"
	"ssh-manage/utils"
	"ssh-manage/web"
)

func main() {
	// 初始化数据库
	err := utils.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	
	// 初始化防火墙模块
	utils.InitFirewall()
	
	// 启动Web服务
	go func() {
		http.HandleFunc("/", web.Handler)
		cfg := config.Load()
		log.Printf("Web server listening on port %s", cfg.WebPort)
		err := http.ListenAndServe(":"+cfg.WebPort, nil)
		if err != nil {
			log.Fatalf("Failed to start web server: %v", err)
		}
	}()
	
	// 启动SSH服务
	cfg := config.Load()
	err = api.StartSSHServer(cfg)
	if err != nil {
		log.Fatalf("Failed to start SSH server: %v", err)
	}
}