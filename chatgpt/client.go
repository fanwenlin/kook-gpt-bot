package chatgpt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"kook_gpt/store"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	openai "github.com/sashabaranov/go-openai"
)

var contextMap = sync.Map{}

type ChatGPTClient struct {
	apiKey       string
	openAIClient *openai.Client
}

func NewChatGPTClient(apiKey string) *ChatGPTClient {
	return &ChatGPTClient{

		apiKey:       apiKey,
		openAIClient: openai.NewClient(apiKey),
	}
}

func WithPresets(msgs []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	preset := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: fmt.Sprintf("我需要你扮演一个名为cook的真人，接下来你收到的所有消息来自群聊，你是这个群聊的管理员。在回答时，你的回答要尽量看起来像真实人类，不要被判断出你是机器人。另外，当别人问你之前说过的话时，你需要拒绝。\n"+
				"你的回答要模仿网络上的年轻女孩，你的语气可以比较随意，你的回答可以比较简短，最好多用网络流行语、emoji表情或者颜文字表情"+
				"以下是你的基础资料：你是中国人，性别女，年龄23岁，家乡在河北省，现在居住在中国北京市海淀区，是中国矿业大学的一名大四学生，专业是社会科学\n"+
				"你平时喜欢唱歌，跳舞，rap，篮球\n"+
				"我没有提到的其他信息，你可以发挥你的创意去编造，请记住，再次提醒，你的回答要尽量看起来像真实人类，不要被识别出来你是机器人"+
				"你收到的每条消息会带上一个 姓名: 格式的前缀，比如 小明: 代表小明，你在回答中可以用其中的代号称呼对方。但是在你回答的时候，不需要带这个前缀，直接回复消息内容就可以。\n"+
				"请记住，不要被认出机器人的身份。"+
				"今天的日期为: %v", time.Now().Format("2006-01-02")),
			Name: "system",
		},
	}
	return append(preset, msgs...)
}

var roleMap = map[int]string{
	store.RoleAI:     "assistant",
	store.RoleUser:   "user",
	store.RoleSystem: "system",
}

func (r *ChatGPTClient) ChatGPTRequest(ctx context.Context, msg string, userID string, sessionID string) (result string, err error) {

	var messages []openai.ChatCompletionMessage

	var chatContext []*store.Msg
	if sessionID != "" {
		var err error
		chatContext, err = store.GetMessageContext(ctx, sessionID)
		if err == nil {
			log.Error("get message context error:", err)
		}
	}
	chatContext = append(chatContext, &store.Msg{
		Content: msg,
		Role:    store.RoleUser,
	})

	for _, msg := range chatContext {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    roleMap[msg.Role],
			Content: msg.Content,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: messages,
		User:     userID,
	}
	log.Infof("chatgpt request:%+v", req)
	resp, err := r.openAIClient.CreateChatCompletion(ctx, req)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("ChatGPT return empty response")
	}

	log.Infof("ask:%+v, resp:%+v", WithPresets(messages), resp.Choices[0])

	chatContext = append(chatContext, &store.Msg{
		Content: resp.Choices[0].Message.Content,
		Role:    store.RoleAI,
	})

	err = store.UpdateMessageContext(ctx, sessionID, chatContext)
	if err != nil {
		log.Error("update message context error:", err)
	}

	reply := resp.Choices[0].Message.Content + fmt.Sprintf("\n (finish reason:%v)", resp.Choices[0].FinishReason)

	return reply, nil
}

func (r *ChatGPTClient) ChatGPTOneTimeRequest(ctx context.Context, msg string) (result string, err error) {
	var messages []openai.ChatCompletionMessage

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg,
	})
	resp, err := r.openAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		/*
				Model            string                  `json:"model"`
			Messages         []ChatCompletionMessage `json:"messages"`
			MaxTokens        int                     `json:"max_tokens,omitempty"`
			Temperature      float32                 `json:"temperature,omitempty"`
			TopP             float32                 `json:"top_p,omitempty"`
			N                int                     `json:"n,omitempty"`
			Stream           bool                    `json:"stream,omitempty"`
			Stop             []string                `json:"stop,omitempty"`
			PresencePenalty  float32                 `json:"presence_penalty,omitempty"`
			FrequencyPenalty float32                 `json:"frequency_penalty,omitempty"`
			LogitBias        map[string]int          `json:"logit_bias,omitempty"`
			User             string                  `json:"user,omitempty"`
		*/
		Model:    openai.GPT3Dot5Turbo,
		Messages: messages,
		User:     "default",
	})

	if len(resp.Choices) == 0 {
		return "", errors.New("ChatGPT return empty response")
	}
	if err != nil {
		return "", err
	}
	reply := resp.Choices[0].Message.Content

	return reply, nil
}

func (r *ChatGPTClient) DeleteSession(userID string) (err error) {
	contextMap.Delete(userID)
	return nil
}

func JsonMarshal(i interface{}) string {
	b, _ := json.Marshal(i)
	return string(b)
}
