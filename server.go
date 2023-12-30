package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type economiaResponseDto struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

const (
	createTableSQL   = `CREATE TABLE IF NOT EXISTS cotacoes (id INTEGER PRIMARY KEY AUTOINCREMENT, bid REAL, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);`
	insertCotacaoSQL = "INSERT INTO cotacoes (bid) VALUES (?);"
	dbFileName       = "cotacao.db"
)

var db *sql.DB

func init() {

	log.Println("[INFO]", "Verificando base de dados")
	iniciaBaseDeDados()
	log.Println("[INFO]", "Iniciando conexão base de dados")
	abreConexao()
	log.Println("[INFO]", "Iniciando tabela da base de dados")
	createTable()
}

func createTable() *sql.DB {
	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err.Error())
	}
	return db
}

func abreConexao() {

	var err error

	db, err = sql.Open("sqlite3", dbFileName)
	if err != nil {
		log.Fatal("[ERROR]", "Error opening database:", err.Error())

	}
}

func iniciaBaseDeDados() {
	if _, err := os.Stat(dbFileName); os.IsNotExist(err) {
		log.Println("[INFO]", "Arquivo nao existe. Criando...")
		file, err := os.Create(dbFileName)
		if err != nil {
			log.Fatal("[ERROR]", "Erro ao criar arquivo da base:", err)
		}
		file.Close()
		log.Println("[INFO]", "Database arquivo criado.")
	}
}

func main() {
	log.Println("[INFO]", "Rodando")
	defer db.Close()
	handleRequest()
}

func handleRequest() {
	http.HandleFunc("/cotacao", handleCotacao)
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func handleCotacao(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		log.Println("[Error]: Timeout de resposta excedido")
		http.Error(w, "Timeout", http.StatusRequestTimeout)
		return
	default:
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	res, err := fetchCotacao(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = saveCotacao(ctx, res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"bid": res,
	}

	json.NewEncoder(w).Encode(response)
}

func fetchCotacao(ctx context.Context) (float64, error) {

	select {
	case <-ctx.Done():
		log.Println("Error: Timeout excedido  no fetching  da cotacao")
		return 0, ctx.Err()
	default:
	}

	client := http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return 0, err
	}
	res, err := client.Do(req)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	var data economiaResponseDto
	err = json.Unmarshal(body, &data)
	if err != nil {
		return 0, err
	}
	var f float64
	f, err = strconv.ParseFloat(data.USDBRL.Bid, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}

func saveCotacao(ctx context.Context, cotacao float64) error {
	select {
	case <-ctx.Done():
		log.Println("[Error] Timeout ao salvar cotacao no db")
		return ctx.Err()
	default:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := db.ExecContext(ctx, insertCotacaoSQL, cotacao)
	return err
}
