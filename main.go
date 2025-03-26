package medfuturalib

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    Funcionario `json:"data"`
}

type Modulo struct {
	CodModulo int    `json:"codModulo"`
	Descricao string `json:"descricao"`
	Consultar bool   `json:"consultar"`
	Gravar    bool   `json:"gravar"`
	Excluir   bool   `json:"excluir"`
}

type Funcionario struct {
	Id         int       `json:"id"`
	Cargo      string    `json:"cargo"`
	Nome       string    `json:"nome"`
	Email      string    `json:"email"`
	Permissoes *[]Modulo `json:"permissoes"`
}

func init() {
	err := godotenv.Load()
	if err != nil {
		panic("Erro ao carregar o arquivo .env")
	}
}

func GetPermissoes(coduser string) (*Funcionario, error) {

	req, err := http.NewRequest("GET", os.Getenv("LINK")+coduser, nil)
	if err != nil {
		return nil, errors.New("erro ao criar requisição: " + err.Error())
	}

	req.Header.Set("Authorization", os.Getenv("JWT"))

	return RequestResponse(req)
}

func GetPermissao(coduser string, codmodulo string) (*Funcionario, error) {

	req, err := http.NewRequest("GET", os.Getenv("LINK")+coduser+"?modulo="+codmodulo, nil)
	if err != nil {
		return nil, errors.New("erro ao criar requisição: " + err.Error())
	}

	req.Header.Set("Authorization", os.Getenv("JWT"))

	return RequestResponse(req)
}

func RequestResponse(req *http.Request) (*Funcionario, error) {

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("erro ao fazer a requisição: " + err.Error())
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("erro ao ler Json: " + err.Error())
	}

	if response.StatusCode != 200 {
		return nil, errors.New("erro ao acessar a API")
	}

	var resp Response

	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, errors.New("Erro ao decodificar JSON: " + err.Error())
	}

	return &resp.Data, nil
}
