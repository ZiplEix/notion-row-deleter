package main

type queryReq struct {
	PageSize    int    `json:"page_size"`
	StartCursor string `json:"start_cursor,omitempty"`
}

type queryResp struct {
	Results []struct {
		ID string `json:"id"`
	} `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}
