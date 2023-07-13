package sonobuoy

type results struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Items  []item `json:"items"`
}

type item struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Items  []item `json:"items"`
}
