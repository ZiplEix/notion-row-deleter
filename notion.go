package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func queryDatabase(client *http.Client, token, databaseID, cursor string) (queryResp, error) {
	body, _ := json.Marshal(queryReq{PageSize: 100, StartCursor: cursor})
	req, _ := http.NewRequest("POST", "https://api.notion.com/v1/databases/"+databaseID+"/query", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionVersion)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return queryResp{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return queryResp{}, fmt.Errorf("query %s: %s", resp.Status, string(b))
	}
	var out queryResp
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func archivePage(client *http.Client, token, pageID string) error {
	payload := map[string]any{"archived": true}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PATCH", "https://api.notion.com/v1/pages/"+pageID, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionVersion)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("archive %s: %s", resp.Status, string(body))
	}
	fmt.Printf("Archived page: %s\n", pageID)
	return nil
}
