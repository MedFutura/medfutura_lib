package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Response struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type JWTResponse struct{
	Username	string `json:"username"`
	Password	string `json:"password"`
	Token 		string `json:"token"`
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

func GetPermissoes(coduser int) (*Funcionario, error) {

	req, err := http.NewRequest("GET", os.Getenv("LINK")+"/auth/"+strconv.Itoa(coduser), nil)
	if err != nil {
		return nil, errors.New("erro ao criar requisição: " + err.Error())
	}

	jwt, err := GetJWT()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", jwt)

	data, err := RequestResponse(req)
	if err != nil{
		return nil, err
	}

	return ToFuncionario(data)
}

func GetPermissao(coduser int, codmodulo int) (*Funcionario, error) {

	req, err := http.NewRequest("GET", os.Getenv("LINK")+"/auth/"+strconv.Itoa(coduser)+"?modulo="+strconv.Itoa(codmodulo), nil)
	if err != nil {
		return nil, errors.New("erro ao criar requisição: " + err.Error())
	}

	jwt, err := GetJWT()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", jwt)

	data, err := RequestResponse(req)
	if err != nil {
		return nil, err
	}

	return ToFuncionario(data)

}

func GetJWT() (string, error) {
	body := map[string]string{
		"username": os.Getenv("USER_SIAC"),
		"password": os.Getenv("SENHA_SIAC"),
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", errors.New("Erro ao codificar corpo da requisicao: " + err.Error())
	}

	req, err := http.NewRequest("POST", os.Getenv("LINK")+"/login", bytes.NewBufferString(string(jsonData)))
	if err != nil {
		return "", errors.New("erro ao criar requisição: " + err.Error())
	}

	req.Header.Set("Content-Type", "application/json")

	data, err := RequestResponse(req)
	if err != nil {
		return "", err
	}

	resp, err := ToJWTResponse(data)
	if err != nil {
		return "", err
	}

	return "Bearer " + resp.Token, nil
}

func RequestResponse(req *http.Request) (any, error) {

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
		return nil, errors.New("erro ao acessar a API: " + string(data))
	}

	var resp Response

	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, errors.New("Erro ao decodificar JSON: " + err.Error())
	}

	return resp.Data, nil
}

func ToFuncionario(data any) (*Funcionario, error) {

	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil, errors.New("erro ao codificar json: " + err.Error())
	}

	var funcionario Funcionario

	err = json.Unmarshal(dataJson, &funcionario)
	if err != nil {
		return nil, errors.New("erro ao fazer a conversao para funcionario: " + err.Error())
	}

	return &funcionario, nil
}

func ToJWTResponse(data any) (*JWTResponse, error) {

	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil, errors.New("erro ao codificar json: " + err.Error())
	}

	var resp JWTResponse

	err = json.Unmarshal(dataJson, &resp)
	if err != nil {
		return nil, errors.New("erro ao fazer a conversao: " + err.Error())
	}

	return &resp, nil
}

func main() {

	fmt.Println(GetPermissao(327, 46))

}
