package postal

import (
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	postalAddr  = os.Getenv("POSTAL_ADDR")
	postalToken = os.Getenv("POSTAL_TOKEN")
	fromAddr    = os.Getenv("POSTAL_FROM")
	toAddr      = os.Getenv("POSTAL_TO")
)

func TestSend(t *testing.T) {
	client, err := NewAPIClient(postalAddr, postalToken, &http.Client{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("error creating api client: %v", err)
	}

	msg := Message{
		From:      fromAddr,
		To:        []string{toAddr},
		Subject:   "Test Email",
		PlainBody: "Test Email from postal_go",
	}
	if err := msg.AttachFile("test/hello.txt"); err != nil {
		t.Fatalf("error attaching file: %s", err)
	}

	resp, err := client.SendMessage(msg)
	if err != nil {
		t.Fatalf("error from send message: %v", err)
	}
	t.Logf("resp: %v", resp)
}
