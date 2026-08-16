package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestHandleToolsListIncludesAIReviewTools(t *testing.T) {
	t.Parallel()

	toolByName := toolMapForTest(t)
	for _, name := range []string{
		"get_ai_review",
		"list_ai_reviews",
		"create_ai_review_fix_pr",
		"get_ai_review_fix_job",
	} {
		if _, ok := toolByName[name]; !ok {
			t.Fatalf("Expected tool %s not found", name)
		}
	}
}

func TestAIReviewToolSchemas(t *testing.T) {
	t.Parallel()

	toolByName := toolMapForTest(t)

	getReview := toolByName["get_ai_review"]
	getRequired, ok := getReview.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected get_ai_review required schema")
	}
	if !containsRequiredField(getRequired, "review_uuid") {
		t.Fatal("Expected get_ai_review to require review_uuid")
	}

	listReviews := toolByName["list_ai_reviews"]
	listRequired, ok := listReviews.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected list_ai_reviews required schema")
	}
	if !containsRequiredField(listRequired, "project_id") {
		t.Fatal("Expected list_ai_reviews to require project_id")
	}

	createFix := toolByName["create_ai_review_fix_pr"]
	createRequired, ok := createFix.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected create_ai_review_fix_pr required schema")
	}
	for _, field := range []string{"project_id", "review_uuid"} {
		if !containsRequiredField(createRequired, field) {
			t.Fatalf("Expected create_ai_review_fix_pr to require %s", field)
		}
	}
	createProps, ok := createFix.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected create_ai_review_fix_pr properties")
	}
	if _, ok := createProps["finding_indexes"]; !ok {
		t.Fatal("Expected create_ai_review_fix_pr to expose finding_indexes")
	}

	getJob := toolByName["get_ai_review_fix_job"]
	jobRequired, ok := getJob.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected get_ai_review_fix_job required schema")
	}
	if !containsRequiredField(jobRequired, "job_uuid") {
		t.Fatal("Expected get_ai_review_fix_job to require job_uuid")
	}
}

func TestAIReviewToolsCallControllerPaths(t *testing.T) {
	t.Parallel()

	var gotPaths []string
	var gotMethods []string

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			gotPaths = append(gotPaths, r.URL.Path)
			gotMethods = append(gotMethods, r.Method)
			body := `{"success":true,"data":{"uuid":"job-1","status":"queued"}}`
			if strings.Contains(r.URL.Path, "/ai-reviews/") && r.Method == http.MethodGet && !strings.Contains(r.URL.Path, "fix-jobs") {
				body = `{"success":true,"data":{"uuid":"rev-1","pr_number":1660,"findings":[]}}`
			}
			if strings.HasSuffix(r.URL.Path, "/ai-reviews") {
				body = `{"success":true,"data":[{"uuid":"rev-1","pr_number":1660}]}`
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	})
	client.SetToken("sat_test")

	server := &Server{client: client}
	ctx := context.Background()

	const (
		projectUUID = "11111111-1111-1111-1111-111111111111"
		reviewUUID  = "22222222-2222-2222-2222-222222222222"
		jobUUID     = "33333333-3333-3333-3333-333333333333"
	)

	if _, err := server.getAIReviewTool(ctx, map[string]interface{}{
		"review_uuid": reviewUUID,
	}); err != nil {
		t.Fatalf("getAIReviewTool: %v", err)
	}
	if _, err := server.listAIReviewsTool(ctx, map[string]interface{}{
		"project_id": projectUUID,
		"limit":      10,
	}); err != nil {
		t.Fatalf("listAIReviewsTool: %v", err)
	}
	if _, err := server.createAIReviewFixPRTool(ctx, map[string]interface{}{
		"project_id":  projectUUID,
		"review_uuid": reviewUUID,
	}); err != nil {
		t.Fatalf("createAIReviewFixPRTool: %v", err)
	}
	if _, err := server.getAIReviewFixJobTool(ctx, map[string]interface{}{
		"job_uuid": jobUUID,
	}); err != nil {
		t.Fatalf("getAIReviewFixJobTool: %v", err)
	}

	want := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/ai-reviews/" + reviewUUID},
		{http.MethodGet, "/api/v1/projects/" + projectUUID + "/ai-reviews"},
		{http.MethodPost, "/api/v1/projects/" + projectUUID + "/ai-reviews/" + reviewUUID + "/fix-pr"},
		{http.MethodGet, "/api/v1/ai-reviews/fix-jobs/" + jobUUID},
	}
	if len(gotPaths) != len(want) {
		t.Fatalf("got %d requests, want %d: paths=%v methods=%v", len(gotPaths), len(want), gotPaths, gotMethods)
	}
	for i, w := range want {
		if gotMethods[i] != w.method || gotPaths[i] != w.path {
			t.Errorf("request[%d] = %s %s, want %s %s", i, gotMethods[i], gotPaths[i], w.method, w.path)
		}
	}
}

func TestCreateAIReviewFixPRBody(t *testing.T) {
	t.Parallel()

	var gotBody map[string]interface{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Body != nil {
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
			}
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"success":true,"data":{"uuid":"job-2","status":"queued"}}`)),
				Request:    r,
			}, nil
		}),
	})
	client.SetToken("sat_test")

	server := &Server{client: client}
	if _, err := server.createAIReviewFixPRTool(context.Background(), map[string]interface{}{
		"project_id":      "11111111-1111-1111-1111-111111111111",
		"review_uuid":     "22222222-2222-2222-2222-222222222222",
		"finding_indexes": []interface{}{0, 2},
		"mode":            "branch_pr",
	}); err != nil {
		t.Fatalf("createAIReviewFixPRTool: %v", err)
	}

	if gotBody["mode"] != "branch_pr" {
		t.Fatalf("mode = %v, want branch_pr", gotBody["mode"])
	}
	indexes, ok := gotBody["finding_indexes"].([]interface{})
	if !ok || len(indexes) != 2 {
		t.Fatalf("finding_indexes = %#v, want [0,2]", gotBody["finding_indexes"])
	}
}
