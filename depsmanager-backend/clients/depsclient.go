package clients

import (
	"bytes"
	"context"
	"depsmanager"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type DepsClient struct {
	address string
	client  *http.Client
}

func NewDepsClient(address string) *DepsClient {
	return &DepsClient{address: address, client: &http.Client{}}
}

func (c *DepsClient) GetProjectVersions(ctx context.Context, project string) (*depsmanager.DepsGetVersionResp, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v3/systems/NPM/packages/%s", c.address, url.PathEscape(project)), nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext(): %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.client.Do(): %v", err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var data depsmanager.DepsGetVersionResp
		if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("json.NewDecoder(): %v", err)
		}

		return &data, nil
	case http.StatusNotFound:
		return nil, depsmanager.ErrProjectNotFound
	}

	return nil, fmt.Errorf("bad response: %d", resp.StatusCode)

}

func (c *DepsClient) GetProjectDependencies(ctx context.Context, system, project, version string) (*depsmanager.DepsProjectDependenciesResp, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v3/systems/%s/packages/%s/versions/%s:dependencies", c.address, system, url.PathEscape(project), version), nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext(): %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.client.Do(): %v", err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var data depsmanager.DepsProjectDependenciesResp
		if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("json.NewDecoder(): %v", err)
		}

		return &data, nil
	case http.StatusNotFound:
		return nil, depsmanager.ErrProjectNotFound
	}

	return nil, fmt.Errorf("bad response: %d", resp.StatusCode)

}

func (c *DepsClient) GetVersionsBatch(ctx context.Context, projects []depsmanager.ProjectDependencies) (*depsmanager.DepsGetVersionsBatchResp, error) {
	var v depsmanager.DepsGetVersionsBatchReq
	for _, p := range projects {
		v.Requests = append(v.Requests, depsmanager.DepsGetVersionKey{VersionKey: p})
	}

	jsonValue, err := json.Marshal(v)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v3alpha/versionbatch", c.address), bytes.NewReader(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext(): %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.client.Do(): %v", err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var data depsmanager.DepsGetVersionsBatchResp

		if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("json.NewDecoder(): %v", err)
		}

		return &data, nil
	case http.StatusNotFound:
		return nil, depsmanager.ErrProjectNotFound
	}

	return nil, fmt.Errorf("bad response: %d", resp.StatusCode)

}

func (c *DepsClient) GetProjectsBatch(ctx context.Context, projects []string) (*depsmanager.DepsGetProjectBatchResp, error) {
	var v depsmanager.DepsGetVersionsBatchReq
	for _, id := range projects {
		v.Requests = append(v.Requests, depsmanager.DepsProjectBatch{ProjectKey: struct {
			ID string `json:"id"`
		}{ID: id}})
	}

	jsonValue, err := json.Marshal(v)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v3alpha/projectbatch", c.address), bytes.NewReader(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext(): %v", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.client.Do(): %v", err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var data depsmanager.DepsGetProjectBatchResp

		if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("json.NewDecoder(): %v", err)
		}

		return &data, nil
	case http.StatusNotFound:
		return nil, depsmanager.ErrProjectNotFound
	}

	return nil, fmt.Errorf("bad response: %d", resp.StatusCode)

}
