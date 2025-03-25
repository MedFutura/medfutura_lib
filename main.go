package medfuturalib

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func init() {
    err := godotenv.Load()
    if err != nil {
        panic("Erro ao carregar o arquivo .env")
    }
}

func RequestData(coduser string) (string, error) {

	req, err := http.NewRequest("GET", os.Getenv("LINK") + coduser, nil)
	if err != nil {
		return "", errors.New("erro ao criar requisição: " + err.Error())
	}

	req.Header.Set("Authorization", os.Getenv("JWT"))

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return "", errors.New("erro ao fazer a requisição: " + err.Error())
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", errors.New("erro ao ler Json: " + err.Error())
	}

	if response.StatusCode != 200 {
		return "", errors.New("erro ao acessar a API")
	}

	type Response struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    string `json:"data"`
	}

	var resp Response

	err = json.Unmarshal(data, &resp)
	return resp.Data, err
}
