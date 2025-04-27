package main

import (
	"time"

	"github.com/shopspring/decimal" // Используем decimal для денег
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Не отправляем хеш клиенту
	CreatedAt    time.Time `json:"created_at"`
}

type Account struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Number    string          `json:"number"` // Номер счета
	Balance   decimal.Decimal `json:"balance"`
	CreatedAt time.Time       `json:"created_at"`
}

type Card struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Number      string    `json:"number"` // Номер карты
	ExpiryMonth int       `json:"expiry_month"`
	ExpiryYear  int       `json:"expiry_year"`
	CVV         string    `json:"-"` // Не отправляем CVV клиенту (в реальном приложении его вообще не хранят)
	CreatedAt   time.Time `json:"created_at"`
}

type Transaction struct {
	ID              string          `json:"id"`
	FromAccountID   string          `json:"from_account_id,omitempty"` // Может быть пустым для пополнения
	ToAccountID     string          `json:"to_account_id,omitempty"`   // Может быть пустым для списания/оплаты
	Amount          decimal.Decimal `json:"amount"`
	Timestamp       time.Time       `json:"timestamp"`
	TransactionType string          `json:"transaction_type"` // e.g., "transfer", "deposit", "payment", "loan_disbursement"
	Description     string          `json:"description,omitempty"`
}

type Loan struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	AccountID       string          `json:"account_id"` // Счет для зачисления кредита и списания платежей
	Amount          decimal.Decimal `json:"amount"`
	InterestRate    decimal.Decimal `json:"interest_rate"` // Годовая ставка на момент выдачи
	TermMonths      int             `json:"term_months"`
	StartDate       time.Time       `json:"start_date"`
	PaymentSchedule []Payment       `json:"payment_schedule"`
	RemainingAmount decimal.Decimal `json:"remaining_amount"`
}

type Payment struct {
	DueDate       time.Time       `json:"due_date"`
	Amount        decimal.Decimal `json:"amount"`
	PrincipalPart decimal.Decimal `json:"principal_part"`
	InterestPart  decimal.Decimal `json:"interest_part"`
	Paid          bool            `json:"paid"`
}

// --- Структуры для запросов ---

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateAccountRequest struct {
	UserID string `json:"user_id"` // В реальном приложении ID берется из токена/сессии
}

type GenerateCardRequest struct {
	AccountID string `json:"account_id"`
}

type PaymentRequest struct {
	CardNumber string          `json:"card_number"`
	Amount     decimal.Decimal `json:"amount"`
	Merchant   string          `json:"merchant"` // Куда платим
}

type TransferRequest struct {
	FromAccountID string          `json:"from_account_id"`
	ToAccountID   string          `json:"to_account_id"`
	Amount        decimal.Decimal `json:"amount"`
}

type DepositRequest struct {
	ToAccountID string          `json:"to_account_id"`
	Amount      decimal.Decimal `json:"amount"`
}

type ApplyLoanRequest struct {
	UserID     string          `json:"user_id"` // Опять же, в реальности из сессии
	AccountID  string          `json:"account_id"`
	Amount     decimal.Decimal `json:"amount"`
	TermMonths int             `json:"term_months"`
}
