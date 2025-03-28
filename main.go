package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)



type Response struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type JWTResponse struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
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

type Client struct{
	Url 		string 
	HttpClient 	*http.Client
	Headers 	map[string]string
}

func init() {
	err := godotenv.Load()
	if err != nil {
		panic("Erro ao carregar o arquivo .env")
	}
}

func NewHTTPClient(url string, headers map[string]string) *Client{
	return &Client{
		Url: url,
		HttpClient: &http.Client{},
		Headers: headers,
	}
}

func (c *Client) Get(endpoint string, queryParams map[string]string) ([]byte, error){
	

	var fullURL string = c.Url + endpoint

	fullURL += buildQuery(queryParams)

	req, err := http.NewRequest("GET", fullURL , nil)
	if err != nil {
		return nil, errors.New("erro ao criar a requisicao: " + err.Error())
	}

	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}

	return requestResponse(req)
}

func (c *Client) Post(endpoint string, queryParams map[string]string, body map[string]string) ([]byte, error){
	
	var fullURL string = c.Url + endpoint

	fullURL += buildQuery(queryParams)

	reader, err := verifyFormat(c.Headers["Content-Type"], body)
	if err != nil{
		return nil, err
	}

	req, err := http.NewRequest("POST", fullURL , reader)
	if err != nil {
		return nil, errors.New("erro ao criar a requisicao: " + err.Error())
	}

	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}

	return requestResponse(req)
}

func GetPermissoes(coduser int) (*Funcionario, error) {

	jwt, err := getJWT()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Authorization": jwt,
	}

	client := NewHTTPClient(os.Getenv("LINK"), headers)

	data, err := client.Get("/auth/" + strconv.Itoa(coduser), nil)
	if err != nil {
		return nil, err
	}

	var resp Response

	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, errors.New("Erro ao decodificar JSON: " + err.Error())
	}

	return toFuncionario(resp.Data)
}

func GetPermissao(coduser int, codmodulo int) (*Funcionario, error) {

	jwt, err := getJWT()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Authorization": jwt,
	}

	params := map[string]string{
		"modulo": strconv.Itoa(codmodulo),
	}

	client := NewHTTPClient(os.Getenv("LINK"), headers)

	data, err := client.Get("/auth/" + strconv.Itoa(coduser), params)
	if err != nil {
		return nil, err
	} 

	var resp Response

	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, errors.New("Erro ao decodificar JSON: " + err.Error())
	}

	return toFuncionario(resp.Data)

}

func Conector(servidor int, banco string) (*sqlx.DB, error) {

	var serverKey string = fmt.Sprintf("ADDR_SERVER_%d", servidor)
	var portKey string = fmt.Sprintf("PORT_SERVER_%d", servidor)
	var userKey string = fmt.Sprintf("SERVER_USER_%d", servidor)
	var passwrdKey string = fmt.Sprintf("SERVER_PSSW_%d", servidor)
	var dbKey string = fmt.Sprintf("DB_%s_%d", banco, servidor)

	var err error
	var server string = os.Getenv(serverKey)
	var port string = os.Getenv(portKey)
	var user string = os.Getenv(userKey)
	var password string = os.Getenv(passwrdKey)
	var database string = os.Getenv(dbKey)
	var db *sqlx.DB

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s", server, user, password, port, database)
	db, err = sqlx.Open("mssql", connString)
	if err != nil {
		log.Printf("Erro na conexao com db: %s", err)
		return db, err
	}

	return db, nil
}

func Filter[T any](data []T, test func(T) bool) (ret []T) {
	for _, s := range data {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

//funcoes privadas

func verifyFormat(contentType string, body map[string]string) (io.Reader, error){

	if body == nil{
		return nil, nil
	}

	switch contentType{
		case "application/json":
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, errors.New("erro ao decodificar jsoon: " + err.Error())
			}
			return bytes.NewBufferString(string(jsonData)), nil
		
		case "application/x-www-form-urlencoded":
			formData := url.Values{}
			for key, value := range body {
				formData.Set(key, value)
			}
			return strings.NewReader(formData.Encode()), nil

		default:
			return nil, errors.New("erro: content type nao suportado")
	}
}

func buildQuery(queryParams map[string]string) string {
	if queryParams == nil {
		return ""
	}

	var params []string
	for key, value := range queryParams {
		params = append(params, key+"="+value)
	}

	return "?" + strings.Join(params, "&")
}

func getJWT() (string, error) {
	body := map[string]string{
		"username": os.Getenv("USER_SIAC"),
		"password": os.Getenv("SENHA_SIAC"),
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	client := NewHTTPClient(os.Getenv("LINK"), headers)

	data, err := client.Post("/login", nil, body)
	if err != nil {
		return "", err
	}

	var resp Response

	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", errors.New("Erro ao decodificar JSON: " + err.Error())
	}

	jwt, err := toJWTResponse(resp.Data)
	if err != nil {
		return "", err
	}

	return "Bearer " + jwt.Token, nil
}

func requestResponse(req *http.Request) ([]byte, error) {

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

	return data, nil
}

func toFuncionario(data any) (*Funcionario, error) {

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

func toJWTResponse(data any) (*JWTResponse, error) {

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

	fmt.Println(GetPermissoes(327))
	//TODO: testar um post com application/x url encodded e nil

}
