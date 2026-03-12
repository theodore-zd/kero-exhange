package handlers

import "encoding/json"

type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func (m PaginationMeta) MarshalJSON() ([]byte, error) {
	type Alias PaginationMeta
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(m),
	})
}
