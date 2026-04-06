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

	getProject := toolByName["get_project"]
	getProjectProperties, ok := getProject.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected get_project properties schema")
	}

	if _, ok := getProjectProperties["workspace_id"]; !ok {
		t.Error("Expected get_project to expose workspace_id override")
	}

	getProjectLogs := toolByName["get_project_logs"]
	getProjectLogsProperties, ok := getProjectLogs.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected get_project_logs properties schema")
	}

	if _, ok := getProjectLogsProperties["workspace_id"]; !ok {
		t.Error("Expected get_project_logs to expose workspace_id override")
	}

	listServers := toolByName["list_servers"]
	listServersProperties, ok := listServers.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_servers properties schema")
	}

	if _, ok := listServersProperties["workspace_id"]; !ok {
		t.Error("Expected list_servers to expose workspace_id filter")
	}

	deployAddOn := toolByName["deploy_addon"]
	deployAddOnProperties, ok := deployAddOn.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected deploy_addon properties schema")
	}

	if _, ok := deployAddOnProperties["workspace_id"]; !ok {
		t.Error("Expected deploy_addon to expose workspace_id override")
	}

	listAddOnDeployments := toolByName["list_addon_deployments"]
	listAddOnDeploymentsProperties, ok := listAddOnDeployments.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_addon_deployments properties schema")
	}

	if _, ok := listAddOnDeploymentsProperties["workspace_id"]; !ok {
		t.Error("Expected list_addon_deployments to expose workspace_id filter")
	}

	listEnvironments := toolByName["list_environments"]
	if _, ok := listEnvironments.InputSchema["required"]; ok {
		t.Error("Did not expect list_environments to require project_id")
	}

	listEnvironmentsProperties, ok := listEnvironments.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_environments properties schema")
	}

	if _, ok := listEnvironmentsProperties["workspace_id"]; !ok {
		t.Error("Expected list_environments to expose workspace_id filter")
	}

	getBillingInfo := toolByName["get_billing_info"]
	if _, ok := getBillingInfo.InputSchema["required"]; ok {
		t.Error("Did not expect get_billing_info to require arguments")
	}

	getBillingInfoProperties, ok := getBillingInfo.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected get_billing_info properties schema")
	}

	if _, ok := getBillingInfoProperties["workspace_id"]; !ok {
		t.Error("Expected get_billing_info to expose workspace_id override")
	}

	listSubscriptions := toolByName["list_subscriptions"]
	if _, ok := listSubscriptions.InputSchema["required"]; ok {
		t.Error("Did not expect list_subscriptions to require arguments")
	}

	listSubscriptionsProperties, ok := listSubscriptions.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_subscriptions properties schema")
	}

	if _, ok := listSubscriptionsProperties["workspace_id"]; !ok {
		t.Error("Expected list_subscriptions to expose workspace_id override")
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

	getClusterCostAllocationProperties, ok := getClusterCostAllocation.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected get_cluster_cost_allocation properties schema")
	}

	if _, ok := getClusterCostAllocationProperties["workspace_id"]; !ok {
		t.Error("Expected get_cluster_cost_allocation to expose workspace_id override")
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

func TestListAddOnDeploymentsToolUsesExplicitWorkspace(t *testing.T) {
	t.Parallel()

	workspaceLookups := 0
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/workspace":
				workspaceLookups++
				t.Fatalf("unexpected workspace lookup")
				return nil, nil
			case "/addons/deployments/overview":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				if got := r.URL.Query().Get("workspace"); got != "ws_explicit" {
					t.Fatalf("workspace = %q, want %q", got, "ws_explicit")
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":[{"UID":"dep_1","Name":"Redis"}]}`), nil
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listAddOnDeploymentsTool(context.Background(), map[string]interface{}{"workspace_id": "ws_explicit"})
	if err != nil {
		t.Fatalf("listAddOnDeploymentsTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
	if workspaceLookups != 0 {
		t.Fatalf("workspace lookups = %d, want 0", workspaceLookups)
	}
}

func TestListAddOnDeploymentsToolFallsBackToFirstWorkspace(t *testing.T) {
	t.Parallel()

	workspaceLookups := 0
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/workspace":
				workspaceLookups++
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"id":"1","uuid":"ws_first"},{"id":"2","uuid":"ws_second"}]}`), nil
			case "/addons/deployments/overview":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				if got := r.URL.Query().Get("workspace"); got != "ws_first" {
					t.Fatalf("workspace = %q, want %q", got, "ws_first")
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":[{"UID":"dep_1","Name":"Redis"}]}`), nil
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listAddOnDeploymentsTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("listAddOnDeploymentsTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
	if workspaceLookups != 1 {
		t.Fatalf("workspace lookups = %d, want 1", workspaceLookups)
	}
}

func TestDeployAddOnToolFallsBackToFirstWorkspace(t *testing.T) {
	t.Parallel()

	workspaceLookups := 0
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/workspace":
				workspaceLookups++
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"uuid":"ws_default"}]}`), nil
			case "/addons/deploy":
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
				}
				body, readErr := io.ReadAll(r.Body)
				if readErr != nil {
					t.Fatalf("ReadAll body error: %v", readErr)
				}
				var payload map[string]interface{}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("unmarshal body error: %v", err)
				}
				if got := payload["id"]; got != "addon_123" {
					t.Fatalf("id = %#v, want %q", got, "addon_123")
				}
				if got := payload["Workspace"]; got != "ws_default" {
					t.Fatalf("Workspace = %#v, want %q", got, "ws_default")
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"status":"success","data":{"deployment":{"UID":"dep_1"}}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.deployAddOnTool(context.Background(), map[string]interface{}{"addon_id": "addon_123"})
	if err != nil {
		t.Fatalf("deployAddOnTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
	if workspaceLookups != 1 {
		t.Fatalf("workspace lookups = %d, want 1", workspaceLookups)
	}
}

func TestListProjectsToolAggregatesAcrossWorkspacesUsingWorkspaceFetchIDFallback(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"w1"},{"ID":2,"UUID":"w2"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"workspace with uuid not found"}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/1":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"p1","Name":"proj-1","ID":1487,"ClusterUUID":"srv1"}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w2":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"p1","Name":"proj-1","ID":1487,"ClusterUUID":"srv1"},{"UUID":"p2","Name":"proj-2","ID":1488,"ClusterUUID":"srv2"}]}}}`), nil
			case r.URL.Path == "/project/fetch" || r.URL.Path == "/project/fetch-names" || r.URL.Path == "/projects":
				t.Fatalf("unexpected legacy project route: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listProjectsTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("listProjectsTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	projects, ok := data["projects"].([]interface{})
	if !ok {
		t.Fatalf("Expected projects list, got %v", data["projects"])
	}
	if len(projects) != 2 {
		t.Fatalf("projects len = %d, want %d", len(projects), 2)
	}
	if requests["/workspace/fetch/w1"] != 1 {
		t.Fatalf("workspace/fetch/w1 calls = %d, want 1", requests["/workspace/fetch/w1"])
	}
	if requests["/workspace/fetch/1"] != 1 {
		t.Fatalf("workspace/fetch/1 calls = %d, want 1", requests["/workspace/fetch/1"])
	}
	if requests["/workspace/fetch/w2"] != 1 {
		t.Fatalf("workspace/fetch/w2 calls = %d, want 1", requests["/workspace/fetch/w2"])
	}
}

func TestListProjectsToolSkipsZeroWorkspaceIDFallback(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":0,"UUID":"w1"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"workspace with uuid not found"}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/0":
				t.Fatalf("unexpected zero workspace id fallback request: %s %s", r.Method, r.URL.Path)
				return nil, nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listProjectsTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("listProjectsTool error: %v", err)
	}
	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]interface{})
	textContent := content[0].(map[string]interface{})["text"].(string)

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &payload); err != nil {
		t.Fatalf("failed to decode result JSON: %v", err)
	}
	data := payload["data"].(map[string]interface{})
	projects := data["projects"].([]interface{})
	if len(projects) != 0 {
		t.Fatalf("projects len = %d, want %d", len(projects), 0)
	}
	if requests["/workspace/fetch/w1"] != 1 {
		t.Fatalf("workspace/fetch/w1 calls = %d, want 1", requests["/workspace/fetch/w1"])
	}
	if requests["/workspace/fetch/0"] != 0 {
		t.Fatalf("workspace/fetch/0 calls = %d, want 0", requests["/workspace/fetch/0"])
	}
}

func TestListProjectsToolExplicitWorkspaceFallsBackToWorkspaceID(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":{"workspaces":[{"id":"1","uuid":"w1"}]}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"workspace with uuid not found"}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/1":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"projects":[{"UUID":"p1","Name":"proj-1","ID":1487}]}}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listProjectsTool(context.Background(), map[string]interface{}{"workspace_id": "w1"})
	if err != nil {
		t.Fatalf("listProjectsTool error: %v", err)
	}
	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]interface{})
	textContent := content[0].(map[string]interface{})["text"].(string)

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &payload); err != nil {
		t.Fatalf("failed to decode result JSON: %v", err)
	}
	data := payload["data"].(map[string]interface{})
	projects := data["projects"].([]interface{})
	if len(projects) != 1 {
		t.Fatalf("projects len = %d, want %d", len(projects), 1)
	}
	if requests["/workspace/fetch/w1"] != 1 {
		t.Fatalf("workspace/fetch/w1 calls = %d, want 1", requests["/workspace/fetch/w1"])
	}
	if requests["/workspace/fetch/1"] != 1 {
		t.Fatalf("workspace/fetch/1 calls = %d, want 1", requests["/workspace/fetch/1"])
	}
}

func TestListServersToolAggregatesAcrossWorkspacesUsingWorkspaceFetchFallback(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"w1"},{"ID":2,"UUID":"w2"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"workspace with uuid not found"}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/1":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"UUID":"w1","Clusters":[{"Cluster":{"uuid":"srv1","name":"alpha","cloudProvider":"aws","region":"us-east-1"},"IsActive":true}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w2":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"UUID":"w2","Clusters":[{"uuid":"srv2","name":"beta","cloudProvider":"gcp","region":"europe-west1","status":"provisioning"}]}}}`), nil
			case strings.HasPrefix(r.URL.Path, "/cluster"):
				t.Fatalf("unexpected legacy cluster route: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listServersTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("listServersTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	servers, ok := data["servers"].([]interface{})
	if !ok {
		t.Fatalf("Expected servers list, got %v", data["servers"])
	}
	if len(servers) != 2 {
		t.Fatalf("servers len = %d, want %d", len(servers), 2)
	}

	serverByUUID := make(map[string]map[string]interface{}, len(servers))
	for _, item := range servers {
		serverMap, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected server map, got %T", item)
		}
		serverByUUID[serverMap["uuid"].(string)] = serverMap
	}
	if got := serverByUUID["srv1"]["workspace_id"]; got != "w1" {
		t.Fatalf("srv1 workspace_id = %v, want %q", got, "w1")
	}
	if got := serverByUUID["srv1"]["status"]; got != "active" {
		t.Fatalf("srv1 status = %v, want %q", got, "active")
	}
	if got := serverByUUID["srv2"]["workspace_id"]; got != "w2" {
		t.Fatalf("srv2 workspace_id = %v, want %q", got, "w2")
	}
	if got := serverByUUID["srv2"]["status"]; got != "provisioning" {
		t.Fatalf("srv2 status = %v, want %q", got, "provisioning")
	}
	if requests["/workspace"] != 1 {
		t.Fatalf("workspace calls = %d, want 1", requests["/workspace"])
	}
	if requests["/workspace/fetch/w1"] != 1 {
		t.Fatalf("workspace/fetch/w1 calls = %d, want 1", requests["/workspace/fetch/w1"])
	}
	if requests["/workspace/fetch/1"] != 1 {
		t.Fatalf("workspace/fetch/1 calls = %d, want 1", requests["/workspace/fetch/1"])
	}
	if requests["/workspace/fetch/w2"] != 1 {
		t.Fatalf("workspace/fetch/w2 calls = %d, want 1", requests["/workspace/fetch/w2"])
	}
}

func TestListServersToolExplicitWorkspaceUsesWorkspaceFetchFallback(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":{"workspaces":[{"id":"1","uuid":"w1"}]}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"workspace with uuid not found"}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/1":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"UUID":"w1","Clusters":[{"uuid":"srv1","name":"alpha","cloudProvider":"aws","region":"us-east-1","status":"running"}]}}}`), nil
			case strings.HasPrefix(r.URL.Path, "/cluster"):
				t.Fatalf("unexpected legacy cluster route: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listServersTool(context.Background(), map[string]interface{}{"workspace_id": "w1"})
	if err != nil {
		t.Fatalf("listServersTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	servers, ok := data["servers"].([]interface{})
	if !ok {
		t.Fatalf("Expected servers list, got %v", data["servers"])
	}
	if len(servers) != 1 {
		t.Fatalf("servers len = %d, want %d", len(servers), 1)
	}
	serverMap, ok := servers[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected server map, got %T", servers[0])
	}
	if got := serverMap["workspace_id"]; got != "w1" {
		t.Fatalf("workspace_id = %v, want %q", got, "w1")
	}
	if requests["/workspace/fetch/w1"] != 1 {
		t.Fatalf("workspace/fetch/w1 calls = %d, want 1", requests["/workspace/fetch/w1"])
	}
	if requests["/workspace/fetch/1"] != 1 {
		t.Fatalf("workspace/fetch/1 calls = %d, want 1", requests["/workspace/fetch/1"])
	}
}

func TestGetProjectToolFallsBackToWorkspaceLookupForProjectName(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	workspaceOne := "5877a4ae-a891-49de-909d-0221f5eefc95"
	workspaceTwo := "6f36dd81-50e9-4ea3-8094-8e0212684a11"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"`+workspaceOne+`"},{"ID":2,"UUID":"`+workspaceTwo+`"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceOne:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"p1","Name":"other-project","ID":1487}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceTwo:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"p2","Name":"faulty-art","NameSlug":"faulty-art","ID":1488}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/project/fetch/faulty-art":
				t.Fatalf("unexpected direct name fetch: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			case r.Method == http.MethodGet && r.URL.Path == "/project/fetch/p2":
				if got := r.URL.Query().Get("workspace_uuid"); got != workspaceTwo {
					t.Fatalf("workspace_uuid = %q, want %q", got, workspaceTwo)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"status":"success","message":"ok","data":{"project":{"UUID":"p2","Name":"faulty-art","Description":"resolved by workspace lookup"}}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getProjectTool(context.Background(), map[string]interface{}{"project_id": "faulty-art"})
	if err != nil {
		t.Fatalf("getProjectTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	project, ok := data["project"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected project map, got %v", data["project"])
	}
	if got := project["UUID"]; got != "p2" {
		t.Fatalf("UUID = %v, want %q", got, "p2")
	}
	if got := project["Description"]; got != "resolved by workspace lookup" {
		t.Fatalf("Description = %v, want %q", got, "resolved by workspace lookup")
	}
	if requests["/workspace"] != 1 {
		t.Fatalf("workspace requests = %d, want 1", requests["/workspace"])
	}
	if requests["/workspace/fetch/"+workspaceOne] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceOne, requests["/workspace/fetch/"+workspaceOne])
	}
	if requests["/workspace/fetch/"+workspaceTwo] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceTwo, requests["/workspace/fetch/"+workspaceTwo])
	}
	if requests["/project/fetch/faulty-art"] != 0 {
		t.Fatalf("project/fetch/faulty-art calls = %d, want 0", requests["/project/fetch/faulty-art"])
	}
	if requests["/project/fetch/p2"] != 1 {
		t.Fatalf("project/fetch/p2 calls = %d, want 1", requests["/project/fetch/p2"])
	}
}

func TestGetProjectToolRetriesWithResolvedWorkspaceForDirectIdentifier(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	workspaceOne := "5877a4ae-a891-49de-909d-0221f5eefc95"
	workspaceTwo := "6f36dd81-50e9-4ea3-8094-8e0212684a11"
	projectUUID := "0a427673-a112-47a4-ac2e-90b175fdabff"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"`+workspaceOne+`"},{"ID":2,"UUID":"`+workspaceTwo+`"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/project/fetch/"+projectUUID:
				switch got := r.URL.Query().Get("workspace_uuid"); got {
				case workspaceOne:
					return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"project not found"}`), nil
				case workspaceTwo:
					return jsonHTTPResponse(r, http.StatusOK, `{"status":"success","message":"ok","data":{"project":{"UUID":"`+projectUUID+`","Name":"faulty-art"}}}`), nil
				default:
					t.Fatalf("workspace_uuid = %q, want one of %q or %q", got, workspaceOne, workspaceTwo)
					return nil, nil
				}
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceOne:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"p1","Name":"other-project","ID":1487}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceTwo:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"`+projectUUID+`","Name":"faulty-art","ID":1488}]}}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getProjectTool(context.Background(), map[string]interface{}{"project_id": projectUUID})
	if err != nil {
		t.Fatalf("getProjectTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
	if requests["/project/fetch/"+projectUUID] != 2 {
		t.Fatalf("project/fetch/%s calls = %d, want 2", projectUUID, requests["/project/fetch/"+projectUUID])
	}
	if requests["/workspace"] != 2 {
		t.Fatalf("workspace requests = %d, want 2", requests["/workspace"])
	}
	if requests["/workspace/fetch/"+workspaceOne] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceOne, requests["/workspace/fetch/"+workspaceOne])
	}
	if requests["/workspace/fetch/"+workspaceTwo] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceTwo, requests["/workspace/fetch/"+workspaceTwo])
	}
}

func TestGetProjectLogsToolFallsBackToWorkspaceLookupForProjectName(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	workspaceOne := "5877a4ae-a891-49de-909d-0221f5eefc95"
	workspaceTwo := "6f36dd81-50e9-4ea3-8094-8e0212684a11"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"`+workspaceOne+`"},{"ID":2,"UUID":"`+workspaceTwo+`"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceOne:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"p1","Name":"other-project","ID":1487}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceTwo:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"p2","Name":"utopian-office","NameSlug":"utopian-office","ID":1488}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/project/logs/utopian-office":
				t.Fatalf("unexpected direct logs request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			case r.Method == http.MethodGet && r.URL.Path == "/project/logs/p2":
				if got := r.URL.Query().Get("workspace_uuid"); got != workspaceTwo {
					t.Fatalf("workspace_uuid = %q, want %q", got, workspaceTwo)
				}
				if got := r.URL.Query().Get("limit"); got != "100" {
					t.Fatalf("limit = %q, want %q", got, "100")
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"message":"hello from resolved logs"}]}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getProjectLogsTool(context.Background(), map[string]interface{}{"project_id": "utopian-office", "limit": 100})
	if err != nil {
		t.Fatalf("getProjectLogsTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	logs, ok := data["logs"].([]interface{})
	if !ok {
		t.Fatalf("Expected logs list, got %v", data["logs"])
	}
	if len(logs) != 1 {
		t.Fatalf("logs len = %d, want %d", len(logs), 1)
	}
	if requests["/workspace"] != 1 {
		t.Fatalf("workspace requests = %d, want 1", requests["/workspace"])
	}
	if requests["/workspace/fetch/"+workspaceOne] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceOne, requests["/workspace/fetch/"+workspaceOne])
	}
	if requests["/workspace/fetch/"+workspaceTwo] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceTwo, requests["/workspace/fetch/"+workspaceTwo])
	}
	if requests["/project/logs/utopian-office"] != 0 {
		t.Fatalf("project/logs/utopian-office calls = %d, want 0", requests["/project/logs/utopian-office"])
	}
	if requests["/project/logs/p2"] != 1 {
		t.Fatalf("project/logs/p2 calls = %d, want 1", requests["/project/logs/p2"])
	}
}

func TestGetProjectLogsToolRetriesWithResolvedWorkspaceForDirectIdentifier(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	workspaceOne := "5877a4ae-a891-49de-909d-0221f5eefc95"
	workspaceTwo := "6f36dd81-50e9-4ea3-8094-8e0212684a11"
	projectUUID := "0a427673-a112-47a4-ac2e-90b175fdabff"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"`+workspaceOne+`"},{"ID":2,"UUID":"`+workspaceTwo+`"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/project/logs/"+projectUUID:
				switch got := r.URL.Query().Get("workspace_uuid"); got {
				case workspaceOne:
					return jsonHTTPResponse(r, http.StatusBadRequest, `{"message":"invalid resource"}`), nil
				case workspaceTwo:
					if limit := r.URL.Query().Get("limit"); limit != "100" {
						t.Fatalf("limit = %q, want %q", limit, "100")
					}
					return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"message":"resolved direct id logs"}]}`), nil
				default:
					t.Fatalf("workspace_uuid = %q, want one of %q or %q", got, workspaceOne, workspaceTwo)
					return nil, nil
				}
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceOne:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"p1","Name":"other-project","ID":1487}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceTwo:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Projects":[{"UUID":"`+projectUUID+`","Name":"utopian-office","ID":1488}]}}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getProjectLogsTool(context.Background(), map[string]interface{}{"project_id": projectUUID, "limit": 100})
	if err != nil {
		t.Fatalf("getProjectLogsTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
	if requests["/project/logs/"+projectUUID] != 2 {
		t.Fatalf("project/logs/%s calls = %d, want 2", projectUUID, requests["/project/logs/"+projectUUID])
	}
	if requests["/workspace"] != 2 {
		t.Fatalf("workspace requests = %d, want 2", requests["/workspace"])
	}
	if requests["/workspace/fetch/"+workspaceOne] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceOne, requests["/workspace/fetch/"+workspaceOne])
	}
	if requests["/workspace/fetch/"+workspaceTwo] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceTwo, requests["/workspace/fetch/"+workspaceTwo])
	}
}

func TestGetClusterCostAllocationToolResolvesWorkspaceAcrossWorkspaces(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	workspaceOne := "5877a4ae-a891-49de-909d-0221f5eefc95"
	workspaceTwo := "6f36dd81-50e9-4ea3-8094-8e0212684a11"
	clusterUUID := "2bd58e0d-2a20-42cf-a471-d0176905bea3"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"`+workspaceOne+`"},{"ID":2,"UUID":"`+workspaceTwo+`"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceOne:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Clusters":[{"uuid":"other-cluster","name":"other"}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceTwo:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Clusters":[{"uuid":"`+clusterUUID+`","name":"target-cluster"}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/cluster/"+clusterUUID+"/cost/allocation/compute":
				if got := r.URL.Query().Get("workspace_uuid"); got != workspaceTwo {
					t.Fatalf("workspace_uuid = %q, want %q", got, workspaceTwo)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"status":"success","message":"ok","data":{"costs":{"total":12.34}}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getClusterCostAllocationTool(context.Background(), map[string]interface{}{"cluster_id": clusterUUID})
	if err != nil {
		t.Fatalf("getClusterCostAllocationTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	costs, ok := data["costs"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected costs map, got %v", data["costs"])
	}
	if got := costs["total"]; got != 12.34 {
		t.Fatalf("total = %v, want %v", got, 12.34)
	}
	if requests["/workspace"] != 1 {
		t.Fatalf("workspace requests = %d, want 1", requests["/workspace"])
	}
	if requests["/workspace/fetch/"+workspaceOne] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceOne, requests["/workspace/fetch/"+workspaceOne])
	}
	if requests["/workspace/fetch/"+workspaceTwo] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceTwo, requests["/workspace/fetch/"+workspaceTwo])
	}
	if requests["/cluster/"+clusterUUID+"/cost/allocation/compute"] != 1 {
		t.Fatalf("cluster/%s/cost/allocation/compute calls = %d, want 1", clusterUUID, requests["/cluster/"+clusterUUID+"/cost/allocation/compute"])
	}
}

func TestGetClusterCostAllocationToolUsesExplicitWorkspaceOverride(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	workspaceUUID := "5877a4ae-a891-49de-909d-0221f5eefc95"
	clusterUUID := "2bd58e0d-2a20-42cf-a471-d0176905bea3"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"`+workspaceUUID+`"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceUUID:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Clusters":[{"uuid":"`+clusterUUID+`","name":"target-cluster"}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/cluster/"+clusterUUID+"/cost/allocation/compute":
				if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
					t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"status":"success","message":"ok","data":{"costs":{"total":7.89}}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getClusterCostAllocationTool(context.Background(), map[string]interface{}{"cluster_id": clusterUUID, "workspace_id": workspaceUUID})
	if err != nil {
		t.Fatalf("getClusterCostAllocationTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
	if requests["/workspace"] != 1 {
		t.Fatalf("workspace requests = %d, want 1", requests["/workspace"])
	}
	if requests["/workspace/fetch/"+workspaceUUID] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceUUID, requests["/workspace/fetch/"+workspaceUUID])
	}
	if requests["/cluster/"+clusterUUID+"/cost/allocation/compute"] != 1 {
		t.Fatalf("cluster/%s/cost/allocation/compute calls = %d, want 1", clusterUUID, requests["/cluster/"+clusterUUID+"/cost/allocation/compute"])
	}
}

func TestGetClusterCostAllocationToolRetriesAcrossWorkspaceCandidates(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	workspaceOne := "5877a4ae-a891-49de-909d-0221f5eefc95"
	workspaceTwo := "6f36dd81-50e9-4ea3-8094-8e0212684a11"
	clusterUUID := "2bd58e0d-2a20-42cf-a471-d0176905bea3"
	attemptedWorkspaceUUIDs := make([]string, 0, 2)
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"`+workspaceOne+`"},{"ID":2,"UUID":"`+workspaceTwo+`"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceOne:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Clusters":[]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/"+workspaceTwo:
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"Clusters":[]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/cluster/"+clusterUUID+"/cost/allocation/compute":
				attemptedWorkspaceUUIDs = append(attemptedWorkspaceUUIDs, r.URL.Query().Get("workspace_uuid"))
				switch got := r.URL.Query().Get("workspace_uuid"); got {
				case workspaceOne:
					return jsonHTTPResponse(r, http.StatusBadRequest, `{"message":"invalid workspace"}`), nil
				case workspaceTwo:
					return jsonHTTPResponse(r, http.StatusOK, `{"status":"success","message":"ok","data":{"costs":{"total":4.56}}}`), nil
				default:
					t.Fatalf("workspace_uuid = %q, want one of %q or %q", got, workspaceOne, workspaceTwo)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getClusterCostAllocationTool(context.Background(), map[string]interface{}{"cluster_id": clusterUUID})
	if err != nil {
		t.Fatalf("getClusterCostAllocationTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	costs, ok := data["costs"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected costs map, got %v", data["costs"])
	}
	if got := costs["total"]; got != 4.56 {
		t.Fatalf("total = %v, want %v", got, 4.56)
	}
	if requests["/workspace"] != 2 {
		t.Fatalf("workspace requests = %d, want 2", requests["/workspace"])
	}
	if requests["/workspace/fetch/"+workspaceOne] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceOne, requests["/workspace/fetch/"+workspaceOne])
	}
	if requests["/workspace/fetch/"+workspaceTwo] != 1 {
		t.Fatalf("workspace/fetch/%s calls = %d, want 1", workspaceTwo, requests["/workspace/fetch/"+workspaceTwo])
	}
	if requests["/cluster/"+clusterUUID+"/cost/allocation/compute"] != 2 {
		t.Fatalf("cluster/%s/cost/allocation/compute calls = %d, want 2", clusterUUID, requests["/cluster/"+clusterUUID+"/cost/allocation/compute"])
	}
	if len(attemptedWorkspaceUUIDs) != 2 {
		t.Fatalf("workspace_uuid attempts = %v, want 2 attempts", attemptedWorkspaceUUIDs)
	}
	if attemptedWorkspaceUUIDs[0] != workspaceOne {
		t.Fatalf("first workspace_uuid = %q, want %q", attemptedWorkspaceUUIDs[0], workspaceOne)
	}
	if attemptedWorkspaceUUIDs[1] != workspaceTwo {
		t.Fatalf("second workspace_uuid = %q, want %q", attemptedWorkspaceUUIDs[1], workspaceTwo)
	}
}

func TestListEnvironmentsToolAggregatesAcrossWorkspacesUsingWorkspaceFetchIDFallback(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"w1"},{"ID":2,"UUID":"w2"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"workspace with uuid not found"}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/1":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"UUID":"w1","Clusters":[{"uuid":"srv1","environments":[{"UUID":"e1","Name":"prod","Namespace":"prod"}]}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w2":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"UUID":"w2","Clusters":[{"Cluster":{"uuid":"srv2","environments":[{"UUID":"e1","Name":"prod","Namespace":"prod","ClusterUUID":"srv1"},{"UUID":"e2","Name":"beta","Namespace":"beta"}]}}]}}}`), nil
			case r.URL.Path == "/environment/fetch":
				t.Fatalf("unexpected legacy environment route: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listEnvironmentsTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("listEnvironmentsTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	environments, ok := data["environments"].([]interface{})
	if !ok {
		t.Fatalf("Expected environments list, got %v", data["environments"])
	}
	if len(environments) != 2 {
		t.Fatalf("environments len = %d, want %d", len(environments), 2)
	}

	envByUUID := make(map[string]map[string]interface{}, len(environments))
	for _, item := range environments {
		env, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("expected environment map, got %T", item)
		}
		envByUUID[env["UUID"].(string)] = env
	}
	if got := envByUUID["e1"]["ClusterUUID"]; got != "srv1" {
		t.Fatalf("e1 ClusterUUID = %v, want %q", got, "srv1")
	}
	if got := envByUUID["e2"]["ClusterUUID"]; got != "srv2" {
		t.Fatalf("e2 ClusterUUID = %v, want %q", got, "srv2")
	}
	if requests["/workspace/fetch/w1"] != 1 {
		t.Fatalf("workspace/fetch/w1 calls = %d, want 1", requests["/workspace/fetch/w1"])
	}
	if requests["/workspace/fetch/1"] != 1 {
		t.Fatalf("workspace/fetch/1 calls = %d, want 1", requests["/workspace/fetch/1"])
	}
	if requests["/workspace/fetch/w2"] != 1 {
		t.Fatalf("workspace/fetch/w2 calls = %d, want 1", requests["/workspace/fetch/w2"])
	}
	if requests["/environment/fetch"] != 0 {
		t.Fatalf("environment/fetch calls = %d, want 0", requests["/environment/fetch"])
	}
}

func TestListEnvironmentsToolSkipsZeroWorkspaceIDFallback(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":0,"UUID":"w1"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"workspace with uuid not found"}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/0":
				t.Fatalf("unexpected zero workspace id fallback request: %s %s", r.Method, r.URL.Path)
				return nil, nil
			case r.URL.Path == "/environment/fetch":
				t.Fatalf("unexpected legacy environment route: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listEnvironmentsTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("listEnvironmentsTool error: %v", err)
	}
	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]interface{})
	textContent := content[0].(map[string]interface{})["text"].(string)

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &payload); err != nil {
		t.Fatalf("failed to decode result JSON: %v", err)
	}
	data := payload["data"].(map[string]interface{})
	environments := data["environments"].([]interface{})
	if len(environments) != 0 {
		t.Fatalf("environments len = %d, want %d", len(environments), 0)
	}
	if requests["/workspace/fetch/w1"] != 1 {
		t.Fatalf("workspace/fetch/w1 calls = %d, want 1", requests["/workspace/fetch/w1"])
	}
	if requests["/workspace/fetch/0"] != 0 {
		t.Fatalf("workspace/fetch/0 calls = %d, want 0", requests["/workspace/fetch/0"])
	}
	if requests["/environment/fetch"] != 0 {
		t.Fatalf("environment/fetch calls = %d, want 0", requests["/environment/fetch"])
	}
}

func TestListEnvironmentsToolExplicitWorkspaceFallsBackToWorkspaceID(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":{"workspaces":[{"id":"1","uuid":"w1"}]}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"workspace with uuid not found"}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/1":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"UUID":"w1","environments":[{"UUID":"e1","Name":"prod","Namespace":"prod","ClusterUUID":"srv1"}]}}}`), nil
			case r.URL.Path == "/environment/fetch":
				t.Fatalf("unexpected legacy environment route: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listEnvironmentsTool(context.Background(), map[string]interface{}{"workspace_id": "w1"})
	if err != nil {
		t.Fatalf("listEnvironmentsTool error: %v", err)
	}
	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]interface{})
	textContent := content[0].(map[string]interface{})["text"].(string)

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &payload); err != nil {
		t.Fatalf("failed to decode result JSON: %v", err)
	}
	data := payload["data"].(map[string]interface{})
	environments := data["environments"].([]interface{})
	if len(environments) != 1 {
		t.Fatalf("environments len = %d, want %d", len(environments), 1)
	}
	if requests["/workspace/fetch/w1"] != 1 {
		t.Fatalf("workspace/fetch/w1 calls = %d, want 1", requests["/workspace/fetch/w1"])
	}
	if requests["/workspace/fetch/1"] != 1 {
		t.Fatalf("workspace/fetch/1 calls = %d, want 1", requests["/workspace/fetch/1"])
	}
	if requests["/environment/fetch"] != 0 {
		t.Fatalf("environment/fetch calls = %d, want 0", requests["/environment/fetch"])
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
			case "/workspace":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"uuid":"5877a4ae-a891-49de-909d-0221f5eefc95"}]}`), nil
			case "/billing/balance":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				if got := r.URL.Query().Get("workspace_uuid"); got == "" {
					return jsonHTTPResponse(r, http.StatusBadRequest, `{"message":"workspace required for billing access"}`), nil
				} else if got != "5877a4ae-a891-49de-909d-0221f5eefc95" {
					t.Fatalf("workspace_uuid = %q, want %q", got, "5877a4ae-a891-49de-909d-0221f5eefc95")
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":{"Balance":"0.01","Currency":"USD"},"message":"ok","success":true}`), nil
			case "/billing/subscriptions/current":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				if got := r.URL.Query().Get("workspace_uuid"); got != "5877a4ae-a891-49de-909d-0221f5eefc95" {
					t.Fatalf("workspace_uuid = %q, want %q", got, "5877a4ae-a891-49de-909d-0221f5eefc95")
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":{"UID":"sub_123","PlanTier":"startup","PlanName":"Start-up","Amount":"34.99","BillingType":"trial","Status":"active"},"message":"ok","success":true}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
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

	if requests["/workspace"] != 1 {
		t.Fatalf("workspace requests = %d, want 1", requests["/workspace"])
	}
	if requests["/billing/balance"] != 2 {
		t.Fatalf("billing/balance requests = %d, want 2", requests["/billing/balance"])
	}
	if requests["/billing/subscriptions/current"] != 1 {
		t.Fatalf("billing/subscriptions/current requests = %d, want 1", requests["/billing/subscriptions/current"])
	}
}

func TestGetBillingInfoToolAllowsMissingCurrentSubscription(t *testing.T) {
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
			case "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"uuid":"5877a4ae-a891-49de-909d-0221f5eefc95"}]}`), nil
			case "/billing/balance":
				if got := r.URL.Query().Get("workspace_uuid"); got == "" {
					return jsonHTTPResponse(r, http.StatusBadRequest, `{"message":"workspace required for billing access"}`), nil
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":{"Balance":"10.00","Currency":"USD"},"message":"ok","success":true}`), nil
			case "/billing/subscriptions/current":
				if got := r.URL.Query().Get("workspace_uuid"); got != "5877a4ae-a891-49de-909d-0221f5eefc95" {
					t.Fatalf("workspace_uuid = %q, want %q", got, "5877a4ae-a891-49de-909d-0221f5eefc95")
				}
				return jsonHTTPResponse(r, http.StatusNotFound, `{"message":"not found"}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
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
	if requests["/workspace"] != 1 {
		t.Fatalf("workspace requests = %d, want 1", requests["/workspace"])
	}
}

func TestGetBillingInfoToolUsesExplicitWorkspaceOverride(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	workspaceUUID := "5877a4ae-a891-49de-909d-0221f5eefc95"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests[r.URL.Path]++
			switch r.URL.Path {
			case "/workspace":
				t.Fatalf("unexpected workspace lookup")
				return nil, nil
			case "/billing/balance":
				if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
					t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":{"Balance":"12.50","Currency":"USD"},"message":"ok","success":true}`), nil
			case "/billing/subscriptions/current":
				if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
					t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"data":{"UID":"sub_explicit"},"message":"ok","success":true}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getBillingInfoTool(context.Background(), map[string]interface{}{"workspace_id": workspaceUUID})
	if err != nil {
		t.Fatalf("getBillingInfoTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
	if requests["/workspace"] != 0 {
		t.Fatalf("workspace requests = %d, want 0", requests["/workspace"])
	}
	if requests["/billing/balance"] != 1 {
		t.Fatalf("billing/balance requests = %d, want 1", requests["/billing/balance"])
	}
	if requests["/billing/subscriptions/current"] != 1 {
		t.Fatalf("billing/subscriptions/current requests = %d, want 1", requests["/billing/subscriptions/current"])
	}
}

func TestListSubscriptionsToolNormalizesArrayResponse(t *testing.T) {
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
			case "/billing/subscriptions":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"UID":"sub_1","PlanTier":"startup"},{"UID":"sub_2","PlanTier":"scale"}]}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listSubscriptionsTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("listSubscriptionsTool error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
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
	subscriptions, ok := data["subscriptions"].([]interface{})
	if !ok {
		t.Fatalf("Expected subscriptions list, got %v", data["subscriptions"])
	}
	if len(subscriptions) != 2 {
		t.Fatalf("subscriptions len = %d, want %d", len(subscriptions), 2)
	}
	if requests["/billing/subscriptions"] != 1 {
		t.Fatalf("billing/subscriptions requests = %d, want 1", requests["/billing/subscriptions"])
	}
}
