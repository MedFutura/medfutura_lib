package main

import (
	"C"

	"encoding/json"
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

//export RequestData
func RequestData(coduser *C.char) *C.char {
	goCoduser := C.GoString(coduser)

	req, err := http.NewRequest("GET", os.Getenv("LINK") + goCoduser, nil)
	if err != nil {
		return C.CString("Erro ao criar requisição: " + err.Error())
	}

	req.Header.Set("Authorization", os.Getenv("JWT"))

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return C.CString("Erro ao fazer requisição: " + err.Error())
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return C.CString("Erro ao ler resposta: " + err.Error())
	}

	if response.StatusCode != 200 {
		return C.CString("Erro ao acessar a API: " + string(data))
	}

	type Response struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    string `json:"data"`
	}

	var resp Response
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return C.CString("Erro ao decodificar JSON: " + err.Error())
	}

	return C.CString(resp.Data)
}

func main() {}
