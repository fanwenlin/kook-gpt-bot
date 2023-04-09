package main

import (
	"context"
	"encoding/json"
	"errors"
	"kook_gpt/store"
	"net/url"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gookit/event"
	"github.com/idodo/golang-bot/kaihela/api/base"
	event2 "github.com/idodo/golang-bot/kaihela/api/base/event"
	"github.com/idodo/golang-bot/kaihela/api/helper"
	log "github.com/sirupsen/logrus"
)

type ChatGPTMsgHandler struct {
	Token     string
	BaseUrl   string
	BotID     int64
	BotRoleID int64
}

func (h *ChatGPTMsgHandler) Handle(e event.Event) error {
	err := func() error {
		if _, ok := e.Data()[base.EventDataFrameKey]; !ok {
			return errors.New("data has no frame field")
		}
		frame := e.Data()[base.EventDataFrameKey].(*event2.FrameMap)
		data, err := sonic.Marshal(frame.Data)
		if err != nil {
			return err
		}
		msgEvent := &event2.MessageKMarkdownEvent{}
		err = sonic.Unmarshal(data, msgEvent)
		log.Infof("Received json event:%+v", msgEvent)
		if err != nil {
			return err
		}

		if msgEvent.Author.Bot {
			log.Info("bot message")
			return nil
		}

		ctx := context.Background()

		if !store.CheckIdempotent(ctx, msgEvent.MsgId) {
			log.Info("multi message event")
		}

		// /api/v3/message/view?${msg_id}
		msgDetailClient := helper.NewApiHelper("/v3/message/view?msg_id="+url.QueryEscape(msgEvent.MsgId), h.Token, h.BaseUrl, "", "")
		respByte, err := msgDetailClient.Get()
		if err != nil {
			log.Error("failed to get msg detail, err:%v", err)
			return err
		}

		// log.Infof("msg detail raw:%+v", string(respByte))
		msgResp := kookApiResp[msgDetail]{}

		err = json.Unmarshal(respByte, &msgResp)
		if err != nil {
			return err
		}
		msg := msgResp.Data

		log.Infof("msg detail struct :%+v", JsonMarshal(msg))

		replyMsgID := ""
		if msg.Quote != nil {
			replyMsgID = msg.Quote.ID
			log.Info("reply msg id:%s", replyMsgID)
		} else {
			log.Info("not reply msg")
		}
		sessionID, err := store.ReceiveMessage(ctx, msg.ID, replyMsgID)
		if err != nil {
			log.Error("failed to receive message, err:%v", err)
		}

		rawContent := msgEvent.KMarkdown.RawContent
		if h.IsMentioned(msgEvent) {

			client := helper.NewApiHelper("/v3/message/create", h.Token, h.BaseUrl, "", "")
			var replyMsg string

			content := strings.Trim(rawContent, " ")[len("@ChatGPT"):]
			content = msgEvent.Author.Username + ": " + content
			answer, err := chatgptClient.ChatGPTRequest(context.Background(), content, msgEvent.AuthorId, sessionID)
			if err != nil {
				replyMsg = "internal error:" + err.Error()
			} else {
				replyMsg = answer
			}
			log.Infof("reply msg:%s", replyMsg)
			body := map[string]string{
				"channel_id": msgEvent.TargetId,
				"content":    replyMsg,
				"quote":      msgEvent.MsgId,
			}
			bodyByte, err := sonic.Marshal(body)
			if err != nil {
				return err
			}
			resp, err := client.SetBody(bodyByte).Post()
			log.Info("sent post:%s, resp:%v, err:%v", client.String(), string(resp), err)

			if err == nil {
				sendMsgResp := kookApiResp[SendMsgResp]{}
				if err = json.Unmarshal(resp, &sendMsgResp); err == nil {
					store.ReceiveMessage(ctx, sendMsgResp.Data.MsgID, msgEvent.MsgId)
				} else {
					log.Errorf("Failed to receive msg resp, err:%v", err)
				}
			} else {
				log.Errorf("Failed to get chatgpt response, err:%v", err)
			}
			return err
		}

		return nil
	}()
	if err != nil {
		log.WithError(err).Error("ChatGPTMsgHandler err")
	}
	return nil
}

func (h *ChatGPTMsgHandler) IsMentioned(msgEvent *event2.MessageKMarkdownEvent) bool {
	mentions := msgEvent.KMarkdown.MentionPart
	for _, mention := range mentions {
		// mention is a number type
		var mentionID int64
		switch mention.(type) {
		case float64:
			mentionID = int64(mention.(float64))
		case int64:
			mentionID = mention.(int64)
		case int:
			mentionID = int64(mention.(int))
		case int32:
			mentionID = int64(mention.(int32))
		case map[string]interface{}:
			mentionMap := mention.(map[string]interface{})
			mentionID, _ = strconv.ParseInt(mentionMap["id"].(string), 10, 64)
		default:
			log.Infof("invalid mention type:%T", mention)
			continue
		}

		if mentionID == h.BotID {
			return true
		}
	}

	for _, mentionRole := range msgEvent.KMarkdown.MentionRolePart {
		// mentionRole is a map
		switch mentionRole.(type) {
		case map[string]interface{}:
			mentionRoleMap := mentionRole.(map[string]interface{})
			if mentionRoleMap["role_id"] != nil && int64(mentionRoleMap["role_id"].(float64)) == h.BotRoleID {
				return true
			} else if mentionRoleMap["name"].(string) == "ChatGPT" {
				return true
			}
		}
	}
	return false
}

/*
	{
	    "id": "058a5f2d-329f-******-f2e156148716",
	    "type": 9,
	    "content": "(met)1780***(met) (met)hell(met) (met)all(met) (rol)702(rol) (rol)711(rol)",
	    "mention": ["1780***"],
	    "mention_all": true,
	    "mention_roles": [],
	    "mention_here": false,
	    "embeds": [],
	    "attachments": null,
	    "create_at": 1614922734322,
	    "updated_at": 0,
	    "reactions": [],
	    "author": {
	      "id": "1780***",
	      "username": "**",
	      "identify_num": "5788",
	      "online": false,
	      "os": "Websocket",
	      "status": 1,
	      "avatar": "**",
	      "vip_avatar": "**",
	      "banner": "",
	      "nickname": "**",
	      "roles": [],
	      "is_vip": true,
	      "bot": false
	    },
	    "image_name": "",
	    "read_status": false,
	    "quote": null,
	    "mention_info": {
	      "mention_part": [],
	      "mention_role_part": []
	    }
	  }
*/

type kookApiResp[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// translate to go struct
type msgDetail struct {
	ID           string `json:"id"`
	Type         int    `json:"type"`
	Content      string `json:"content"`
	Mention      []string
	MentionAll   bool `json:"mention_all"`
	MentionRoles []string
	MentionHere  bool `json:"mention_here"`
	Embeds       []interface{}
	Attachments  interface{}
	CreateAt     int64 `json:"create_at"`
	UpdatedAt    int64 `json:"updated_at"`
	Reactions    []interface{}
	Author       struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		IdentifyNum string `json:"identify_num"`
		Online      bool   `json:"online"`
		Os          string `json:"os"`
		Status      int    `json:"status"`
		Avatar      string `json:"avatar"`
		VipAvatar   string `json:"vip_avatar"`
		Banner      string `json:"banner"`
		Nickname    string `json:"nickname"`
		Roles       []interface{}
		IsVip       bool `json:"is_vip"`
		Bot         bool `json:"bot"`
	} `json:"author"`
	ImageName   string `json:"image_name"`
	ReadStatus  bool   `json:"read_status"`
	Quote       *Quote `json:"quote"`
	MentionInfo struct {
		MentionPart     []interface{} `json:"mention_part"`
		MentionRolePart []interface{} `json:"mention_role_part"`
	} `json:"mention_info"`
}

type Quote struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	Content   string `json:"content"`
	CreateAt  int64  `json:"create_at"`
	UpdatedAt int64  `json:"updated_at"`
	Author    struct {
		ID             string `json:"id"`
		Username       string `json:"username"`
		IdentifyNum    string `json:"identify_num"`
		Online         bool   `json:"online"`
		Os             string `json:"os"`
		Status         int    `json:"status"`
		Avatar         string `json:"avatar"`
		VipAvatar      string `json:"vip_avatar"`
		Nickname       string `json:"nickname"`
		Roles          []int  `json:"roles"`
		IsVip          bool   `json:"is_vip"`
		Bot            bool   `json:"bot"`
		MobileVerified bool   `json:"mobile_verified"`
		JoinedAt       int64  `json:"joined_at"`
		ActiveTime     int64  `json:"active_time"`
	} `json:"author"`
}

// {"msg_id":"251bb967-ccbf-498c-a726-de40e4a9d1c1","msg_timestamp":1680703702253,"nonce":"","not_permissions_mention":[]}}
type SendMsgResp struct {
	MsgID                 string        `json:"msg_id"`
	MsgTimestamp          int64         `json:"msg_timestamp"`
	Nonce                 string        `json:"nonce"`
	NotPermissionsMention []interface{} `json:"not_permissions_mention"`
}

func JsonMarshal(i interface{}) string {
	bytes, _ := json.Marshal(i)
	return string(bytes)
}
