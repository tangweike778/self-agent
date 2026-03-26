package main

import (
	"fmt"
	"log"
	"net/http"
	"self-agent/config"
	"self-agent/gateway"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	gatewayObj := gateway.NewGateway()
	gatewayObj.AutoRegisterSession()
	http.Handle("/ai", gatewayObj)
	if err = gatewayObj.Init(); err != nil {
		log.Printf("init fail, gatewayObj: %v", gatewayObj)
		panic(err)
	}
	gatewayObj.Start()

	log.Printf("Server starting on port %d", cfg.Server.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), nil); err != nil {
		log.Fatal(err)
	}
}
