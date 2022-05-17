<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>

# Postal-Go

This is a client library for sending emails through [Postal](https://github.com/postalserver/postal).

Uses the [Send a raw RFC2882 message](http://apiv1.postalserver.io/controllers/send/raw) api.

## Usage

```go
client, err := NewAPIClient(postalAddr, postalToken, &http.Client{
	Timeout: 10 * time.Second,
})
if err != nil {
	log.Fatalf("error creating api client: %v", err)
}

msg := Message{
	From:      fromAddr,
	To:        []string{toAddr},
	Subject:   "Test Email",
	PlainBody: "Test Email from postal_go",
}
if err := msg.AttachFile("test/hello.txt"); err != nil {
	log.Fatalf("error attaching file: %s", err)
}

resp, err := client.SendMessage(msg)
if err != nil {
	log.Fatalf("error from send message: %v", err)
}
```

## Docs

See the docs at [Godoc](https://pkg.go.dev/github.com/iamd3vil/postal_go).
