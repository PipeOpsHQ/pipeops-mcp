package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestHandleInitialize(t *testing.T) {
	server := &Server{}
	result := server.handleInitialize()

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if resultMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %v", resultMap["protocolVersion"])
	}

	if _, ok := resultMap["capabilities"]; !ok {
		t.Error("Expected capabilities in result")
	}

	if _, ok := resultMap["serverInfo"]; !ok {
		t.Error("Expected serverInfo in result")
	}
}

func TestHandleToolsList(t *testing.T) {
	server := &Server{}
	result := server.handleToolsList()

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	tools, ok := resultMap["tools"].([]Tool)
	if !ok {
		t.Fatal("Expected tools to be a slice of Tool")
	}

	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}

	// Check for expected tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"list_projects",
		"get_project",
		"create_project",
		"update_project",
		"delete_project",
		"deploy_project",
		"restart_project",
		"stop_project",
		"get_project_logs",
		"get_project_env_variables",
		"update_project_env_variables",
		"deploy_project_from_image",
		"create_external_registry",
		"list_external_registries",
		"get_external_registry",
		"delete_external_registry",
		"list_external_registry_images",
		"list_external_registry_tags",
		"search_public_registry_images",
		"list_public_registry_tags",
		"list_servers",
		"get_server",
		"get_cluster_connection",
		"get_cluster_cost_allocation",
		"list_cloud_provider_regions",
		"list_cloud_provider_instance_categories",
		"list_cloud_provider_instance_types",
		"list_cloud_provider_server_templates",
		"list_environments",
		"get_environment",
		"create_environment",
		"update_environment",
		"delete_environment",
		"set_environment_variables",
		"list_teams",
		"create_team",
		"update_team",
		"get_team",
		"invite_team_member",
		"list_team_members",
		"remove_team_member",
		"update_team_member_role",
		"list_workspaces",
		"create_workspace",
		"update_workspace",
		"get_workspace",
		"delete_workspace",
		"set_workspace_billing_email",
		"get_current_user",
		"list_addons",
		"get_addon",
		"deploy_addon",
		"list_addon_deployments",
		"get_addon_deployment",
		"get_addon_deployment_session",
		"view_addon_deployment_configs",
		"add_addon_domain",
		"list_addon_categories",
		"get_my_addon_submissions",
		"get_billing_info",
		"list_billing_plans",
		"subscribe_to_plan",
		"cancel_subscription",
		"add_billing_card",
		"delete_billing_card",
		"create_workspace_checkout",
		"start_trial",
		"get_billing_portal_url",
		"get_balance",
		"list_workspace_cards",
		"get_active_card",
		"list_subscriptions",
		"get_subscription",
		"list_invoices",
		"list_service_account_tokens",
		"get_service_account_token",
		"create_service_account_token",
		"update_service_account_token",
		"revoke_service_account_token",
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool %s not found", expected)
		}
	}
}

func TestHandleToolsListSchemas(t *testing.T) {
	server := &Server{}
	result := server.handleToolsList()

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	tools, ok := resultMap["tools"].([]Tool)
	if !ok {
		t.Fatal("Expected tools to be a slice of Tool")
	}

	toolByName := make(map[string]Tool)
	for _, tool := range tools {
		toolByName[tool.Name] = tool
	}

	listProjects := toolByName["list_projects"]
	if _, ok := listProjects.InputSchema["required"]; ok {
		t.Error("Did not expect list_projects to require arguments")
	}

	properties, ok := listProjects.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_projects properties schema")
	}

	if _, ok := properties["workspace_id"]; !ok {
		t.Error("Expected list_projects to expose workspace_id filter")
	}

	if _, ok := properties["server_id"]; !ok {
		t.Error("Expected list_projects to expose server_id filter")
	}

	listServers := toolByName["list_servers"]
	listServersProperties, ok := listServers.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_servers properties schema")
	}

	if _, ok := listServersProperties["workspace_id"]; !ok {
		t.Error("Expected list_servers to expose workspace_id filter")
	}

	listEnvironments := toolByName["list_environments"]
	if _, ok := listEnvironments.InputSchema["required"]; ok {
		t.Error("Did not expect list_environments to require project_id")
	}

	createProject := toolByName["create_project"]
	required, ok := createProject.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected create_project required schema to be []string")
	}

	requiredFields := make(map[string]bool)
	for _, field := range required {
		requiredFields[field] = true
	}

	for _, field := range []string{"name", "server_id", "environment_id", "repository", "branch"} {
		if !requiredFields[field] {
			t.Errorf("Expected create_project to require %s", field)
		}
	}

	createEnvironment := toolByName["create_environment"]
	environmentRequired, ok := createEnvironment.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected create_environment required schema to be []string")
	}

	environmentRequiredFields := make(map[string]bool)
	for _, field := range environmentRequired {
		environmentRequiredFields[field] = true
	}

	for _, field := range []string{"name", "workspace_id"} {
		if !environmentRequiredFields[field] {
			t.Errorf("Expected create_environment to require %s", field)
		}
	}

	inviteTeamMember := toolByName["invite_team_member"]
	inviteRequired, ok := inviteTeamMember.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected invite_team_member required schema to be []string")
	}

	inviteRequiredFields := make(map[string]bool)
	for _, field := range inviteRequired {
		inviteRequiredFields[field] = true
	}

	for _, field := range []string{"team_id", "email"} {
		if !inviteRequiredFields[field] {
			t.Errorf("Expected invite_team_member to require %s", field)
		}
	}

	setWorkspaceBillingEmail := toolByName["set_workspace_billing_email"]
	workspaceBillingRequired, ok := setWorkspaceBillingEmail.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected set_workspace_billing_email required schema to be []string")
	}

	workspaceBillingRequiredFields := make(map[string]bool)
	for _, field := range workspaceBillingRequired {
		workspaceBillingRequiredFields[field] = true
	}

	for _, field := range []string{"workspace_id", "email"} {
		if !workspaceBillingRequiredFields[field] {
			t.Errorf("Expected set_workspace_billing_email to require %s", field)
		}
	}

	deployAddon := toolByName["deploy_addon"]
	deployAddonRequired, ok := deployAddon.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected deploy_addon required schema to be []string")
	}

	if len(deployAddonRequired) != 1 || deployAddonRequired[0] != "addon_id" {
		t.Fatalf("Expected deploy_addon to require only addon_id, got %v", deployAddonRequired)
	}

	addAddonDomain := toolByName["add_addon_domain"]
	addAddonDomainRequired, ok := addAddonDomain.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected add_addon_domain required schema to be []string")
	}

	addAddonDomainRequiredFields := make(map[string]bool)
	for _, field := range addAddonDomainRequired {
		addAddonDomainRequiredFields[field] = true
	}

	for _, field := range []string{"addon_id", "domain"} {
		if !addAddonDomainRequiredFields[field] {
			t.Errorf("Expected add_addon_domain to require %s", field)
		}
	}

	addBillingCard := toolByName["add_billing_card"]
	addBillingCardRequired, ok := addBillingCard.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected add_billing_card required schema to be []string")
	}

	if len(addBillingCardRequired) != 1 || addBillingCardRequired[0] != "token" {
		t.Fatalf("Expected add_billing_card to require only token, got %v", addBillingCardRequired)
	}

	deleteBillingCard := toolByName["delete_billing_card"]
	deleteBillingCardRequired, ok := deleteBillingCard.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected delete_billing_card required schema to be []string")
	}

	if len(deleteBillingCardRequired) != 1 || deleteBillingCardRequired[0] != "card_id" {
		t.Fatalf("Expected delete_billing_card to require only card_id, got %v", deleteBillingCardRequired)
	}

	startTrial := toolByName["start_trial"]
	startTrialRequired, ok := startTrial.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected start_trial required schema to be []string")
	}

	if len(startTrialRequired) != 1 || startTrialRequired[0] != "plan_id" {
		t.Fatalf("Expected start_trial to require only plan_id, got %v", startTrialRequired)
	}

	createServiceAccountToken := toolByName["create_service_account_token"]
	createServiceAccountTokenRequired, ok := createServiceAccountToken.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected create_service_account_token required schema to be []string")
	}

	if len(createServiceAccountTokenRequired) != 1 || createServiceAccountTokenRequired[0] != "name" {
		t.Fatalf("Expected create_service_account_token to require only name, got %v", createServiceAccountTokenRequired)
	}

	updateServiceAccountToken := toolByName["update_service_account_token"]
	updateServiceAccountTokenRequired, ok := updateServiceAccountToken.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected update_service_account_token required schema to be []string")
	}

	if len(updateServiceAccountTokenRequired) != 1 || updateServiceAccountTokenRequired[0] != "token_id" {
		t.Fatalf("Expected update_service_account_token to require only token_id, got %v", updateServiceAccountTokenRequired)
	}

	revokeServiceAccountToken := toolByName["revoke_service_account_token"]
	revokeServiceAccountTokenRequired, ok := revokeServiceAccountToken.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected revoke_service_account_token required schema to be []string")
	}

	if len(revokeServiceAccountTokenRequired) != 1 || revokeServiceAccountTokenRequired[0] != "token_id" {
		t.Fatalf("Expected revoke_service_account_token to require only token_id, got %v", revokeServiceAccountTokenRequired)
	}

	getClusterConnection := toolByName["get_cluster_connection"]
	getClusterConnectionRequired, ok := getClusterConnection.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected get_cluster_connection required schema to be []string")
	}

	if len(getClusterConnectionRequired) != 1 || getClusterConnectionRequired[0] != "cluster_id" {
		t.Fatalf("Expected get_cluster_connection to require only cluster_id, got %v", getClusterConnectionRequired)
	}

	getClusterConnectionProperties, ok := getClusterConnection.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected get_cluster_connection properties schema")
	}

	if _, ok := getClusterConnectionProperties["server_id"]; !ok {
		t.Error("Expected get_cluster_connection to expose server_id alias")
	}

	getClusterCostAllocation := toolByName["get_cluster_cost_allocation"]
	getClusterCostAllocationRequired, ok := getClusterCostAllocation.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected get_cluster_cost_allocation required schema to be []string")
	}

	if len(getClusterCostAllocationRequired) != 1 || getClusterCostAllocationRequired[0] != "cluster_id" {
		t.Fatalf("Expected get_cluster_cost_allocation to require only cluster_id, got %v", getClusterCostAllocationRequired)
	}

	deployProjectFromImage := toolByName["deploy_project_from_image"]
	deployProjectFromImageRequired, ok := deployProjectFromImage.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected deploy_project_from_image required schema to be []string")
	}

	deployProjectFromImageRequiredFields := make(map[string]bool)
	for _, field := range deployProjectFromImageRequired {
		deployProjectFromImageRequiredFields[field] = true
	}

	for _, field := range []string{"name", "container_image", "port", "vcpu", "memory", "server_id", "environment_id", "workspace_id"} {
		if !deployProjectFromImageRequiredFields[field] {
			t.Errorf("Expected deploy_project_from_image to require %s", field)
		}
	}

	createExternalRegistry := toolByName["create_external_registry"]
	createExternalRegistryRequired, ok := createExternalRegistry.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected create_external_registry required schema to be []string")
	}

	createExternalRegistryRequiredFields := make(map[string]bool)
	for _, field := range createExternalRegistryRequired {
		createExternalRegistryRequiredFields[field] = true
	}

	for _, field := range []string{"workspace_id", "name", "type", "username", "password"} {
		if !createExternalRegistryRequiredFields[field] {
			t.Errorf("Expected create_external_registry to require %s", field)
		}
	}

	listExternalRegistries := toolByName["list_external_registries"]
	listExternalRegistriesRequired, ok := listExternalRegistries.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected list_external_registries required schema to be []string")
	}

	if len(listExternalRegistriesRequired) != 1 || listExternalRegistriesRequired[0] != "workspace_id" {
		t.Fatalf("Expected list_external_registries to require only workspace_id, got %v", listExternalRegistriesRequired)
	}

	searchPublicRegistryImages := toolByName["search_public_registry_images"]
	searchPublicRegistryImagesRequired, ok := searchPublicRegistryImages.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected search_public_registry_images required schema to be []string")
	}

	if len(searchPublicRegistryImagesRequired) != 1 || searchPublicRegistryImagesRequired[0] != "query" {
		t.Fatalf("Expected search_public_registry_images to require only query, got %v", searchPublicRegistryImagesRequired)
	}

	listPublicRegistryTags := toolByName["list_public_registry_tags"]
	listPublicRegistryTagsRequired, ok := listPublicRegistryTags.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected list_public_registry_tags required schema to be []string")
	}

	listPublicRegistryTagsRequiredFields := make(map[string]bool)
	for _, field := range listPublicRegistryTagsRequired {
		listPublicRegistryTagsRequiredFields[field] = true
	}

	for _, field := range []string{"namespace", "repository"} {
		if !listPublicRegistryTagsRequiredFields[field] {
			t.Errorf("Expected list_public_registry_tags to require %s", field)
		}
	}

	listCloudProviderRegions := toolByName["list_cloud_provider_regions"]
	listCloudProviderRegionsRequired, ok := listCloudProviderRegions.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected list_cloud_provider_regions required schema to be []string")
	}

	if len(listCloudProviderRegionsRequired) != 1 || listCloudProviderRegionsRequired[0] != "cloud_provider" {
		t.Fatalf("Expected list_cloud_provider_regions to require only cloud_provider, got %v", listCloudProviderRegionsRequired)
	}

	listCloudProviderInstanceTypes := toolByName["list_cloud_provider_instance_types"]
	listCloudProviderInstanceTypesRequired, ok := listCloudProviderInstanceTypes.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected list_cloud_provider_instance_types required schema to be []string")
	}

	listCloudProviderInstanceTypesRequiredFields := make(map[string]bool)
	for _, field := range listCloudProviderInstanceTypesRequired {
		listCloudProviderInstanceTypesRequiredFields[field] = true
	}

	for _, field := range []string{"cloud_provider", "instance_class", "region"} {
		if !listCloudProviderInstanceTypesRequiredFields[field] {
			t.Errorf("Expected list_cloud_provider_instance_types to require %s", field)
		}
	}
}

func TestToolCallParams(t *testing.T) {
	jsonData := []byte(`{
		"name": "get_project",
		"arguments": {
			"project_id": "test-project-123"
		}
	}`)

	var params ToolCallParams
	err := json.Unmarshal(jsonData, &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if params.Name != "get_project" {
		t.Errorf("Expected name 'get_project', got %s", params.Name)
	}

	projectID, ok := params.Arguments["project_id"].(string)
	if !ok || projectID != "test-project-123" {
		t.Errorf("Expected project_id 'test-project-123', got %v", params.Arguments["project_id"])
	}
}

func TestFormatJSON(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
		"age":  30,
	}

	result := formatJSON(data)
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Check if it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Errorf("Result is not valid JSON: %v", err)
	}
}

func TestHandleMessage(t *testing.T) {
	server := &Server{}
	ctx := context.Background()

	tests := []struct {
		name        string
		method      string
		expectError bool
		checkResult bool
	}{
		{
			name:        "initialize",
			method:      "initialize",
			expectError: false,
			checkResult: true,
		},
		{
			name:        "tools/list",
			method:      "tools/list",
			expectError: false,
			checkResult: true,
		},
		{
			name:        "unknown method",
			method:      "unknown/method",
			expectError: true,
			checkResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{
				JSONRPC: "2.0",
				ID:      1,
				Method:  tt.method,
			}

			response := server.handleMessage(ctx, msg)

			if response.JSONRPC != "2.0" {
				t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
			}

			if tt.expectError {
				if response.Error == nil {
					t.Error("Expected error in response")
				}
			} else {
				if response.Error != nil {
					t.Errorf("Unexpected error: %v", response.Error)
				}
			}

			if tt.checkResult && response.Result == nil {
				t.Error("Expected result in response")
			}
		})
	}
}

func TestHandleToolsCallValidatesRequiredArguments(t *testing.T) {
	server := &Server{}
	ctx := context.Background()

	_, err := server.handleToolsCall(ctx, []byte(`{"name":"get_project","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing required argument error")
	}

	if err.Error() != "project_id is required" {
		t.Fatalf("Expected project_id error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"set_workspace_billing_email","arguments":{"workspace_id":"ws_123"}}`))
	if err == nil {
		t.Fatal("Expected missing email error")
	}

	if err.Error() != "email is required" {
		t.Fatalf("Expected email error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"add_addon_domain","arguments":{"addon_id":"addon_123"}}`))
	if err == nil {
		t.Fatal("Expected missing domain error")
	}

	if err.Error() != "domain is required" {
		t.Fatalf("Expected domain error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"add_billing_card","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing token error")
	}

	if err.Error() != "token is required" {
		t.Fatalf("Expected token error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"start_trial","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing plan_id error")
	}

	if err.Error() != "plan_id is required" {
		t.Fatalf("Expected plan_id error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"create_service_account_token","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing name error")
	}

	if err.Error() != "name is required" {
		t.Fatalf("Expected name error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"update_service_account_token","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing token_id error for update_service_account_token")
	}

	if err.Error() != "token_id is required" {
		t.Fatalf("Expected token_id error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"revoke_service_account_token","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing token_id error for revoke_service_account_token")
	}

	if err.Error() != "token_id is required" {
		t.Fatalf("Expected token_id error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"get_cluster_connection","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing cluster_id error for get_cluster_connection")
	}

	if err.Error() != "cluster_id is required" {
		t.Fatalf("Expected cluster_id error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"get_cluster_cost_allocation","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing cluster_id error for get_cluster_cost_allocation")
	}

	if err.Error() != "cluster_id is required" {
		t.Fatalf("Expected cluster_id error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"deploy_project_from_image","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing name error for deploy_project_from_image")
	}

	if err.Error() != "name is required" {
		t.Fatalf("Expected name error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"create_external_registry","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing workspace_id error for create_external_registry")
	}

	if err.Error() != "workspace_id is required" {
		t.Fatalf("Expected workspace_id error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"list_external_registry_tags","arguments":{"registry_id":"reg_123"}}`))
	if err == nil {
		t.Fatal("Expected missing namespace error for list_external_registry_tags")
	}

	if err.Error() != "namespace is required" {
		t.Fatalf("Expected namespace error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"search_public_registry_images","arguments":{}}`))
	if err == nil {
		t.Fatal("Expected missing query error for search_public_registry_images")
	}

	if err.Error() != "query is required" {
		t.Fatalf("Expected query error, got %v", err)
	}

	_, err = server.handleToolsCall(ctx, []byte(`{"name":"list_cloud_provider_instance_types","arguments":{"cloud_provider":"digital_ocean"}}`))
	if err == nil {
		t.Fatal("Expected missing instance_class error for list_cloud_provider_instance_types")
	}

	if err.Error() != "instance_class is required" {
		t.Fatalf("Expected instance_class error, got %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResponse(req *http.Request, statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func TestGetBillingInfoToolUsesSupportedControllerEndpoints(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++

			switch r.URL.Path {
			case "/billing/balance":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":{"Balance":"0.01","Currency":"USD"},"message":"ok","success":true}`), nil
			case "/billing/subscriptions/current":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":{"UID":"sub_123","PlanTier":"startup","PlanName":"Start-up","Amount":"34.99","BillingType":"trial","Status":"active"},"message":"ok","success":true}`), nil
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getBillingInfoTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("getBillingInfoTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("Expected single content item, got %v", resultMap["content"])
	}

	textContent, ok := content[0].(map[string]interface{})["text"].(string)
	if !ok {
		t.Fatalf("Expected text content, got %v", content[0])
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &payload); err != nil {
		t.Fatalf("failed to decode result JSON: %v", err)
	}

	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data map, got %v", payload["data"])
	}

	balance, ok := data["balance"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected balance map, got %v", data["balance"])
	}
	if balance["Balance"] != "0.01" {
		t.Fatalf("balance Balance = %v, want %v", balance["Balance"], "0.01")
	}

	currentSubscription, ok := data["current_subscription"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected current_subscription map, got %v", data["current_subscription"])
	}
	if currentSubscription["PlanTier"] != "startup" {
		t.Fatalf("PlanTier = %v, want %v", currentSubscription["PlanTier"], "startup")
	}

	if requests["/billing/balance"] != 1 {
		t.Fatalf("billing/balance requests = %d, want 1", requests["/billing/balance"])
	}
	if requests["/billing/subscriptions/current"] != 1 {
		t.Fatalf("billing/subscriptions/current requests = %d, want 1", requests["/billing/subscriptions/current"])
	}
}

func TestGetBillingInfoToolAllowsMissingCurrentSubscription(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/billing/balance":
				return jsonHTTPResponse(r, http.StatusOK, `{"data":{"Balance":"10.00","Currency":"USD"},"message":"ok","success":true}`), nil
			case "/billing/subscriptions/current":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"not found"}`), nil
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getBillingInfoTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("getBillingInfoTool error: %v", err)
	}

	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]interface{})
	textContent := content[0].(map[string]interface{})["text"].(string)

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &payload); err != nil {
		t.Fatalf("failed to decode result JSON: %v", err)
	}

	data := payload["data"].(map[string]interface{})
	if _, ok := data["balance"].(map[string]interface{}); !ok {
		t.Fatalf("Expected balance map, got %v", data["balance"])
	}
	if data["current_subscription"] != nil {
		t.Fatalf("Expected nil current_subscription, got %v", data["current_subscription"])
	}
}
