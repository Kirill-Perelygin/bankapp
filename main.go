package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println("Starting Simple Bank API...")

	InitStorage()
	log.Println("In-memory storage initialized.")

	r := mux.NewRouter()

	r.HandleFunc("/register", RegisterUserHandler).Methods("POST")
	r.HandleFunc("/login", LoginUserHandler).Methods("POST")

	r.HandleFunc("/accounts", CreateAccountHandler).Methods("POST")
	r.HandleFunc("/users/{userId}/accounts", GetUserAccountsHandler).Methods("GET")

	r.HandleFunc("/cards", GenerateCardHandler).Methods("POST")
	r.HandleFunc("/accounts/{accountId}/cards", GetAccountCardsHandler).Methods("GET")
	r.HandleFunc("/payments/card", PayWithCardHandler).Methods("POST")

	r.HandleFunc("/transfers", TransferHandler).Methods("POST")
	r.HandleFunc("/deposits", DepositHandler).Methods("POST")

	r.HandleFunc("/loans", ApplyLoanHandler).Methods("POST")
	r.HandleFunc("/loans/{loanId}/schedule", GetLoanScheduleHandler).Methods("GET")

	r.HandleFunc("/analytics/transactions/{accountId}", GetTransactionsHandler).Methods("GET")
	r.HandleFunc("/analytics/summary/{userId}", GetFinancialSummaryHandler).Methods("GET")

	port := "8080"
	log.Printf("Server starting on port %s", port)

	loggedRouter := loggingMiddleware(r)

	err := http.ListenAndServe(":"+port, loggedRouter)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("--> %s %s %s", r.Method, r.RequestURI, r.Proto)
		next.ServeHTTP(w, r)
		log.Printf("<-- %s %s (%v)", r.Method, r.RequestURI, time.Since(start))
	})
}
