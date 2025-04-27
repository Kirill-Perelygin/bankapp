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

// --- CBR Key Rate ---

const cbrURL = "http://www.cbr.ru/scripts/XML_daily.asp" // URL может меняться!

// Структуры для парсинга XML ответа ЦБ РФ
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
	Value    string   `xml:"Value"` // Значение как строка "74,3410"
}

// Переменная для кеширования ставки (чтобы не дергать ЦБ каждый раз)
var cachedKeyRate struct {
	rate decimal.Decimal
	time time.Time
}
var keyRateMutex sync.Mutex

// GetCBRKeyRate получает ключевую ставку ЦБ РФ (в данном примере мы получаем курс доллара для демонстрации)
// ВАЖНО: ЦБ РФ не публикует ключевую ставку в XML_daily.asp!
// Обычно ее получают с сайта cbr.ru парсингом HTML или через другие неофициальные API.
// Для УЧЕБНЫХ ЦЕЛЕЙ здесь мы будем просто возвращать ФИКСИРОВАННОЕ значение или парсить USD как пример.
func GetCBRKeyRate() (decimal.Decimal, error) {
	keyRateMutex.Lock()
	defer keyRateMutex.Unlock()

	// Простой кеш на 1 час
	if !cachedKeyRate.rate.IsZero() && time.Since(cachedKeyRate.time) < time.Hour {
		log.Println("Using cached key rate")
		return cachedKeyRate.rate, nil
	}

	log.Println("Fetching key rate from external source (using fixed value for demo)")
	// --- ЗАГЛУШКА ---
	// Вернем фиксированное значение, т.к. получить ставку из XML_daily нельзя.
	// В реальном приложении здесь был бы парсер HTML страницы cbr.ru или другой источник.
	fixedRate := decimal.NewFromFloat(16.0) // Пример: 16.0%
	cachedKeyRate.rate = fixedRate
	cachedKeyRate.time = time.Now()
	return fixedRate, nil

	// --- Пример парсинга курса USD из XML_daily.asp (НЕ КЛЮЧЕВАЯ СТАВКА!) ---
	/*
	   resp, err := http.Get(cbrURL)
	   if err != nil {
	       return decimal.Zero, fmt.Errorf("failed to get CBR data: %w", err)
	   }
	   defer resp.Body.Close()

	   if resp.StatusCode != http.StatusOK {
	       return decimal.Zero, fmt.Errorf("failed to get CBR data: status %d", resp.StatusCode)
	   }

	   body, err := io.ReadAll(resp.Body)
	   if err != nil {
	       return decimal.Zero, fmt.Errorf("failed to read CBR response: %w", err)
	   }

	   var valCurs ValCurs
	   err = xml.Unmarshal(body, &valCurs)
	   if err != nil {
	       return decimal.Zero, fmt.Errorf("failed to parse CBR XML: %w", err)
	   }

	   for _, valute := range valCurs.Valute {
	       if valute.CharCode == "USD" { // Ищем доллар США как пример
	           // Значение в формате "74,3410", заменяем запятую на точку
	           valueStr := strings.Replace(valute.Value, ",", ".", 1)
	           rate, err := decimal.NewFromString(valueStr)
	           if err != nil {
	               return decimal.Zero, fmt.Errorf("failed to parse USD rate '%s': %w", valute.Value, err)
	           }
	           // Делим на номинал, если он не 1
	           if valute.Nominal > 1 {
	               rate = rate.Div(decimal.NewFromInt(int64(valute.Nominal)))
	           }
	           log.Printf("Fetched USD rate from CBR: %s", rate.String())
	           cachedKeyRate.rate = rate // Кешируем полученный курс USD
	           cachedKeyRate.time = time.Now()
	           return rate, nil
	       }
	   }

	   return decimal.Zero, fmt.Errorf("USD rate not found in CBR response")
	*/
}

// --- SMTP Email Notifications ---

// Конфигурация SMTP (лучше выносить в переменные окружения или конфиг файл)
var smtpConfig = struct {
	Host     string
	Port     int
	Username string
	Password string // В реальном приложении используйте App Password или OAuth
	From     string
}{
	Host:     "smtp.example.com",       // Замените на ваш SMTP хост
	Port:     587,                      // Обычно 587 (TLS) или 465 (SSL)
	Username: "your_email@example.com", // Замените
	Password: "your_password",          // Замените (или используйте App Password)
	From:     "bankapp@example.com",    // Адрес отправителя
}

// SendEmailNotification отправляет простое уведомление
func SendEmailNotification(to, subject, body string) error {
	if smtpConfig.Host == "smtp.example.com" {
		log.Printf("SMTP not configured. Skipping email to %s: Subject: %s", to, subject)
		return nil // Не отправляем, если дефолтные настройки
	}

	auth := smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)

	// Формируем сообщение
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
