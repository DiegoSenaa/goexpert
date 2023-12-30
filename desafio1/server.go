package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type CotacaoResponse struct {
	USDBRL struct {
		Bid string `json:"bid"`
	} `json:"USDBRL"`
}

const (
	dbFileName       string = "cotacoes.db"
	createTableSQL   string = `CREATE TABLE IF NOT EXISTS cotacoes (id INTEGER PRIMARY KEY AUTOINCREMENT, bid TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);`
	insertCotacaoSQL string = "INSERT INTO cotacoes (bid) VALUES (?);"
)

var db *sql.DB

func main() {
	log.Println("[INFO]", "Iniciando servidor na porta 8080")
	http.HandleFunc("/cotacao", handleCotacao)
	initDB()
	log.Println("[INFO]", "Escutando na porta 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB() {
	var err error

	db, err = sql.Open("sqlite3", dbFileName)
	if err != nil {
		log.Fatal("[ERROR]", "Erro ao abrir banco de dados:", err.Error())
	}

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("[ERROR]", "Erro ao criar tabela no banco de dados:", err.Error())
	}

	log.Println("[INFO]", "Banco de dados inicializado")
}

func handleCotacao(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		log.Println("[ERROR] Timeout na requisição da cotação")
		http.Error(w, "Timeout na requisição da cotação", http.StatusRequestTimeout)
		return
	default:
	}

	cotacao, err := fetchCotacao(ctx)
	if err != nil {
		log.Println("[ERROR]", "Erro ao obter cotação:", err)
		http.Error(w, "Erro ao obter cotação", http.StatusInternalServerError)
		return
	}

	err = saveCotacao(ctx, cotacao)
	if err != nil {
		log.Println("[ERROR]", "Erro ao salvar cotação no banco de dados:", err.Error())
		http.Error(w, "Erro ao salvar cotação no banco de dados", http.StatusInternalServerError)
		return
	}

	response := map[string]string{"Dólar": cotacao}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func fetchCotacao(ctx context.Context) (string, error) {
	client := http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var data CotacaoResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	return data.USDBRL.Bid, nil
}

func saveCotacao(ctx context.Context, cotacao string) error {
	select {
	case <-ctx.Done():
		log.Println("[ERROR] Timeout ao salvar cotação no banco de dados")
		return ctx.Err()
	default:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := db.ExecContext(ctx, insertCotacaoSQL, cotacao)
	return err
}
