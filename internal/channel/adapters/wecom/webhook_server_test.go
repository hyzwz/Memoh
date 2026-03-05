package wecom

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/channel"
)

type fakeWebhookStore struct {
	configs []channel.ChannelConfig
}

func (f *fakeWebhookStore) ListConfigsByType(_ context.Context, _ channel.ChannelType) ([]channel.ChannelConfig, error) {
	return f.configs, nil
}

type fakeWebhookManager struct {
	gotMsg channel.InboundMessage
	gotCfg channel.ChannelConfig
	err    error
}

func (f *fakeWebhookManager) HandleInbound(_ context.Context, cfg channel.ChannelConfig, msg channel.InboundMessage) error {
	f.gotCfg = cfg
	f.gotMsg = msg
	return f.err
}

func TestWebhookServerHandlerRegister(t *testing.T) {
	h := NewWeComWebhookHandler(nil, nil, nil)
	e := echo.New()
	h.Register(e)

	routes := e.Routes()
	found := false
	for _, r := range routes {
		if strings.Contains(r.Path, "/channels/wecom/webhook") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("wecom webhook route not registered")
	}
}

func TestWebhookServerHandlerVerify(t *testing.T) {
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	aesKey, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		t.Fatal(err)
	}
	token := "testtoken"
	corpID := "ww1234567890"

	store := &fakeWebhookStore{
		configs: []channel.ChannelConfig{
			{
				ID:          "cfg-1",
				BotID:       "bot-1",
				ChannelType: Type,
				Credentials: map[string]any{
					"corpId":         corpID,
					"secret":         "sec",
					"token":          token,
					"encodingAESKey": encodingAESKey,
				},
			},
		},
	}
	manager := &fakeWebhookManager{}
	h := NewWeComWebhookHandler(nil, store, manager)

	echoStr := "hello-verify"
	encrypted, err := Encrypt(aesKey, echoStr, corpID)
	if err != nil {
		t.Fatal(err)
	}
	timestamp := "1409659813"
	nonce := "1372623149"
	sig := CalcSignature(token, timestamp, nonce, encrypted)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/channels/wecom/webhook/cfg-1?msg_signature="+sig+"&timestamp="+timestamp+"&nonce="+nonce+"&echostr="+url.QueryEscape(encrypted), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("config_id")
	c.SetParamValues("cfg-1")

	if err := h.HandleVerify(c); err != nil {
		t.Fatalf("HandleVerify: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookServerHandlerMessage(t *testing.T) {
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	aesKey, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		t.Fatal(err)
	}
	token := "testtoken"
	corpID := "ww1234567890"

	store := &fakeWebhookStore{
		configs: []channel.ChannelConfig{
			{
				ID:          "cfg-1",
				BotID:       "bot-1",
				ChannelType: Type,
				Credentials: map[string]any{
					"corpId":         corpID,
					"secret":         "sec",
					"token":          token,
					"encodingAESKey": encodingAESKey,
				},
			},
		},
	}
	manager := &fakeWebhookManager{}
	h := NewWeComWebhookHandler(nil, store, manager)

	msgXML := `<xml>
<ToUserName><![CDATA[` + corpID + `]]></ToUserName>
<FromUserName><![CDATA[zhangsan]]></FromUserName>
<CreateTime>1409659813</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[Hello WeCom!]]></Content>
<MsgId>1234567890</MsgId>
<AgentID>1000002</AgentID>
</xml>`

	encrypted, err := Encrypt(aesKey, msgXML, corpID)
	if err != nil {
		t.Fatal(err)
	}
	timestamp := "1409659813"
	nonce := "1372623149"
	sig := CalcSignature(token, timestamp, nonce, encrypted)

	type EncryptMsg struct {
		XMLName    xml.Name `xml:"xml"`
		ToUserName string
		Encrypt    string
		AgentID    string
	}
	postBody, _ := xml.Marshal(EncryptMsg{
		ToUserName: corpID,
		Encrypt:    encrypted,
		AgentID:    "1000002",
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/channels/wecom/webhook/cfg-1?msg_signature="+sig+"&timestamp="+timestamp+"&nonce="+nonce, strings.NewReader(string(postBody)))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("config_id")
	c.SetParamValues("cfg-1")

	if err := h.HandleMessage(c); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if manager.gotMsg.Message.Text != "Hello WeCom!" {
		t.Fatalf("message text = %q, want Hello WeCom!", manager.gotMsg.Message.Text)
	}
	if manager.gotMsg.Sender.SubjectID != "zhangsan" {
		t.Fatalf("sender = %q, want zhangsan", manager.gotMsg.Sender.SubjectID)
	}
}
