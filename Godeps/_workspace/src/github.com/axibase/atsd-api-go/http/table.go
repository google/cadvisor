package http

type Table struct {
	Metadata interface{}     `json:"metadata"`
	Data     [][]interface{} `json:"data"`
}
