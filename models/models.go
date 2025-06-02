package models

type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
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

type NotificacaoPost struct {
	Mensagem          string  `db:"mensagem" json:"mensagem"`
	Path              *string `json:"path" db:"path"`
	Tipo              int     `db:"tipo" json:"tipo"`
	UserGerador       *int    `db:"userGerador" json:"userGerador"`
	UserDestinatario  []int   `db:"userDestinatario" json:"userDestinatario"`
	GrupoDestinatario *int    `db:"grupoDestinatario" json:"grupoDestinatario"`
}
