package main

import (
	"log"
	"net/http"
	"os" // Для логгирования
	"time"

	"github.com/gorilla/mux"
)

func main() {
	// Настройка логгирования
	log.SetOutput(os.Stdout)                             // Писать логи в консоль
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Формат логов

	log.Println("Starting Simple Bank API...")

	// Инициализация хранилища
	InitStorage()
	log.Println("In-memory storage initialized.")

	// Инициализация роутера
	r := mux.NewRouter()

	// --- Маршруты ---
	// Пользователи
	r.HandleFunc("/register", RegisterUserHandler).Methods("POST")
	r.HandleFunc("/login", LoginUserHandler).Methods("POST")

	// Счета
	r.HandleFunc("/accounts", CreateAccountHandler).Methods("POST")
	r.HandleFunc("/users/{userId}/accounts", GetUserAccountsHandler).Methods("GET")

	// Карты
	r.HandleFunc("/cards", GenerateCardHandler).Methods("POST")
	r.HandleFunc("/accounts/{accountId}/cards", GetAccountCardsHandler).Methods("GET")
	r.HandleFunc("/payments/card", PayWithCardHandler).Methods("POST") // Оплата картой

	// Переводы и пополнения
	r.HandleFunc("/transfers", TransferHandler).Methods("POST")
	r.HandleFunc("/deposits", DepositHandler).Methods("POST")

	// Кредиты
	r.HandleFunc("/loans", ApplyLoanHandler).Methods("POST")
	r.HandleFunc("/loans/{loanId}/schedule", GetLoanScheduleHandler).Methods("GET") // Получить график

	// Аналитика
	r.HandleFunc("/analytics/transactions/{accountId}", GetTransactionsHandler).Methods("GET")
	r.HandleFunc("/analytics/summary/{userId}", GetFinancialSummaryHandler).Methods("GET")

	// --- Запуск сервера ---
	port := "8080" // Порт можно вынести в конфигурацию
	log.Printf("Server starting on port %s", port)

	// Оборачиваем роутер в Middleware для логгирования запросов
	loggedRouter := loggingMiddleware(r)

	err := http.ListenAndServe(":"+port, loggedRouter)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// Простая Middleware для логгирования каждого запроса
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("--> %s %s %s", r.Method, r.RequestURI, r.Proto)
		// Передаем управление следующему обработчику
		next.ServeHTTP(w, r)
		// Логгируем после завершения обработки
		log.Printf("<-- %s %s (%v)", r.Method, r.RequestURI, time.Since(start))
	})
}
