package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

type FeiShuConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

type FeiShu struct {
	config *FeiShuConfig
}


type FeiShuMessage struct {
	Timestamp string         `json:"timestamp"`
	Sign      string         `json:"sign"`
	MsgType   string         `json:"msg_type"`
	Content   FeiShuContent  `json:"content"`
}

type FeiShuContent struct {
	Post FeiShuPostContent `json:"post"`
}

type FeiShuPostContent struct {
	ZhCn FeiShuZhCnContent `json:"zh_cn"`
}

type FeiShuZhCnContent struct {
	Title   string           `json:"title"`
	Content [][]FeiShuContentItem `json:"content"`
}

type FeiShuContentItem struct {
	Tag     string `json:"tag"`
	Text    string `json:"text,omitempty"`
	Href    string `json:"href,omitempty"`
	UserID  string `json:"user_id,omitempty"`
}

func NewFeiShu() *FeiShu {
	config := &FeiShuConfig{}
	viper.UnmarshalKey("webhook.feishu", config)
	return &FeiShu{config: config}
}

func (f *FeiShu) GenSign(timestamp int64) (string, error) {
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + f.config.Secret
	h := hmac.New(sha256.New, []byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

func (f *FeiShu) Send(title, text string) error {
	timestamp := time.Now().Unix()
	sign, err := f.GenSign(timestamp)
	if err != nil {
		return fmt.Errorf("failed to generate sign: %v", err)
	}

	msg := FeiShuMessage{
		Timestamp: fmt.Sprintf("%v", timestamp),
		Sign:      sign,
		MsgType:   "post",
		Content:   FeiShuContent{
			Post: FeiShuPostContent{
				ZhCn: FeiShuZhCnContent{
					Title:   title,
					Content: [][]FeiShuContentItem{
						{
							{
							Tag:   "text",
							Text:  text,
						}},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook message: %v", err)
	}

	resp, err := http.Post(f.config.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned non-200 status: %d", resp.StatusCode)
	}
	return nil
}
