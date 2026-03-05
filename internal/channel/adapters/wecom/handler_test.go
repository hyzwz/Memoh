package wecom

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestVerifyURL(t *testing.T) {
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	aesKey, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		t.Fatal(err)
	}
	token := "testtoken"
	corpID := "ww1234567890"

	// Encrypt a known echostr.
	echoStr := "1234567890"
	encrypted, err := Encrypt(aesKey, echoStr, corpID)
	if err != nil {
		t.Fatal(err)
	}

	timestamp := "1409659813"
	nonce := "1372623149"
	sig := CalcSignature(token, timestamp, nonce, encrypted)

	h, err := NewWebhookHandler(token, encodingAESKey, corpID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/callback?msg_signature="+sig+"&timestamp="+timestamp+"&nonce="+nonce+"&echostr="+url.QueryEscape(encrypted), nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if w.Body.String() != echoStr {
		t.Fatalf("body = %q, want %q", w.Body.String(), echoStr)
	}
}

func TestReceiveMessage(t *testing.T) {
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	aesKey, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		t.Fatal(err)
	}
	token := "testtoken"
	corpID := "ww1234567890"

	// Build XML message content.
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

	// Build POST body.
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

	var received *WeComMessage
	h, err := NewWebhookHandler(token, encodingAESKey, corpID)
	if err != nil {
		t.Fatal(err)
	}
	h.OnMessage(func(msg *WeComMessage) {
		received = msg
	})

	req := httptest.NewRequest(http.MethodPost, "/callback?msg_signature="+sig+"&timestamp="+timestamp+"&nonce="+nonce, strings.NewReader(string(postBody)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if received == nil {
		t.Fatal("message handler was not called")
	}
	if received.FromUserName != "zhangsan" {
		t.Fatalf("FromUserName = %q", received.FromUserName)
	}
	if received.Content != "Hello WeCom!" {
		t.Fatalf("Content = %q", received.Content)
	}
	if received.MsgType != "text" {
		t.Fatalf("MsgType = %q", received.MsgType)
	}
}

func TestNewWebhookHandlerInvalidKey(t *testing.T) {
	_, err := NewWebhookHandler("token", "tooshort", "ww123")
	if err == nil {
		t.Fatal("expected error for invalid encodingAESKey")
	}
}

func TestReceiveMessageInvalidSignature(t *testing.T) {
	h, err := NewWebhookHandler("token", "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG", "ww123")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/callback?msg_signature=invalid&timestamp=123&nonce=456", strings.NewReader("<xml><Encrypt>bad</Encrypt></xml>"))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}
