package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
)

type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    Funcionario `json:"data"`
}

type JWTResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    string `json:"data"`
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

// Requests é uma função que faz uma requisição HTTP para a url especificada
// com o método especificado (GET, POST, PUT, DELETE) e retorna o corpo da resposta
// como um ponteiro para um slice de bytes. Ela aceita parâmetros opcionais
// parametros: metodo, url, resp, param (map[string]string), data (interface{}),
// headers (map[string]string)).
func Requests(method string, url string, resp interface{}, args ...interface{}) error {
	var params map[string]string
	var data any
	var headers map[string]string
	var payload *bytes.Buffer
	payload = bytes.NewBuffer(nil)

	// ptr, ok := resp.(*interface{})
	// if !ok {
	// 	return fmt.Errorf("out deve ser um ponteiro para interface")
	// }

	// // Verifica se o ponteiro dentro da interface é válido
	// if *ptr == nil {
	// 	return fmt.Errorf("out é um ponteiro nulo")
	// }

	if len(args) > 0 {
		if len(args) > 0 {
			if args[0] != nil {
				params = args[0].(map[string]string)
			}
		}
		if len(args) > 1 {
			data = args[1]
		}
		if len(args) > 2 {
			if args[2] != nil {
				headers = args[2].(map[string]string)
			}
		}
	}

	if data != nil {
		payloadBuffer, err := json.Marshal(data)
		if err != nil {
			return err
		}
		payload = bytes.NewBuffer(payloadBuffer)
	}

	if params != nil {
		url += buildQuery(params)
	}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return err
	}

	for header := range headers {
		req.Header.Add(header, headers[header])
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return errors.New(string(body))
	}

	err = json.Unmarshal(body, resp)
	if err != nil {
		return err
	}

	return nil
}

func GetPermissao(coduser int, codmodulo int) (*Funcionario, error) {
	var err error
	var jwt string

	jwt, err = getJWT()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Authorization": jwt,
	}

	params := map[string]string{
		"modulo": strconv.Itoa(codmodulo),
	}

	var resp Response
	err = Requests("GET", os.Getenv("AUTH_LINK")+"/auth/"+strconv.Itoa(coduser), &resp, params, nil, headers)
	if err != nil {
		return nil, err
	}

	return &resp.Data, err

}

func GetPermissoes(coduser int) (*Funcionario, error) {
	var err error
	var resp Response

	jwt, err := getJWT()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Authorization": jwt,
	}

	err = Requests("GET", os.Getenv("AUTH_LINK")+"/auth/"+strconv.Itoa(coduser), &resp, nil, nil, headers)
	if err != nil {
		return nil, err
	}

	return &resp.Data, err
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
	var jwtResp JWTResponse
	var err error

	body := map[string]string{
		"username": os.Getenv("USER_AUTH"),
		"password": os.Getenv("PASS_AUTH"),
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	err = Requests("POST", os.Getenv("AUTH_LINK")+"/login", &jwtResp, nil, body, headers)
	if err != nil {
		return "", err
	}

	return "Bearer " + jwtResp.Data, err
}

// func main() {
// 	var err error
// 	var resp *Funcionario

// 	err = godotenv.Load(".env")
// 	if err != nil {
// 		log.Printf("Error ao carregar o arquivo .env: %s", err)
// 	}

// 	resp, err = GetPermissoes(327)
// 	if err != nil {
// 		log.Panic(err)
// 	}
// 	log.Print(*resp.Permissoes)
// 	//TODO: testar um post com application/x url encodded e nil

// }
