package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	serverURL  = "http://localhost:8080/cotacao"
	outputFile = "cotacao.txt"
)

func main() {
	log.Println("[INFO]", "Realizando requisição para obter cotação")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	res, err := callCotacao(ctx)
	if err != nil {
		log.Println("[ERROR]", "Erro ao obter cotação:", err.Error())
		return
	}

	err = saveToFile(res)
	if err != nil {
		log.Println("[ERROR]", "Erro ao salvar cotação no arquivo:", err.Error())
		return
	}

	log.Println("[INFO]", "Cotação salva com sucesso em", outputFile)
}

func callCotacao(ctx context.Context) (map[string]string, error) {
	client := http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]string
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func saveToFile(data map[string]string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for key, value := range data {
		_, err := fmt.Fprintf(file, "%s: %s\n", key, value)
		if err != nil {
			return err
		}
	}

	return nil
}
