package medfutura_lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/MedFutura/medfutura_lib/models"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
)

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
		if args[0] != nil {
			params = args[0].(map[string]string)
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
	//TODO: URL ENCODDED
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

// Função que faz uma requição pra api de permissao e retorna as permissoes
// daquele usuario no modulo especificado
func GetPermissao(coduser int, codmodulo int) (*models.Funcionario, error) {
	var err error
	var jwt string

	jwt, err = GetJWT()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Authorization": jwt,
	}

	params := map[string]string{
		"modulo": strconv.Itoa(codmodulo),
	}

	var resp models.ResponseAuth
	err = Requests("GET", os.Getenv("AUTH_LINK")+"/auth/"+strconv.Itoa(coduser), &resp, params, nil, headers)
	if err != nil {
		return nil, err
	}

	return &resp.Data, err

}

// Função que faz uma requisição pra api de permissao e retorna todas
// as permissoes de um usuario
func GetPermissoes(coduser int) (*models.Funcionario, error) {
	var err error
	var resp models.ResponseAuth
	jwt, err := GetJWT()
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

// Função que cria conector dos bancos de dados padrao da empresa
// (apenas sql server por enquanto)
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

// Função de filtro de array
func Filter[T any](data []T, test func(T) bool) (ret []T) {
	for _, s := range data {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

// Funcao de paginação.
// Recebe um slice T, o numero da pagina e o numero de itens por pagina.
// Retorna o numero de paginas e o slice formatado
func Paginar[T any](data *[]T, page int, nItens int) (qtdpaginas int) {
	if nItens == 0 {
		nItens = 30
	}

	qtdPaginas := 0

	qtdPaginas = len(*data) / nItens
	if len(*data)%nItens != 0 {
		qtdPaginas++
	}

	if page > qtdPaginas {
		page = qtdPaginas
	}

	if page != 0 {
		index := (page - 1) * nItens
		if len(*data) < index+nItens {
			nItens = len(*data) - index
		}
		*data = (*data)[index : index+nItens]
	}

	return qtdPaginas
}

// Criar notificacao
func CriarNotificacao(notificacao models.NotificacaoPost) error {
	var err error
	var resp models.Response

	jwt, err := GetJWT()
	if err != nil {
		return err
	}

	headers := map[string]string{
		"Authorization": jwt,
	}

	err = Requests("POST", os.Getenv("NOTIFICACAO_LINK"), &resp, nil, notificacao, headers)
	if err != nil {
		return err
	}

	return nil
}

// Criar auditoria
func CriarAuditoria(notificacao models.AuditoriaPost) error {
	var err error
	var resp models.Response

	jwt, err := GetJWT()
	if err != nil {
		return err
	}

	headers := map[string]string{
		"Authorization": jwt,
	}

	err = Requests("POST", os.Getenv("NOTIFICACAO_LINK")+"/auditoria", &resp, nil, notificacao, headers)
	if err != nil {
		return err
	}

	return nil
}

// Pega o token de autorização
func GetJWT() (string, error) {
	var jwtResp models.Response
	var data string
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

	data = jwtResp.Data.(string)

	return "Bearer " + data, err
}

// Ordena um slice de interface baseado em uma chave
func OrderBy[T any](data []T, key string, asc bool) []T {
	sort.Slice(data, func(i, j int) bool {
		valI := reflect.ValueOf(data[i])
		valJ := reflect.ValueOf(data[j])

		fieldI := valI.FieldByName(key)
		fieldJ := valJ.FieldByName(key)

		if !fieldI.IsValid() || !fieldJ.IsValid() {
			return false
		}

		switch fieldI.Kind() {
		case reflect.String:
			if asc {
				return fieldI.String() < fieldJ.String()
			}
			return fieldI.String() > fieldJ.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if asc {
				return fieldI.Int() < fieldJ.Int()
			}
			return fieldI.Int() > fieldJ.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if asc {
				return fieldI.Uint() < fieldJ.Uint()
			}
			return fieldI.Uint() > fieldJ.Uint()
		case reflect.Float32, reflect.Float64:
			if asc {
				return fieldI.Float() < fieldJ.Float()
			}
			return fieldI.Float() > fieldJ.Float()
		default:
			return false
		}
	})
	return data
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
