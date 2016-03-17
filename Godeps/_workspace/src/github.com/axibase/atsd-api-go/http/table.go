package http

type Table struct {
	Columns []*Column       `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

type Column struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Metric  string `json:"metric"`
	Type    string `json:"type"`
	Numeric bool   `json:"numeric"`
}
