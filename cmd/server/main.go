// Заготовка сервера. Закрывайте этапы по одному, пока не позеленеет go test ./tests/ -v
//
// Запуск: go run ./cmd/server — порт берётся из PORT, по умолчанию 8080.
// Панель уже раздаётся: откройте http://localhost:8080/ и смотрите, как этапы
// зеленеют по ходу работы.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EchoRequest struct {
	Message string `json:"message"`
}

type Message struct {
	ID        int       `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	messages      []Message
	idCounter     int
	messagesMutex sync.Mutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/echo", EchoHandler)
	mux.HandleFunc("/messages", MessageHandler)
	mux.HandleFunc("DELETE /messages/{id}", DeleteHandler)

	// Панель из frontend/. Каталог берётся относительно рабочего, поэтому
	// запускайте из корня модуля: go run ./cmd/server
	mux.Handle("/", http.FileServer(http.Dir("frontend")))

	// TODO Этап 1: GET /health           -> 200, тело "ok"
	// TODO Этап 2: POST /echo            -> тело запроса без изменений
	// TODO Этап 3: POST /echo            -> на application/json разобрать {"message": "..."} и вернуть JSON
	// TODO Этап 4: POST /messages        -> сохранить в памяти, 201
	// TODO Этап 5: GET /messages         -> все сообщения, новые сверху
	// TODO Этап 6: DELETE /messages/{id} -> 204, либо 404 если такого нет

	log.Printf("сервер слушает http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func EchoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {

		var req EchoRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(req)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err := io.Copy(w, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	messagesMutex.Lock()
	defer messagesMutex.Unlock()

	foundIndex := -1
	for i, msg := range messages {
		if msg.ID == id {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	messages = append(messages[:foundIndex], messages[foundIndex+1:]...)

	w.WriteHeader(http.StatusNoContent)
}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var input struct {
			Message *string `json:"message"`
		}

		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if input.Message == nil || *input.Message == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		messagesMutex.Lock()

		idCounter++
		newMsg := Message{
			ID:        idCounter,
			Message:   *input.Message,
			CreatedAt: time.Now().UTC(),
		}

		messages = append([]Message{newMsg}, messages...)
		messagesMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newMsg)

	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		messagesMutex.Lock()
		result := messages
		if result == nil {
			result = []Message{}
		}

		json.NewEncoder(w).Encode(result)
		messagesMutex.Unlock()

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
