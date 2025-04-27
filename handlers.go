package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
)

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func respondError(w http.ResponseWriter, code int, message string) {
	log.Printf("HTTP Error %d: %s", code, message)
	respondJSON(w, code, map[string]string{"error": message})
}

func RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Username, email, and password are required")
		return
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user := User{
		ID:           GenerateID(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
	}

	if err := AddUser(user); err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}

	go func() {
		subject := "Welcome to Simple Bank!"
		body := fmt.Sprintf("Hello %s,\n\nThank you for registering at Simple Bank.", user.Username)
		err := SendEmailNotification(user.Email, subject, body)
		if err != nil {
			log.Printf("Failed to send registration email to %s: %v", user.Email, err)
		}
	}()

	log.Printf("User registered: %s (ID: %s)", user.Username, user.ID)
	user.PasswordHash = ""
	respondJSON(w, http.StatusCreated, user)
}

func LoginUserHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	user, ok := GetUserByUsername(req.Username)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	if !CheckPasswordHash(req.Password, user.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	log.Printf("User logged in: %s", user.Username)
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Login successful",
		"user_id": user.ID,
	})
}

func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if req.UserID == "" {
		respondError(w, http.StatusBadRequest, "UserID is required")
		return
	}

	account := Account{
		ID:        GenerateID(),
		UserID:    req.UserID,
		Number:    GenerateAccountNumber(),
		Balance:   decimal.Zero,
		CreatedAt: time.Now(),
	}

	if err := AddAccount(account); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create account: %v", err))
		return
	}

	log.Printf("Account created: %s for user %s", account.Number, account.UserID)
	respondJSON(w, http.StatusCreated, account)
}

func GetUserAccountsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	accounts := GetUserAccounts(userID)
	log.Printf("Fetched %d accounts for user %s", len(accounts), userID)
	respondJSON(w, http.StatusOK, accounts)
}

func GenerateCardHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if _, ok := GetAccount(req.AccountID); !ok {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Account %s not found", req.AccountID))
		return
	}

	month, year := GenerateExpiryDate()
	card := Card{
		ID:          GenerateID(),
		AccountID:   req.AccountID,
		Number:      GenerateCardNumber(),
		ExpiryMonth: month,
		ExpiryYear:  year,
		CVV:         GenerateCVV(),
		CreatedAt:   time.Now(),
	}

	if err := AddCard(card); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate card: %v", err))
		return
	}

	log.Printf("Card generated for account %s", card.AccountID)
	card.CVV = "***"
	respondJSON(w, http.StatusCreated, card)
}

func GetAccountCardsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["accountId"]

	if _, ok := GetAccount(accountID); !ok {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Account %s not found", accountID))
		return
	}

	cards := GetAccountCards(accountID)
	for i := range cards {
		cards[i].CVV = "***"
	}
	log.Printf("Fetched %d cards for account %s", len(cards), accountID)
	respondJSON(w, http.StatusOK, cards)
}

func PayWithCardHandler(w http.ResponseWriter, r *http.Request) {
	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		respondError(w, http.StatusBadRequest, "Payment amount must be positive")
		return
	}

	card, ok := GetCardByNumber(req.CardNumber)
	if !ok {
		respondError(w, http.StatusNotFound, "Card not found")
		return
	}

	now := time.Now()
	expiry := time.Date(card.ExpiryYear, time.Month(card.ExpiryMonth)+1, 0, 23, 59, 59, 0, time.UTC) // Последний день месяца
	if now.After(expiry) {
		respondError(w, http.StatusBadRequest, "Card expired")
		return
	}

	account, ok := GetAccount(card.AccountID)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Associated account not found")
		return
	}

	if account.Balance.LessThan(req.Amount) {
		respondError(w, http.StatusPaymentRequired, "Insufficient funds")
		return
	}

	err := UpdateAccountBalance(account.ID, req.Amount.Neg())
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process payment: %v", err))
		return
	}

	tx := Transaction{
		ID:              GenerateID(),
		FromAccountID:   account.ID,
		ToAccountID:     "",
		Amount:          req.Amount,
		Timestamp:       time.Now(),
		TransactionType: "payment",
		Description:     fmt.Sprintf("Payment to %s", req.Merchant),
	}
	AddTransaction(tx)

	log.Printf("Payment of %s processed from account %s (card %s) to %s", req.Amount.String(), account.ID, card.Number[:4]+"...", req.Merchant)
	respondJSON(w, http.StatusOK, map[string]string{"message": "Payment successful"})
}

func TransferHandler(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if req.FromAccountID == req.ToAccountID {
		respondError(w, http.StatusBadRequest, "Cannot transfer to the same account")
		return
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		respondError(w, http.StatusBadRequest, "Transfer amount must be positive")
		return
	}

	storage.mu.Lock()
	defer storage.mu.Unlock()

	fromAccount, okFrom := storage.accounts[req.FromAccountID]
	toAccount, okTo := storage.accounts[req.ToAccountID]

	if !okFrom {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Source account %s not found", req.FromAccountID))
		return
	}
	if !okTo {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Destination account %s not found", req.ToAccountID))
		return
	}

	if fromAccount.Balance.LessThan(req.Amount) {
		respondError(w, http.StatusPaymentRequired, "Insufficient funds in source account")
		return
	}

	fromAccount.Balance = fromAccount.Balance.Sub(req.Amount)
	toAccount.Balance = toAccount.Balance.Add(req.Amount)

	storage.accounts[req.FromAccountID] = fromAccount
	storage.accounts[req.ToAccountID] = toAccount

	tx := Transaction{
		ID:              GenerateID(),
		FromAccountID:   req.FromAccountID,
		ToAccountID:     req.ToAccountID,
		Amount:          req.Amount,
		Timestamp:       time.Now(),
		TransactionType: "transfer",
		Description:     fmt.Sprintf("Transfer from %s to %s", fromAccount.Number, toAccount.Number),
	}
	storage.transactions = append(storage.transactions, tx)

	log.Printf("Transfer of %s from %s to %s successful", req.Amount.String(), req.FromAccountID, req.ToAccountID)
	respondJSON(w, http.StatusOK, map[string]string{"message": "Transfer successful"})
}

func DepositHandler(w http.ResponseWriter, r *http.Request) {
	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		respondError(w, http.StatusBadRequest, "Deposit amount must be positive")
		return
	}

	err := UpdateAccountBalance(req.ToAccountID, req.Amount)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
		} else {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process deposit: %v", err))
		}
		return
	}

	account, _ := GetAccount(req.ToAccountID)
	tx := Transaction{
		ID:              GenerateID(),
		FromAccountID:   "",
		ToAccountID:     req.ToAccountID,
		Amount:          req.Amount,
		Timestamp:       time.Now(),
		TransactionType: "deposit",
		Description:     fmt.Sprintf("Deposit to account %s", account.Number),
	}
	AddTransaction(tx)

	log.Printf("Deposit of %s to account %s successful", req.Amount.String(), req.ToAccountID)
	respondJSON(w, http.StatusOK, map[string]string{"message": "Deposit successful"})
}

func ApplyLoanHandler(w http.ResponseWriter, r *http.Request) {
	var req ApplyLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if req.Amount.LessThanOrEqual(decimal.Zero) || req.TermMonths <= 0 {
		respondError(w, http.StatusBadRequest, "Loan amount and term must be positive")
		return
	}

	storage.mu.RLock()
	_, userExists := storage.users[req.UserID]
	_, accountExists := storage.accounts[req.AccountID]
	storage.mu.RUnlock()

	if !userExists {
		respondError(w, http.StatusNotFound, fmt.Sprintf("User %s not found", req.UserID))
		return
	}
	if !accountExists {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Account %s not found", req.AccountID))
		return
	}

	baseRate, err := GetCBRKeyRate()
	if err != nil {
		log.Printf("Warning: Failed to get key rate, using default 10%%: %v", err)
		baseRate = decimal.NewFromInt(10)
	}

	interestRate := baseRate.Add(decimal.NewFromInt(5))

	monthlyPayment := CalculateMonthlyPayment(req.Amount, interestRate, req.TermMonths)
	startDate := time.Now()
	schedule := GeneratePaymentSchedule(req.Amount, interestRate, req.TermMonths, startDate, monthlyPayment)

	loan := Loan{
		ID:              GenerateID(),
		UserID:          req.UserID,
		AccountID:       req.AccountID,
		Amount:          req.Amount,
		InterestRate:    interestRate,
		TermMonths:      req.TermMonths,
		StartDate:       startDate,
		PaymentSchedule: schedule,
		RemainingAmount: req.Amount,
	}

	if err := AddLoan(loan); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save loan: %v", err))
		return
	}

	err = UpdateAccountBalance(req.AccountID, req.Amount)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to disburse loan funds: %v", err))
		return
	}

	tx := Transaction{
		ID:              GenerateID(),
		FromAccountID:   "", //
		ToAccountID:     req.AccountID,
		Amount:          req.Amount,
		Timestamp:       time.Now(),
		TransactionType: "loan_disbursement",
		Description:     fmt.Sprintf("Loan disbursement (ID: %s)", loan.ID),
	}
	AddTransaction(tx)

	log.Printf("Loan %s approved for user %s, amount %s, rate %s%%, term %d months. Funds disbursed to account %s.",
		loan.ID, req.UserID, req.Amount.String(), interestRate.String(), req.TermMonths, req.AccountID)

	respondJSON(w, http.StatusCreated, loan)
}

func GetLoanScheduleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	loanID := vars["loanId"]

	loan, ok := GetLoan(loanID)
	if !ok {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Loan %s not found", loanID))
		return
	}

	log.Printf("Fetched payment schedule for loan %s", loanID)
	respondJSON(w, http.StatusOK, loan.PaymentSchedule)
}

func GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["accountId"]

	if _, ok := GetAccount(accountID); !ok {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Account %s not found", accountID))
		return
	}

	transactions := GetAccountTransactions(accountID)

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Timestamp.After(transactions[j].Timestamp)
	})

	log.Printf("Fetched %d transactions for account %s", len(transactions), accountID)
	respondJSON(w, http.StatusOK, transactions)
}

func GetFinancialSummaryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	accounts := GetUserAccounts(userID)
	loans := GetUserLoans(userID)

	totalBalance := decimal.Zero
	for _, acc := range accounts {
		totalBalance = totalBalance.Add(acc.Balance)
	}

	totalLoanDebt := decimal.Zero
	activeLoans := 0
	for _, loan := range loans {
		totalLoanDebt = totalLoanDebt.Add(loan.RemainingAmount)
		if loan.RemainingAmount.GreaterThan(decimal.Zero) {
			activeLoans++
		}
	}

	summary := map[string]interface{}{
		"user_id":               userID,
		"total_account_balance": totalBalance,
		"number_of_accounts":    len(accounts),
		"total_loan_debt":       totalLoanDebt,
		"active_loans":          activeLoans,
	}

	log.Printf("Generated financial summary for user %s", userID)
	respondJSON(w, http.StatusOK, summary)
}
