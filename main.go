package main

import (
	"main/internal/handlers"
	"main/internal/services"
	"main/pkg/logger"
	"net/http"
)

func main() {
	logger.DInit()
	logger.DPrintf(logger.DLog, "Starting server")

	chatService := services.NewChatService()
	chatHandler := handlers.NewChatHandler(chatService)
	http.HandleFunc("/chat", chatHandler.ServeHTTP)

	findService := services.NewFindService()
	findHandler := handlers.NewFindHandler(findService)
	http.HandleFunc("/find", findHandler.ServeHTTP)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
