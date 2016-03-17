package http

type EntityGroup struct {
	Name       string            `json:"name"`
	Expression string            `json:"expression"`
	Tags       map[string]string `json:"tags"`
}
