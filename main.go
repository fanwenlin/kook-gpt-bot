package main

import (
	"fmt"
	"io"
	"kook_gpt/chatgpt"
	"kook_gpt/conf"
	"net/http"

	"github.com/idodo/golang-bot/kaihela/api/base"
	"github.com/idodo/golang-bot/kaihela/example/handler"
	log "github.com/sirupsen/logrus"
)

var chatgptClient *chatgpt.ChatGPTClient

func main() {

	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.InfoLevel)

	chatgptClient = chatgpt.NewChatGPTClient(conf.GetChatGPTConfig().APIKey)
	session := base.NewWebhookSession(conf.GetKookConfig().EncryptKey, conf.GetKookConfig().VerifyToken, 1)

	session.On(base.EventReceiveFrame, &handler.ReceiveFrameHandler{})
	session.On("GROUP_9", &ChatGPTMsgHandler{
		Token:     conf.GetKookConfig().Token,
		BaseUrl:   conf.GetKookConfig().BaseUrl,
		BotID:     conf.GetKookConfig().BotID,
		BotRoleID: conf.GetKookConfig().BotRoleID,
	})

	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "application/json")
		defer req.Body.Close()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.WithError(err).Error("Read req body error")
			return
		}

		err, resData := session.ReceiveData(body)
		if err != nil {
			log.WithError(err).Error("handle req err")
		}
		resp.Write(resData)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", conf.GetKookConfig().BotServePort), nil))
}
