package medfutura_lib

import (
	"bytes"
	"encoding/base64"
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
	"sync"
	"time"

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
func connString(servidor int, banco string) string {
	return fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s",
		os.Getenv(fmt.Sprintf("ADDR_SERVER_%d", servidor)),
		os.Getenv(fmt.Sprintf("SERVER_USER_%d", servidor)),
		os.Getenv(fmt.Sprintf("SERVER_PSSW_%d", servidor)),
		os.Getenv(fmt.Sprintf("PORT_SERVER_%d", servidor)),
		os.Getenv(fmt.Sprintf("DB_%s_%d", banco, servidor)))
}

// Conector abre uma conexão nova a cada chamada (quem chama fecha com Close).
// Uma conexão só: uma query fora de uma transação aberta nela espera para sempre. Prefira Pool.
func Conector(servidor int, banco string) (*sqlx.DB, error) {
	db, err := sqlx.Open("mssql", connString(servidor, banco))
	if err != nil {
		log.Printf("Erro na conexao com db: %s", err)
		return db, err
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxIdleTime(1)
	db.SetConnMaxLifetime(1 * time.Minute)

	return db, nil
}

var (
	pools   = map[string]*sqlx.DB{}
	poolsMu sync.Mutex
)

// Pool devolve o pool de conexões compartilhado do banco, criado na primeira chamada.
// Não feche: ele vive enquanto o processo viver. Toda transação precisa de Commit ou Rollback,
// senão a conexão dela não volta para o pool.
func Pool(servidor int, banco string) (*sqlx.DB, error) {
	chave := fmt.Sprintf("%d/%s", servidor, banco)
	poolsMu.Lock()
	defer poolsMu.Unlock()
	if db, ok := pools[chave]; ok {
		return db, nil
	}
	db, err := sqlx.Open("mssql", connString(servidor, banco))
	if err != nil {
		log.Printf("Erro na conexao com db: %s", err)
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(30 * time.Minute)
	pools[chave] = db
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

var jwtCache struct {
	sync.Mutex
	token  string
	expira time.Time
}

// Pega o token de autorização. Cada login na api de autorização é um login no Auth0, então o token
// fica em memória até 5 min antes de expirar. O lock segura as requisições concorrentes enquanto
// uma renova, para não dispararem vários logins de uma vez.
func GetJWT() (string, error) {
	jwtCache.Lock()
	defer jwtCache.Unlock()

	if jwtCache.token != "" && time.Now().Before(jwtCache.expira) {
		return jwtCache.token, nil
	}

	token, err := loginJWT()
	if err != nil {
		return "", err
	}

	jwtCache.token = "Bearer " + token
	jwtCache.expira = expiracaoJWT(token).Add(-5 * time.Minute)
	return jwtCache.token, nil
}

// Lê o exp do payload do JWT, sem validar a assinatura (o token veio da nossa api de autorização).
// Se não conseguir ler, usa 15 min.
func expiracaoJWT(token string) time.Time {
	// com o desconto de 5 min do GetJWT, vira 10 min de cache
	padrao := time.Now().Add(15 * time.Minute)
	partes := strings.Split(token, ".")
	if len(partes) != 3 {
		return padrao
	}
	payload, err := base64.RawURLEncoding.DecodeString(partes[1])
	if err != nil {
		return padrao
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp == 0 {
		return padrao
	}
	return time.Unix(claims.Exp, 0)
}

func loginJWT() (string, error) {
	var jwtResp models.Response
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

	token, ok := jwtResp.Data.(string)
	if !ok || token == "" {
		return "", errors.New("api de autorização não retornou o token")
	}

	return token, nil
}

// Função auxiliar para encontrar um campo ignorando o case
func caseInsensitiveFieldByName(v reflect.Value, name string) reflect.Value {
	// 1. Converte o nome procurado para minúsculas.
	lowerName := strings.ToLower(name)

	// 2. Utiliza FieldByNameFunc, que itera sobre todos os campos.
	field := v.FieldByNameFunc(func(fieldName string) bool {
		// 3. Compara o nome do campo atual (fieldName)
		// com o nome procurado (lowerName), ambos em minúsculas.
		return strings.ToLower(fieldName) == lowerName
	})

	return field
}

// Ordena um slice de interface baseado em uma chave
func OrderBy[T any](data []T, key string, asc bool) []T {
	sort.Slice(data, func(i, j int) bool {
		valI := reflect.ValueOf(data[i])
		valJ := reflect.ValueOf(data[j])

		fieldI := caseInsensitiveFieldByName(valI, key)
		fieldJ := caseInsensitiveFieldByName(valJ, key)

		// 1. DESREFERENCIAR O PONTEIRO (se for um)
		// Se o Kind for um ponteiro, ele deve chamar Elem()
		// O Elem() retorna o valor para o qual o ponteiro aponta.
		if fieldI.Kind() == reflect.Ptr {
			fieldI = fieldI.Elem()
		}
		if fieldJ.Kind() == reflect.Ptr {
			fieldJ = fieldJ.Elem()
		}

		// Se o campo for um ponteiro NULO, Elem() retornará um valor inválido.
		// Se o campo não for um ponteiro, Elem() panicaria, por isso a verificação acima.

		if !fieldI.IsValid() || !fieldJ.IsValid() {
			// Se um ponteiro for nulo, ele será considerado 'inválido' aqui.
			// Você pode querer uma lógica mais específica para nil, mas 'false' (não-ordenado) é seguro.
			return false
		}

		// ... o restante do código de switch/case é o mesmo
		switch fieldI.Kind() {
		case reflect.String:
			if asc {
				return fieldI.String() < fieldJ.String()
			}
			return fieldI.String() > fieldJ.String()
		// ... (outros cases para Int, Uint, Float)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if asc {
				return fieldI.Int() < fieldJ.Int()
			}
			return fieldI.Int() > fieldJ.Int()
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
