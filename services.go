package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/smtp"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

const cbrURL = "http://www.cbr.ru/scripts/XML_daily.asp"

type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Date    string   `xml:"Date,attr"`
	Valute  []Valute `xml:"Valute"`
}

type Valute struct {
	XMLName  xml.Name `xml:"Valute"`
	ID       string   `xml:"ID,attr"`
	NumCode  string   `xml:"NumCode"`
	CharCode string   `xml:"CharCode"`
	Nominal  int      `xml:"Nominal"`
	Name     string   `xml:"Name"`
	Value    string   `xml:"Value"`
}

var cachedKeyRate struct {
	rate decimal.Decimal
	time time.Time
}
var keyRateMutex sync.Mutex

func GetCBRKeyRate() (decimal.Decimal, error) {
	keyRateMutex.Lock()
	defer keyRateMutex.Unlock()

	if !cachedKeyRate.rate.IsZero() && time.Since(cachedKeyRate.time) < time.Hour {
		log.Println("Using cached key rate")
		return cachedKeyRate.rate, nil
	}

	log.Println("Fetching key rate from external source (using fixed value for demo)")

	fixedRate := decimal.NewFromFloat(16.0)
	cachedKeyRate.rate = fixedRate
	cachedKeyRate.time = time.Now()
	return fixedRate, nil

}

var smtpConfig = struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}{
	Host:     "smtp.example.com",
	Port:     587,
	Username: "your_email@example.com",
	Password: "your_password",
	From:     "bankapp@example.com",
}

func SendEmailNotification(to, subject, body string) error {
	if smtpConfig.Host == "smtp.example.com" {
		log.Printf("SMTP not configured. Skipping email to %s: Subject: %s", to, subject)
		return nil
	}

	auth := smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		smtpConfig.From, to, subject, body)

	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)

	err := smtp.SendMail(addr, auth, smtpConfig.From, []string{to}, []byte(msg))
	if err != nil {
		log.Printf("Error sending email to %s: %v", to, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent successfully to %s", to)
	return nil
}
