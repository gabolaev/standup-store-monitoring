package main

import (
	"os"
	"os/signal"
	"stand-up-store-monitoring/internal/monitoring"
	"stand-up-store-monitoring/pkg/logger"
	"strconv"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
)

var (
	envToken         = os.Getenv("TOKEN")
	envChatID        = os.Getenv("CHAT_ID")
	envCheckInterval = os.Getenv("CHECK_INTERVAL")
)

func main() {
	log, err := logger.New()
	if err != nil {
		panic(err)
	}

	token := envToken
	chatID, err := strconv.ParseInt(envChatID, 10, 64)
	if err != nil {
		log.Fatal("unable to parse chat-id", zap.Error(err))
	}
	checkInterval, err := time.ParseDuration(envCheckInterval)
	if err != nil {
		log.Fatal("unable to parse check-interval", zap.Error(err))
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal("unable to init botapi client", zap.Error(err))
	}

	log.Info("Authorized", zap.String("account", bot.Self.UserName))

	shutdownChan := make(chan os.Signal)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	watcher := monitoring.NewWatcher(log, checkInterval)
	for newEvent := range watcher.Watch(shutdownChan) {
		for {
			text := newEvent.BuildMessage()
			msg := tgbotapi.MessageConfig{
				BaseChat:              tgbotapi.BaseChat{ChatID: chatID},
				Text:                  text,
				ParseMode:             "MarkdownV2",
				DisableWebPagePreview: false,
			}
			_, err := bot.Send(msg)
			if err != nil {
				log.Error("unable to send message", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			time.Sleep(2 * time.Second)
			log.Info("message sent", zap.String("message raw text", newEvent.BuildMessage()))
			break
		}
	}
}
