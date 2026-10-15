package cloud

import (
	"context"
	"fmt"
	"kandev-plugin-bitbucket/internal/domain"
	"net/http"
	"strings"
)

type sourcePipelinePayload struct {
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	State       struct {
		Name   string `json:"name"`
		Result struct {
			Name string `json:"name"`
		} `json:"result"`
	} `json:"state"`
	Target struct {
		RefName string `json:"ref_name"`
		Type    string `json:"type"`
		RefType string `json:"ref_type"`
		Source  string `json:"source"`
		Commit  struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"target"`
}

func sourcePipelineView(p sourcePipelinePayload) map[string]any {
	return map[string]any{"uuid": p.UUID, "build_number": p.BuildNumber, "state": p.State.Name, "result": p.State.Result.Name, "source_branch": sourcePipelineBranch(p), "source_head": p.Target.Commit.Hash}
}

func (c *Client) sourcePipeline(ctx context.Context, repository domain.Repository, source string, input SourceInput) (map[string]any, error) {
	var branch struct {
		Name   string `json:"name"`
		Target struct {
			Hash string `json:"hash"`
		} `json:"target"`
	}
	if err := c.sourceJSON(ctx, http.MethodGet, c.sourceEndpoint(repository, "refs", "branches", source), nil, &branch); err != nil {
		return nil, err
	}
	if branch.Name != source || !sourceHash.MatchString(branch.Target.Hash) {
		return nil, fmt.Errorf("bitbucket_source_head_unavailable")
	}
	head := branch.Target.Hash
	if input.Operation == "statuses" {
		return c.sourceStatuses(ctx, repository, head)
	}
	if input.Operation == "pipelines" {
		endpoint := c.sourceEndpoint(repository, "pipelines")
		query := endpoint.Query()
		query.Set("pagelen", fmt.Sprint(sourcePageLimit))
		query.Set("target.commit.hash", head)
		query.Set("sort", "-created_on")
		endpoint.RawQuery = query.Encode()
		var page struct {
			Values []sourcePipelinePayload `json:"values"`
			Next   string                  `json:"next"`
		}
		if err := c.sourceJSON(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
			return nil, err
		}
		if len(page.Values) > sourcePageLimit {
			return nil, fmt.Errorf("bitbucket_source_response_limit")
		}
		values := []map[string]any{}
		for _, pipeline := range page.Values {
			if sourcePipelineBranch(pipeline) == source && pipeline.Target.Commit.Hash == head {
				values = append(values, sourcePipelineView(pipeline))
			}
		}
		return map[string]any{"pipelines": values, "source_head": head, "truncated": page.Next != ""}, nil
	}
	if !validSourceSegment(input.PipelineID) {
		return nil, fmt.Errorf("bitbucket_pipeline_id_required")
	}
	var pipeline sourcePipelinePayload
	if err := c.sourceJSON(ctx, http.MethodGet, c.sourceEndpoint(repository, "pipelines", input.PipelineID), nil, &pipeline); err != nil {
		return nil, err
	}
	if pipeline.UUID != input.PipelineID || sourcePipelineBranch(pipeline) != source || pipeline.Target.Commit.Hash != head {
		return nil, fmt.Errorf("bitbucket_pipeline_source_mismatch")
	}
	if input.Operation == "pipeline_get" {
		return sourcePipelineView(pipeline), nil
	}
	if input.Operation == "pipeline_steps" {
		return c.sourceSteps(ctx, repository, input.PipelineID)
	}
	if !validSourceSegment(input.StepID) {
		return nil, fmt.Errorf("bitbucket_step_id_required")
	}
	var step sourceStepPayload
	if err := c.sourceJSON(ctx, http.MethodGet, c.sourceEndpoint(repository, "pipelines", input.PipelineID, "steps", input.StepID), nil, &step); err != nil {
		return nil, err
	}
	if step.UUID != input.StepID {
		return nil, fmt.Errorf("bitbucket_step_source_mismatch")
	}
	body, truncated, err := c.sourceRequest(ctx, http.MethodGet, c.sourceEndpoint(repository, "pipelines", input.PipelineID, "steps", input.StepID, "log"), nil, true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"log": string(body), "truncated": truncated, "source_head": head, "pipeline_id": pipeline.UUID, "step_id": step.UUID}, nil
}

type sourceStepPayload struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	State struct {
		Name string `json:"name"`
	} `json:"state"`
}

func (c *Client) sourceSteps(ctx context.Context, repository domain.Repository, pipeline string) (map[string]any, error) {
	endpoint := c.sourceEndpoint(repository, "pipelines", pipeline, "steps")
	query := endpoint.Query()
	query.Set("pagelen", fmt.Sprint(sourcePageLimit))
	endpoint.RawQuery = query.Encode()
	var page struct {
		Values []sourceStepPayload `json:"values"`
		Next   string              `json:"next"`
	}
	if err := c.sourceJSON(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return nil, err
	}
	if len(page.Values) > sourcePageLimit {
		return nil, fmt.Errorf("bitbucket_source_response_limit")
	}
	values := []map[string]any{}
	for _, step := range page.Values {
		values = append(values, map[string]any{"uuid": step.UUID, "name": step.Name, "state": step.State.Name})
	}
	return map[string]any{"steps": values, "truncated": page.Next != ""}, nil
}
func (c *Client) sourceStatuses(ctx context.Context, repository domain.Repository, head string) (map[string]any, error) {
	endpoint := c.sourceEndpoint(repository, "commit", head, "statuses")
	query := endpoint.Query()
	query.Set("pagelen", fmt.Sprint(sourcePageLimit))
	endpoint.RawQuery = query.Encode()
	var page struct {
		Values []struct {
			Key         string `json:"key"`
			Name        string `json:"name"`
			State       string `json:"state"`
			Description string `json:"description"`
		} `json:"values"`
		Next string `json:"next"`
	}
	if err := c.sourceJSON(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return nil, err
	}
	if len(page.Values) > sourcePageLimit {
		return nil, fmt.Errorf("bitbucket_source_response_limit")
	}
	values := []map[string]any{}
	for _, status := range page.Values {
		values = append(values, map[string]any{"key": status.Key, "name": status.Name, "state": strings.ToUpper(status.State), "description": status.Description})
	}
	return map[string]any{"statuses": values, "source_head": head, "truncated": page.Next != ""}, nil
}

func sourcePipelineBranch(p sourcePipelinePayload) string {
	switch p.Target.Type {
	case "pipeline_ref_target":
		if p.Target.RefType == "branch" {
			return p.Target.RefName
		}
	case "pipeline_pullrequest_target":
		return p.Target.Source
	}
	return ""
}
