package protocols

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BatikanHyt/netbench/pkg/collector"
)

type smtpClient struct {
	Address string `json:"address"`
	Tls     bool   `json:"tls"`
	Auth    struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Method   string `json:"method"`
	} `json:"auth"`
	EmlFile     string            `json:"eml"`
	From        string            `json:"from"`
	To          []string          `json:"to"`
	CC          []string          `json:"cc"`
	BCC         []string          `json:"bcc"`
	Subject     string            `json:"subject"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	BodyFile    string            `json:"body_file"`
	BodyHtml    string            `json:"body_html"`
	Attachments []string          `json:"attachments"`
	Timeout     time.Duration     `json:"Timeout"`
	ReportChan  chan *collector.SmtpEntry
	initialized bool
	readSize    int64
	writeSize   int64
	Connection  *net.Conn
	data        []byte
}

func NewSmtpClient() *smtpClient {
	client := &smtpClient{
		initialized: false,
	}

	return client
}

func writePlainBodyPart(writer *bytes.Buffer, content []byte, is_multi bool, boundary *string) {
	if is_multi {
		writer.WriteString(fmt.Sprintf("--%s\n", *boundary))
		writer.WriteString("Content-Type: text/plain; charset=UTF-8\n")
		writer.WriteString("Content-Transfer-Encoding: quoted-printable\n")
		writer.WriteString("Content-Disposition: inline\n\n")
	}
	writer.Write(content)
	writer.WriteString("\n\n")
}

func writeHtmlBodyPart(writer *bytes.Buffer, html_file []byte, is_multi bool, boundary *string) {
	if is_multi {
		writer.WriteString(fmt.Sprintf("--%s\n", *boundary))
		writer.WriteString("Content-Type: text/html; charset=UTF-8\n")
		writer.WriteString("Content-Transfer-Encoding: quoted-printable\n")
		writer.WriteString("Content-Disposition: inline\n\n")
	}
	writer.Write(html_file)
	writer.WriteString("\n\n")
}

func writeAttachmentPart(writer *bytes.Buffer, attachments []string, boundary *string) {

	for _, file := range attachments {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		writer.WriteString(fmt.Sprintf("--%s\n", *boundary))
		writer.WriteString(fmt.Sprintf("Content-Type: %s\n", http.DetectContentType(content)))
		writer.WriteString("Content-Transfer-Encoding: base64\n")
		writer.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\n\n", filepath.Base(file)))
		b := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
		base64.StdEncoding.Encode(b, content)
		writer.Write(b)
		writer.WriteString(fmt.Sprintf("\n\n--%s", *boundary))
	}
	if len(attachments) > 0 {
		writer.WriteString("--")
	}
}

func (c *smtpClient) Initialize(clc *collector.StatBase) {
	sclc, _ := (*clc).(*collector.SmtpStatCollector)
	c.ReportChan = sclc.StatChannel
	var err error
	if c.EmlFile != "" {
		c.data, err = c.createMailFromEml()
	} else {
		c.data, err = c.createMailFromConf()
	}
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	c.initialized = true
}

func (c *smtpClient) createMailFromEml() ([]byte, error) {
	emlfile, err := ioutil.ReadFile(c.EmlFile)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(emlfile)
	mail, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}
	from, err := mail.Header.AddressList("From")
	if err != nil {
		return nil, err
	}
	to, err := mail.Header.AddressList("To")
	if err != nil {
		return nil, err
	}
	c.Subject = mail.Header.Get("Subject")
	if c.Subject == "" {
		return nil, errors.New("EML file does not contain Subject field")
	}
	cc, err := mail.Header.AddressList("Cc")
	if err == nil {
		for _, addr := range cc {
			c.CC = append(c.CC, addr.Address)
		}
	}
	bcc, err := mail.Header.AddressList("Bcc")
	if err == nil {
		for _, addr := range bcc {
			c.BCC = append(c.BCC, addr.Address)
		}
	}
	for _, addr := range to {
		c.To = append(c.To, addr.Address)
	}
	c.From = from[0].Address

	return emlfile, nil
}

func (c *smtpClient) createMailFromConf() ([]byte, error) {
	data := &bytes.Buffer{}

	if c.From == "" || len(c.To) == 0 || c.Subject == "" {
		return nil, errors.New("STMP requires From, To and Subject to be non empty")
	}

	data.WriteString(fmt.Sprintf("From: %s\n", c.From))
	data.WriteString(fmt.Sprintf("To: %s\n", strings.Join(c.To, ",")))
	data.WriteString(fmt.Sprintf("Subject: %s\n", c.Subject))
	if len(c.CC) > 0 {
		data.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(c.CC, ",")))
	}
	if len(c.BCC) > 0 {
		data.WriteString(fmt.Sprintf("Bcc: %s\n", strings.Join(c.BCC, ",")))
	}
	for key, value := range c.Headers {
		data.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	}
	multipart_alternative := false
	if c.BodyHtml != "" && (c.BodyFile != "" || c.Body != "") {
		multipart_alternative = true
	}
	data.WriteString("MIME-Version: 1.0\n")
	writer := multipart.NewWriter(data)
	boundary := writer.Boundary()
	if len(c.Attachments) > 0 {
		data.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\n\n", boundary))
	} else {
		if multipart_alternative {
			data.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\n\n", boundary))
		}
	}
	if c.BodyFile != "" {
		content, err := ioutil.ReadFile(c.BodyFile)
		if err != nil {
			fmt.Printf("Error reading body %s \n", err)
		} else {
			writePlainBodyPart(data, content, multipart_alternative, &boundary)
		}
	} else {
		writePlainBodyPart(data, []byte(c.Body), multipart_alternative, &boundary)
	}
	if c.BodyHtml != "" {
		content, err := ioutil.ReadFile(c.BodyHtml)
		if err != nil {
			fmt.Printf("Error reading html body %s \n", err)
		} else {
			writeHtmlBodyPart(data, content, multipart_alternative, &boundary)
		}
	}
	writeAttachmentPart(data, c.Attachments, &boundary)
	return data.Bytes(), nil
}

func getSmtpErrorCode(err error) int {
	if e, ok := err.(*textproto.Error); ok {
		return e.Code
	}
	return 1000
}

func (c *smtpClient) sendStat(code int, dur time.Duration) {
	stat := &collector.SmtpEntry{
		ResponseCode: code,
		WriteSize:    c.writeSize,
		ReadSize:     c.readSize,
		Duration:     dur,
	}
	c.ReportChan <- stat
}

func (c *smtpClient) StartBenchmark() {
	if !c.initialized {
		fmt.Println("SMTP not initialized correctly!")
		return
	}
	start := time.Now()
	code := 250
	conn, err := c.initializeConnection()
	if err != nil {
		code = getSmtpErrorCode(err)
		elapsed := time.Since(start)
		c.sendStat(code, elapsed)
		fmt.Printf("Error initializing the connection")
		return
	}
	cc, err := conn.Data()
	if err != nil {
		code = getSmtpErrorCode(err)
		elapsed := time.Since(start)
		c.sendStat(code, elapsed)
		fmt.Printf("Error data %s\n", err)
		return
	}
	cc.Write(c.data)
	cc.Close()

	err = conn.Quit()
	if err != nil {
		code = getSmtpErrorCode(err)
	}

	elapsed := time.Since(start)
	c.sendStat(code, elapsed)
}

func (c *smtpClient) initializeConnection() (*smtp.Client, error) {
	ctx := context.Background()
	conT, err := DialContextWithBytesTracked(ctx, "tcp", c.Address, &c.readSize, &c.writeSize)
	if err != nil {
		fmt.Printf("Error dial context: %s\n", err)
		return nil, err
	}
	conn, err := smtp.NewClient(conT, c.Address)
	if err != nil {
		fmt.Printf("Error initializng smtp client. Error: %s\n", err)
		return nil, err
	}
	if c.Tls {
		conn.StartTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
	}

	switch c.Auth.Method {
	case "CRAM":
		auth := smtp.CRAMMD5Auth(c.Auth.Username, c.Auth.Password)
		err := conn.Auth(auth)
		if err != nil {
			fmt.Printf("Error in auth: %s \n", err)
			return nil, err
		}
	case "PLAIN":
		auth := smtp.PlainAuth("", c.Auth.Username, c.Auth.Password, c.Address)
		err := conn.Auth(auth)
		if err != nil {
			fmt.Printf("Error in auth: %s \n", err)
			return nil, err
		}
	}
	conn.Mail(c.From)
	uniq_recp := make(map[string]bool)
	for _, arr := range [][]string{c.To, c.CC, c.BCC} {
		for _, elem := range arr {
			uniq_recp[elem] = true
		}
	}
	for key := range uniq_recp {
		err := conn.Rcpt(key)
		if err != nil {
			fmt.Printf("Error adding recepient %s \n", err)
		}
	}

	return conn, nil
}
