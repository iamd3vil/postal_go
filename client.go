package postal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/smtppool"
)

const (
	HdrContentType             = "Content-Type"
	HdrContentTransferEncoding = "Content-Transfer-Encoding"
	HdrContentDisposition      = "Content-Disposition"
	ContentTypeOctetStream     = "application/octet-stream"
	HdrContentID               = "Content-ID"
	contentEncBase64           = "base64"
)

type Message struct {
	To          []string
	From        string
	Sender      string
	Subject     string
	ReplyTo     []string
	Cc          []string
	Bcc         []string
	PlainBody   string
	HTMLBody    string
	Headers     textproto.MIMEHeader
	Attachments []Attachment
}

func (m *Message) Attach(r io.Reader, filename string, contentType string) (Attachment, error) {
	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, r); err != nil {
		return Attachment{}, err
	}

	at := Attachment{
		Filename: filename,
		Header:   textproto.MIMEHeader{},
		Content:  buffer.Bytes(),
	}

	if contentType != "" {
		at.Header.Set(HdrContentType, contentType)
	} else {
		at.Header.Set(HdrContentType, ContentTypeOctetStream)
	}

	at.Header.Set(HdrContentDisposition, fmt.Sprintf("attachment;\r\n filename=\"%s\"", filename))
	at.Header.Set(HdrContentID, fmt.Sprintf("<%s>", filename))
	at.Header.Set(HdrContentTransferEncoding, contentEncBase64)
	m.Attachments = append(m.Attachments, at)
	return at, nil
}

func (m *Message) AttachFile(filename string) (Attachment, error) {
	f, err := os.Open(filename)
	if err != nil {
		return Attachment{}, err
	}
	defer f.Close()

	ct := mime.TypeByExtension(filepath.Ext(filename))
	basename := filepath.Base(filename)
	return m.Attach(f, basename, ct)
}

type Attachment struct {
	Filename    string
	Header      textproto.MIMEHeader
	Content     []byte
	HTMLRelated bool
}

type ResponseMessage struct {
	ID    int64  `json:"id"`
	Token string `json:"token"`
}

type Response struct {
	MessageID string                     `json:"message_id"`
	Messages  map[string]ResponseMessage `json:"messages"`
}

type request struct {
	From   string   `json:"mail_from"`
	To     []string `json:"rcpt_to"`
	Data   string   `json:"data"`
	Bounce bool     `json:"bounce"`
}

type response struct {
	Status string   `json:"status"`
	Time   float64  `json:"time"`
	Data   Response `json:"data"`
}

type Client interface {
	SendMessage(Message) (Response, error)
}

type apiClient struct {
	baseURI    string
	token      string
	httpClient *http.Client
}

func NewAPIClient(url, token string, httpClient *http.Client) (Client, error) {
	return &apiClient{
		baseURI:    url,
		token:      token,
		httpClient: httpClient,
	}, nil
}

func (a *apiClient) SendMessage(msg Message) (Response, error) {
	attachments := make([]smtppool.Attachment, 0, len(msg.Attachments))
	for _, ac := range msg.Attachments {
		attachments = append(attachments, smtppool.Attachment{
			Filename:    ac.Filename,
			Header:      ac.Header,
			Content:     ac.Content,
			HTMLRelated: ac.HTMLRelated,
		})
	}

	// Format the message into RFC2882 message.
	email := smtppool.Email{
		ReplyTo:     msg.ReplyTo,
		From:        msg.From,
		To:          msg.To,
		Bcc:         msg.Bcc,
		Cc:          msg.Cc,
		Subject:     msg.Subject,
		Text:        []byte(msg.PlainBody),
		HTML:        []byte(msg.HTMLBody),
		Sender:      msg.Sender,
		Headers:     msg.Headers,
		Attachments: attachments,
	}

	rawMsg, err := email.Bytes()
	if err != nil {
		return Response{}, fmt.Errorf("error converting email to rfc 2882 message: %v", err)
	}

	reqJson, err := json.Marshal(request{
		From:   msg.From,
		To:     msg.To,
		Data:   base64.RawStdEncoding.EncodeToString(rawMsg),
		Bounce: false,
	})
	if err != nil {
		return Response{}, fmt.Errorf("error marshalling request to json: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/send/raw", strings.TrimSuffix(a.baseURI, "/")), bytes.NewBuffer(reqJson))
	if err != nil {
		return Response{}, fmt.Errorf("error sending request to postal: %v", err)
	}
	req.Header.Add("X-Server-API-Key", a.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("error sending request to postal: %v", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("error reading body from postal response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("error sending message to postal, status code: %d, error: %s", resp.StatusCode, body)
	}

	r := response{}
	if err := json.Unmarshal(body, &r); err != nil {
		return Response{}, fmt.Errorf("error unmarshalling json from postal response: %v", err)
	}

	return r.Data, nil
}
