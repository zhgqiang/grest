package grest

// Filter is Query Conditions
type Filter struct {
	Fields    []string      `json:"fields"`
	Order     string        `json:"order"`
	Where     []interface{} `json:"where"`
	WithCount bool          `json:"withCount"`
	Joins     []string      `json:"joins"`
	Groups    []string      `json:"groups"`
	Preloads  []string      `json:"preloads"`
	Offset    string        `json:"offset"`
	Limit     string        `json:"limit"`
}
