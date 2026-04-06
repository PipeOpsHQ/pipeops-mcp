package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCallParams represents the parameters for a tools/call request.
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type toolDefinition struct {
	tool    Tool
	handler func(context.Context, map[string]interface{}) (interface{}, error)
}

func (s *Server) toolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			tool: Tool{
				Name:        "list_projects",
				Description: "List all projects in PipeOps",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Filter projects by workspace ID"),
					"server_id":    stringProperty("Filter projects by server ID"),
					"page":         integerProperty("Optional page number"),
					"limit":        integerProperty("Optional page size"),
				}),
			},
			handler: s.listProjectsTool,
		},
		{
			tool: Tool{
				Name:        "get_project",
				Description: "Get detailed information about a specific project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id": stringProperty("The project ID or UUID"),
				}, "project_id"),
			},
			handler: s.getProjectTool,
		},
		{
			tool: Tool{
				Name:        "create_project",
				Description: "Create a new project",
				InputSchema: objectSchema(map[string]interface{}{
					"name":           stringProperty("The project name"),
					"description":    stringProperty("Optional project description"),
					"server_id":      stringProperty("The server ID that will host the project"),
					"environment_id": stringProperty("The environment ID for the project"),
					"repository":     stringProperty("Repository URL for the project source"),
					"branch":         stringProperty("Repository branch to deploy"),
					"build_command":  stringProperty("Optional build command"),
					"start_command":  stringProperty("Optional start command"),
					"port":           integerProperty("Optional application port"),
					"framework":      stringProperty("Optional framework name"),
					"env_vars":       objectProperty("Optional environment variables keyed by name", true),
				}, "name", "server_id", "environment_id", "repository", "branch"),
			},
			handler: s.createProjectTool,
		},
		{
			tool: Tool{
				Name:        "update_project",
				Description: "Update project configuration",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":    stringProperty("The project ID or UUID"),
					"name":          stringProperty("Updated project name"),
					"description":   stringProperty("Updated project description"),
					"build_command": stringProperty("Updated build command"),
					"start_command": stringProperty("Updated start command"),
					"port":          integerProperty("Updated application port"),
				}, "project_id"),
			},
			handler: s.updateProjectTool,
		},
		{
			tool: Tool{
				Name:        "delete_project",
				Description: "Delete a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id": stringProperty("The project ID or UUID"),
				}, "project_id"),
			},
			handler: s.deleteProjectTool,
		},
		{
			tool: Tool{
				Name:        "deploy_project",
				Description: "Trigger a deployment for a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id": stringProperty("The project ID or UUID to deploy"),
				}, "project_id"),
			},
			handler: s.deployProjectTool,
		},
		{
			tool: Tool{
				Name:        "restart_project",
				Description: "Restart a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id": stringProperty("The project ID or UUID to restart"),
				}, "project_id"),
			},
			handler: s.restartProjectTool,
		},
		{
			tool: Tool{
				Name:        "stop_project",
				Description: "Stop a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id": stringProperty("The project ID or UUID to stop"),
				}, "project_id"),
			},
			handler: s.stopProjectTool,
		},
		{
			tool: Tool{
				Name:        "get_project_logs",
				Description: "Get logs for a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id": stringProperty("The project ID or UUID"),
					"start_time": stringProperty("Optional RFC3339 start time"),
					"end_time":   stringProperty("Optional RFC3339 end time"),
					"limit":      integerProperty("Optional maximum number of log entries"),
					"search":     stringProperty("Optional search text"),
				}, "project_id"),
			},
			handler: s.getProjectLogsTool,
		},
		{
			tool: Tool{
				Name:        "get_project_env_variables",
				Description: "Get environment variables for a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id": stringProperty("The project ID or UUID"),
				}, "project_id"),
			},
			handler: s.getProjectEnvVariablesTool,
		},
		{
			tool: Tool{
				Name:        "update_project_env_variables",
				Description: "Update environment variables for a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":    stringProperty("The project ID or UUID"),
					"env_variables": envVariablesProperty("Environment variables to set"),
				}, "project_id", "env_variables"),
			},
			handler: s.updateProjectEnvVariablesTool,
		},
		{
			tool: Tool{
				Name:        "deploy_project_from_image",
				Description: "Create and deploy a project from a pre-built container image",
				InputSchema: objectSchema(map[string]interface{}{
					"name":                 stringProperty("The project name"),
					"container_image":      stringProperty("The full container image reference without tag"),
					"image_tag":            stringProperty("Optional image tag, defaults to latest if omitted"),
					"external_registry_id": integerProperty("Optional external registry ID for private images"),
					"port":                 integerProperty("The container port to expose"),
					"env_variables":        envVariablesProperty("Optional environment variables to inject"),
					"replicas":             integerProperty("Optional number of replicas"),
					"vcpu":                 numberProperty("The requested vCPU allocation"),
					"memory": objectSchema(map[string]interface{}{
						"value": integerProperty("Memory amount"),
						"unit":  stringProperty("Memory unit such as MB or GB"),
					}, "value", "unit"),
					"server_id":      stringProperty("The target server or cluster ID or UUID"),
					"cluster_id":     stringProperty("Optional alias for server_id using cluster terminology"),
					"environment_id": stringProperty("The environment ID for the project"),
					"workspace_id":   stringProperty("The workspace ID that owns the deployment"),
					"preset":         stringProperty("Optional resource preset name"),
				}, "name", "container_image", "port", "vcpu", "memory", "server_id", "environment_id", "workspace_id"),
			},
			handler: s.deployProjectFromImageTool,
		},
		{
			tool: Tool{
				Name:        "create_external_registry",
				Description: "Create an external container registry configuration",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("The workspace ID that owns the registry"),
					"name":         stringProperty("Registry display name"),
					"type":         stringProperty("Registry type such as dockerhub, ghcr, ecr, or custom"),
					"username":     stringProperty("Registry username"),
					"password":     stringProperty("Registry password or access token"),
					"registry_url": stringProperty("Optional registry URL for custom registries"),
					"region":       stringProperty("Optional region, used for ECR registries"),
					"account_id":   stringProperty("Optional account ID, used for ECR registries"),
				}, "workspace_id", "name", "type", "username", "password"),
			},
			handler: s.createExternalRegistryTool,
		},
		{
			tool: Tool{
				Name:        "list_external_registries",
				Description: "List external registries for a workspace",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("The workspace ID that owns the registries"),
					"page":         integerProperty("Optional page number"),
					"limit":        integerProperty("Optional page size"),
				}, "workspace_id"),
			},
			handler: s.listExternalRegistriesTool,
		},
		{
			tool: Tool{
				Name:        "get_external_registry",
				Description: "Get detailed information about an external registry",
				InputSchema: objectSchema(map[string]interface{}{
					"registry_id": stringProperty("The external registry UID"),
				}, "registry_id"),
			},
			handler: s.getExternalRegistryTool,
		},
		{
			tool: Tool{
				Name:        "delete_external_registry",
				Description: "Delete an external registry configuration",
				InputSchema: objectSchema(map[string]interface{}{
					"registry_id": stringProperty("The external registry UID"),
				}, "registry_id"),
			},
			handler: s.deleteExternalRegistryTool,
		},
		{
			tool: Tool{
				Name:        "list_external_registry_images",
				Description: "List Docker Hub repositories available through an external registry",
				InputSchema: objectSchema(map[string]interface{}{
					"registry_id": stringProperty("The external registry UID"),
					"page":        integerProperty("Optional page number"),
					"limit":       integerProperty("Optional page size"),
				}, "registry_id"),
			},
			handler: s.listExternalRegistryImagesTool,
		},
		{
			tool: Tool{
				Name:        "list_external_registry_tags",
				Description: "List image tags for a repository in an external registry",
				InputSchema: objectSchema(map[string]interface{}{
					"registry_id": stringProperty("The external registry UID"),
					"namespace":   stringProperty("The repository namespace or organization"),
					"repository":  stringProperty("The repository name"),
					"page":        integerProperty("Optional page number"),
					"limit":       integerProperty("Optional page size"),
				}, "registry_id", "namespace", "repository"),
			},
			handler: s.listExternalRegistryTagsTool,
		},
		{
			tool: Tool{
				Name:        "search_public_registry_images",
				Description: "Search public Docker Hub images without configuring a registry",
				InputSchema: objectSchema(map[string]interface{}{
					"query": stringProperty("The public image search query"),
					"page":  integerProperty("Optional page number"),
					"limit": integerProperty("Optional page size"),
				}, "query"),
			},
			handler: s.searchPublicRegistryImagesTool,
		},
		{
			tool: Tool{
				Name:        "list_public_registry_tags",
				Description: "List tags for a public Docker Hub image",
				InputSchema: objectSchema(map[string]interface{}{
					"namespace":  stringProperty("The image namespace or organization, use library for official images"),
					"repository": stringProperty("The repository name"),
					"page":       integerProperty("Optional page number"),
					"limit":      integerProperty("Optional page size"),
				}, "namespace", "repository"),
			},
			handler: s.listPublicRegistryTagsTool,
		},
		{
			tool: Tool{
				Name:        "list_servers",
				Description: "List all servers",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID to scope the server list"),
				}),
			},
			handler: s.listServersTool,
		},
		{
			tool: Tool{
				Name:        "get_server",
				Description: "Get detailed information about a server",
				InputSchema: objectSchema(map[string]interface{}{
					"server_id":    stringProperty("The server ID or UUID"),
					"workspace_id": stringProperty("Optional workspace ID to scope the server lookup"),
				}, "server_id"),
			},
			handler: s.getServerTool,
		},
		{
			tool: Tool{
				Name:        "get_cluster_connection",
				Description: "Get connection information for a cluster",
				InputSchema: objectSchema(map[string]interface{}{
					"cluster_id": stringProperty("The cluster ID or UUID"),
					"server_id":  stringProperty("Optional alias for cluster_id using existing server terminology"),
				}, "cluster_id"),
			},
			handler: s.getClusterConnectionTool,
		},
		{
			tool: Tool{
				Name:        "get_cluster_cost_allocation",
				Description: "Get compute cost allocation for a cluster",
				InputSchema: objectSchema(map[string]interface{}{
					"cluster_id": stringProperty("The cluster ID or UUID"),
					"server_id":  stringProperty("Optional alias for cluster_id using existing server terminology"),
				}, "cluster_id"),
			},
			handler: s.getClusterCostAllocationTool,
		},
		{
			tool: Tool{
				Name:        "list_cloud_provider_regions",
				Description: "List available regions for a cloud provider",
				InputSchema: objectSchema(map[string]interface{}{
					"cloud_provider": stringProperty("The cloud provider, such as aws, gcp, huawei, or digital_ocean"),
				}, "cloud_provider"),
			},
			handler: s.listCloudProviderRegionsTool,
		},
		{
			tool: Tool{
				Name:        "list_cloud_provider_instance_categories",
				Description: "List instance categories for a cloud provider",
				InputSchema: objectSchema(map[string]interface{}{
					"cloud_provider": stringProperty("The cloud provider, such as aws, gcp, huawei, or digital_ocean"),
				}, "cloud_provider"),
			},
			handler: s.listCloudProviderInstanceCategoriesTool,
		},
		{
			tool: Tool{
				Name:        "list_cloud_provider_instance_types",
				Description: "List instance types for a cloud provider and class",
				InputSchema: objectSchema(map[string]interface{}{
					"cloud_provider": stringProperty("The cloud provider, such as aws, gcp, huawei, or digital_ocean"),
					"instance_class": stringProperty("The instance class, such as Basic or General purpose"),
					"region":         stringProperty("The provider region code"),
				}, "cloud_provider", "instance_class", "region"),
			},
			handler: s.listCloudProviderInstanceTypesTool,
		},
		{
			tool: Tool{
				Name:        "list_cloud_provider_server_templates",
				Description: "List recommended server templates for a cloud provider",
				InputSchema: objectSchema(map[string]interface{}{
					"cloud_provider": stringProperty("The cloud provider, such as aws, gcp, huawei, or digital_ocean"),
				}, "cloud_provider"),
			},
			handler: s.listCloudProviderServerTemplatesTool,
		},
		{
			tool: Tool{
				Name:        "list_environments",
				Description: "List all environments",
				InputSchema: emptySchema(),
			},
			handler: s.listEnvironmentsTool,
		},
		{
			tool: Tool{
				Name:        "get_environment",
				Description: "Get detailed information about an environment",
				InputSchema: objectSchema(map[string]interface{}{
					"environment_id": stringProperty("The environment ID or UUID"),
				}, "environment_id"),
			},
			handler: s.getEnvironmentTool,
		},
		{
			tool: Tool{
				Name:        "create_environment",
				Description: "Create a new environment",
				InputSchema: objectSchema(map[string]interface{}{
					"name":          stringProperty("The environment name"),
					"workspace_id":  stringProperty("The workspace ID that owns the environment"),
					"env_variables": envVariablesProperty("Optional environment variables to create with the environment"),
				}, "name", "workspace_id"),
			},
			handler: s.createEnvironmentTool,
		},
		{
			tool: Tool{
				Name:        "update_environment",
				Description: "Update an environment",
				InputSchema: objectSchema(map[string]interface{}{
					"environment_id": stringProperty("The environment ID or UUID"),
					"name":           stringProperty("Updated environment name"),
				}, "environment_id"),
			},
			handler: s.updateEnvironmentTool,
		},
		{
			tool: Tool{
				Name:        "delete_environment",
				Description: "Delete an environment",
				InputSchema: objectSchema(map[string]interface{}{
					"environment_id": stringProperty("The environment ID or UUID"),
				}, "environment_id"),
			},
			handler: s.deleteEnvironmentTool,
		},
		{
			tool: Tool{
				Name:        "set_environment_variables",
				Description: "Set environment variables for an environment",
				InputSchema: objectSchema(map[string]interface{}{
					"environment_id": stringProperty("The environment ID or UUID"),
					"env_variables":  envVariablesProperty("Environment variables to set"),
				}, "environment_id", "env_variables"),
			},
			handler: s.setEnvironmentVariablesTool,
		},
		{
			tool: Tool{
				Name:        "list_teams",
				Description: "List all teams",
				InputSchema: emptySchema(),
			},
			handler: s.listTeamsTool,
		},
		{
			tool: Tool{
				Name:        "create_team",
				Description: "Create a new team",
				InputSchema: objectSchema(map[string]interface{}{
					"name":        stringProperty("The team name"),
					"description": stringProperty("Optional team description"),
				}, "name"),
			},
			handler: s.createTeamTool,
		},
		{
			tool: Tool{
				Name:        "update_team",
				Description: "Update a team",
				InputSchema: objectSchema(map[string]interface{}{
					"team_id":     stringProperty("The team ID or UUID"),
					"name":        stringProperty("Updated team name"),
					"description": stringProperty("Updated team description"),
				}, "team_id"),
			},
			handler: s.updateTeamTool,
		},
		{
			tool: Tool{
				Name:        "get_team",
				Description: "Get detailed information about a team",
				InputSchema: objectSchema(map[string]interface{}{
					"team_id": stringProperty("The team ID or UUID"),
				}, "team_id"),
			},
			handler: s.getTeamTool,
		},
		{
			tool: Tool{
				Name:        "invite_team_member",
				Description: "Invite a member to a team",
				InputSchema: objectSchema(map[string]interface{}{
					"team_id":     stringProperty("The team ID or UUID"),
					"email":       stringProperty("The email address to invite"),
					"role":        stringProperty("Optional team role"),
					"permissions": stringArrayProperty("Optional team permissions"),
				}, "team_id", "email"),
			},
			handler: s.inviteTeamMemberTool,
		},
		{
			tool: Tool{
				Name:        "list_team_members",
				Description: "List members of a team",
				InputSchema: objectSchema(map[string]interface{}{
					"team_id": stringProperty("The team ID or UUID"),
				}, "team_id"),
			},
			handler: s.listTeamMembersTool,
		},
		{
			tool: Tool{
				Name:        "remove_team_member",
				Description: "Remove a member from a team",
				InputSchema: objectSchema(map[string]interface{}{
					"team_id":   stringProperty("The team ID or UUID"),
					"member_id": stringProperty("The member ID or UUID"),
				}, "team_id", "member_id"),
			},
			handler: s.removeTeamMemberTool,
		},
		{
			tool: Tool{
				Name:        "update_team_member_role",
				Description: "Update a team member role and permissions",
				InputSchema: objectSchema(map[string]interface{}{
					"team_id":     stringProperty("The team ID or UUID"),
					"member_id":   stringProperty("The member ID or UUID"),
					"role":        stringProperty("The new member role"),
					"permissions": stringArrayProperty("Optional updated permissions"),
				}, "team_id", "member_id", "role"),
			},
			handler: s.updateTeamMemberRoleTool,
		},
		{
			tool: Tool{
				Name:        "list_workspaces",
				Description: "List all workspaces",
				InputSchema: emptySchema(),
			},
			handler: s.listWorkspacesTool,
		},
		{
			tool: Tool{
				Name:        "create_workspace",
				Description: "Create a new workspace",
				InputSchema: objectSchema(map[string]interface{}{
					"name":        stringProperty("The workspace name"),
					"description": stringProperty("Optional workspace description"),
					"team_id":     stringProperty("Optional team ID to associate with the workspace"),
				}, "name"),
			},
			handler: s.createWorkspaceTool,
		},
		{
			tool: Tool{
				Name:        "update_workspace",
				Description: "Update a workspace",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("The workspace ID or UUID"),
					"name":         stringProperty("Updated workspace name"),
					"description":  stringProperty("Updated workspace description"),
				}, "workspace_id"),
			},
			handler: s.updateWorkspaceTool,
		},
		{
			tool: Tool{
				Name:        "get_workspace",
				Description: "Get detailed information about a workspace",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("The workspace ID or UUID"),
				}, "workspace_id"),
			},
			handler: s.getWorkspaceTool,
		},
		{
			tool: Tool{
				Name:        "delete_workspace",
				Description: "Delete a workspace",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("The workspace ID or UUID"),
				}, "workspace_id"),
			},
			handler: s.deleteWorkspaceTool,
		},
		{
			tool: Tool{
				Name:        "set_workspace_billing_email",
				Description: "Set the billing email for a workspace",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("The workspace ID or UUID"),
					"email":        stringProperty("The billing email address"),
				}, "workspace_id", "email"),
			},
			handler: s.setWorkspaceBillingEmailTool,
		},
		{
			tool: Tool{
				Name:        "get_current_user",
				Description: "Get the current user profile",
				InputSchema: emptySchema(),
			},
			handler: s.getCurrentUserTool,
		},
		{
			tool: Tool{
				Name:        "list_addons",
				Description: "List all available add-ons",
				InputSchema: emptySchema(),
			},
			handler: s.listAddOnsTool,
		},
		{
			tool: Tool{
				Name:        "get_addon",
				Description: "Get detailed information about an add-on",
				InputSchema: objectSchema(map[string]interface{}{
					"addon_id": stringProperty("The add-on ID or UUID"),
				}, "addon_id"),
			},
			handler: s.getAddOnTool,
		},
		{
			tool: Tool{
				Name:        "deploy_addon",
				Description: "Deploy an add-on to a project or server",
				InputSchema: objectSchema(map[string]interface{}{
					"addon_id":     stringProperty("The add-on ID or UUID to deploy"),
					"project_id":   stringProperty("Optional project ID to attach the deployment to"),
					"server_id":    stringProperty("Optional server ID to attach the deployment to"),
					"workspace_id": stringProperty("Optional workspace ID to scope the deployment; defaults to the first available workspace"),
					"config":       objectProperty("Optional deployment configuration", true),
				}, "addon_id"),
			},
			handler: s.deployAddOnTool,
		},
		{
			tool: Tool{
				Name:        "list_addon_deployments",
				Description: "List add-on deployments",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID to scope deployments; defaults to the first available workspace"),
				}),
			},
			handler: s.listAddOnDeploymentsTool,
		},
		{
			tool: Tool{
				Name:        "get_addon_deployment",
				Description: "Get detailed information about an add-on deployment",
				InputSchema: objectSchema(map[string]interface{}{
					"deployment_id": stringProperty("The add-on deployment ID or UUID"),
				}, "deployment_id"),
			},
			handler: s.getAddOnDeploymentTool,
		},
		{
			tool: Tool{
				Name:        "get_addon_deployment_session",
				Description: "Get an add-on deployment session",
				InputSchema: objectSchema(map[string]interface{}{
					"session_id": stringProperty("The deployment session ID"),
				}, "session_id"),
			},
			handler: s.getAddOnDeploymentSessionTool,
		},
		{
			tool: Tool{
				Name:        "view_addon_deployment_configs",
				Description: "View configuration for an add-on deployment",
				InputSchema: objectSchema(map[string]interface{}{
					"deployment_id": stringProperty("The add-on deployment ID or UUID"),
				}, "deployment_id"),
			},
			handler: s.viewAddOnDeploymentConfigsTool,
		},
		{
			tool: Tool{
				Name:        "add_addon_domain",
				Description: "Add a custom domain to an add-on",
				InputSchema: objectSchema(map[string]interface{}{
					"addon_id": stringProperty("The add-on ID or UUID"),
					"domain":   stringProperty("The custom domain to attach"),
				}, "addon_id", "domain"),
			},
			handler: s.addAddOnDomainTool,
		},
		{
			tool: Tool{
				Name:        "list_addon_categories",
				Description: "List add-on categories",
				InputSchema: emptySchema(),
			},
			handler: s.listAddOnCategoriesTool,
		},
		{
			tool: Tool{
				Name:        "get_my_addon_submissions",
				Description: "List the current user's add-on submissions",
				InputSchema: emptySchema(),
			},
			handler: s.getMyAddOnSubmissionsTool,
		},
		{
			tool: Tool{
				Name:        "get_billing_info",
				Description: "Get current billing balance and subscription information for the account",
				InputSchema: emptySchema(),
			},
			handler: s.getBillingInfoTool,
		},
		{
			tool: Tool{
				Name:        "list_billing_plans",
				Description: "List available billing plans",
				InputSchema: emptySchema(),
			},
			handler: s.listBillingPlansTool,
		},
		{
			tool: Tool{
				Name:        "subscribe_to_plan",
				Description: "Subscribe the current account to a billing plan",
				InputSchema: objectSchema(map[string]interface{}{
					"plan_id": stringProperty("The billing plan ID or UUID"),
				}, "plan_id"),
			},
			handler: s.subscribeToPlanTool,
		},
		{
			tool: Tool{
				Name:        "cancel_subscription",
				Description: "Cancel a billing subscription",
				InputSchema: objectSchema(map[string]interface{}{
					"subscription_id": stringProperty("The subscription ID or UUID"),
				}, "subscription_id"),
			},
			handler: s.cancelSubscriptionTool,
		},
		{
			tool: Tool{
				Name:        "add_billing_card",
				Description: "Add a billing card using a payment provider token",
				InputSchema: objectSchema(map[string]interface{}{
					"token": stringProperty("The payment provider token for the card"),
				}, "token"),
			},
			handler: s.addBillingCardTool,
		},
		{
			tool: Tool{
				Name:        "delete_billing_card",
				Description: "Delete a billing card",
				InputSchema: objectSchema(map[string]interface{}{
					"card_id": stringProperty("The billing card ID or UUID"),
				}, "card_id"),
			},
			handler: s.deleteBillingCardTool,
		},
		{
			tool: Tool{
				Name:        "create_workspace_checkout",
				Description: "Create workspace billing or checkout configuration",
				InputSchema: emptySchema(),
			},
			handler: s.createWorkspaceCheckoutTool,
		},
		{
			tool: Tool{
				Name:        "start_trial",
				Description: "Start a billing trial for a plan",
				InputSchema: objectSchema(map[string]interface{}{
					"plan_id": stringProperty("The billing plan ID or UUID"),
				}, "plan_id"),
			},
			handler: s.startTrialTool,
		},
		{
			tool: Tool{
				Name:        "get_billing_portal_url",
				Description: "Get the billing portal URL",
				InputSchema: emptySchema(),
			},
			handler: s.getBillingPortalURLTool,
		},
		{
			tool: Tool{
				Name:        "get_balance",
				Description: "Get the current account balance",
				InputSchema: emptySchema(),
			},
			handler: s.getBalanceTool,
		},
		{
			tool: Tool{
				Name:        "list_workspace_cards",
				Description: "List billing cards for the current workspace",
				InputSchema: emptySchema(),
			},
			handler: s.listWorkspaceCardsTool,
		},
		{
			tool: Tool{
				Name:        "get_active_card",
				Description: "Get the active billing card for the current workspace",
				InputSchema: emptySchema(),
			},
			handler: s.getActiveCardTool,
		},
		{
			tool: Tool{
				Name:        "list_subscriptions",
				Description: "List billing subscriptions",
				InputSchema: emptySchema(),
			},
			handler: s.listSubscriptionsTool,
		},
		{
			tool: Tool{
				Name:        "get_subscription",
				Description: "Get detailed information about a subscription",
				InputSchema: objectSchema(map[string]interface{}{
					"subscription_id": stringProperty("The subscription ID or UUID"),
				}, "subscription_id"),
			},
			handler: s.getSubscriptionTool,
		},
		{
			tool: Tool{
				Name:        "list_invoices",
				Description: "List billing invoices",
				InputSchema: emptySchema(),
			},
			handler: s.listInvoicesTool,
		},
		{
			tool: Tool{
				Name:        "list_service_account_tokens",
				Description: "List service account tokens",
				InputSchema: emptySchema(),
			},
			handler: s.listServiceAccountTokensTool,
		},
		{
			tool: Tool{
				Name:        "get_service_account_token",
				Description: "Get detailed information about a service account token",
				InputSchema: objectSchema(map[string]interface{}{
					"token_id": stringProperty("The service account token ID or UUID"),
				}, "token_id"),
			},
			handler: s.getServiceAccountTokenTool,
		},
		{
			tool: Tool{
				Name:        "create_service_account_token",
				Description: "Create a new service account token",
				InputSchema: objectSchema(map[string]interface{}{
					"name":        stringProperty("The service account token name"),
					"description": stringProperty("Optional service account token description"),
					"permissions": stringArrayProperty("Optional permissions for the token"),
					"expires_at":  stringProperty("Optional token expiration timestamp"),
				}, "name"),
			},
			handler: s.createServiceAccountTokenTool,
		},
		{
			tool: Tool{
				Name:        "update_service_account_token",
				Description: "Update a service account token",
				InputSchema: objectSchema(map[string]interface{}{
					"token_id":    stringProperty("The service account token ID or UUID"),
					"name":        stringProperty("Updated service account token name"),
					"description": stringProperty("Updated service account token description"),
					"permissions": stringArrayProperty("Updated permissions for the token"),
					"is_active":   booleanProperty("Whether the token should remain active"),
				}, "token_id"),
			},
			handler: s.updateServiceAccountTokenTool,
		},
		{
			tool: Tool{
				Name:        "revoke_service_account_token",
				Description: "Revoke a service account token",
				InputSchema: objectSchema(map[string]interface{}{
					"token_id": stringProperty("The service account token ID or UUID"),
				}, "token_id"),
			},
			handler: s.revokeServiceAccountTokenTool,
		},
	}
}

// handleToolsList returns the list of available tools.
func (s *Server) handleToolsList() interface{} {
	definitions := s.toolDefinitions()
	tools := make([]Tool, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, definition.tool)
	}

	return map[string]interface{}{"tools": tools}
}

// handleToolsCall handles a tool invocation.
func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var callParams ToolCallParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if callParams.Arguments == nil {
		callParams.Arguments = map[string]interface{}{}
	}

	for _, definition := range s.toolDefinitions() {
		if definition.tool.Name == callParams.Name {
			return definition.handler(ctx, callParams.Arguments)
		}
	}

	return nil, fmt.Errorf("unknown tool: %s", callParams.Name)
}

func emptySchema() map[string]interface{} {
	return objectSchema(map[string]interface{}{})
}

func objectSchema(properties map[string]interface{}, required ...string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

func integerProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": description,
	}
}

func numberProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "number",
		"description": description,
	}
}

func booleanProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

func stringArrayProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items": map[string]interface{}{
			"type": "string",
		},
	}
}

func objectProperty(description string, additionalProperties interface{}) map[string]interface{} {
	property := map[string]interface{}{
		"type":        "object",
		"description": description,
	}
	if additionalProperties != nil {
		property["additionalProperties"] = additionalProperties
	}
	return property
}

func envVariablesProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items": objectSchema(map[string]interface{}{
			"key":   stringProperty("Environment variable key"),
			"value": stringProperty("Environment variable value"),
		}, "key", "value"),
	}
}

func decodeArguments(args map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

func requiredString(args map[string]interface{}, key string) (string, error) {
	value, ok := args[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func (s *Server) resolveWorkspaceID(ctx context.Context, args map[string]interface{}) (string, error) {
	if workspaceID, ok := args["workspace_id"].(string); ok && workspaceID != "" {
		return workspaceID, nil
	}

	resp, _, err := s.client.Workspaces.List(ctx)
	if err != nil {
		return "", err
	}
	if len(resp.Data.Workspaces) == 1 {
		workspace := resp.Data.Workspaces[0]
		if workspace.UUID != "" {
			return workspace.UUID, nil
		}
		if workspace.ID != "" {
			return workspace.ID, nil
		}
	}

	return "", fmt.Errorf("workspace_id is required")
}

func (s *Server) resolveDefaultWorkspaceID(ctx context.Context, args map[string]interface{}) (string, error) {
	if workspaceID, ok := args["workspace_id"].(string); ok && workspaceID != "" {
		return workspaceID, nil
	}

	resp, _, err := s.client.Workspaces.List(ctx)
	if err != nil {
		return "", err
	}
	if len(resp.Data.Workspaces) == 0 {
		return "", fmt.Errorf("workspace_id is required")
	}

	workspace := resp.Data.Workspaces[0]
	if workspace.UUID != "" {
		return workspace.UUID, nil
	}
	if workspace.ID != "" {
		return workspace.ID, nil
	}

	return "", fmt.Errorf("workspace_id is required")
}

func jsonResult(v interface{}) (interface{}, error) {
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(v),
			},
		},
	}, nil
}

func (s *Server) requestJSON(ctx context.Context, method, path string, body interface{}) (map[string]interface{}, error) {
	req, err := s.client.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if _, err := s.client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func responseData(resp map[string]interface{}) interface{} {
	if data, ok := resp["data"]; ok {
		return data
	}

	return resp
}

func isHTTPStatus(err error, statusCode int) bool {
	var apiErr *pipeops.ErrorResponse
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.Response != nil && apiErr.Response.StatusCode == statusCode
}

func textResult(text string) interface{} {
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": text,
			},
		},
	}
}

type updateProjectArgs struct {
	ProjectID    string `json:"project_id"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	BuildCommand string `json:"build_command,omitempty"`
	StartCommand string `json:"start_command,omitempty"`
	Port         int    `json:"port,omitempty"`
}

type projectLogsArgs struct {
	ProjectID string `json:"project_id"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Search    string `json:"search,omitempty"`
}

type projectEnvVariablesArgs struct {
	ProjectID    string                `json:"project_id"`
	EnvVariables []pipeops.EnvVariable `json:"env_variables"`
}

type updateEnvironmentArgs struct {
	EnvironmentID string `json:"environment_id"`
	Name          string `json:"name,omitempty"`
}

type setEnvironmentVariablesArgs struct {
	EnvironmentID string                `json:"environment_id"`
	EnvVariables  []pipeops.EnvVariable `json:"env_variables"`
}

type updateWorkspaceArgs struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type setWorkspaceBillingEmailArgs struct {
	WorkspaceID string `json:"workspace_id"`
	Email       string `json:"email"`
}

type updateTeamArgs struct {
	TeamID      string `json:"team_id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type inviteTeamMemberArgs struct {
	TeamID      string   `json:"team_id"`
	Email       string   `json:"email"`
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type teamMemberArgs struct {
	TeamID   string `json:"team_id"`
	MemberID string `json:"member_id"`
}

type updateTeamMemberRoleArgs struct {
	TeamID      string   `json:"team_id"`
	MemberID    string   `json:"member_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions,omitempty"`
}

type deployAddOnArgs struct {
	AddOnID     string                 `json:"addon_id"`
	ProjectID   string                 `json:"project_id,omitempty"`
	ServerID    string                 `json:"server_id,omitempty"`
	WorkspaceID string                 `json:"workspace_id,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

type addOnDeploymentArgs struct {
	DeploymentID string `json:"deployment_id"`
}

type addOnDeploymentSessionArgs struct {
	SessionID string `json:"session_id"`
}

type addOnDomainArgs struct {
	AddOnID string `json:"addon_id"`
	Domain  string `json:"domain"`
}

type planIDArgs struct {
	PlanID string `json:"plan_id"`
}

type billingCardArgs struct {
	Token string `json:"token"`
}

type billingCardIDArgs struct {
	CardID string `json:"card_id"`
}

type updateServiceAccountTokenArgs struct {
	TokenID     string   `json:"token_id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

type deployProjectFromImageMemoryArgs struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type deployProjectFromImageArgs struct {
	Name               string                           `json:"name"`
	ContainerImage     string                           `json:"container_image"`
	ImageTag           string                           `json:"image_tag,omitempty"`
	ExternalRegistryID int                              `json:"external_registry_id,omitempty"`
	Port               int                              `json:"port"`
	EnvVariables       []pipeops.EnvVariable            `json:"env_variables,omitempty"`
	Replicas           int                              `json:"replicas,omitempty"`
	VCPU               float64                          `json:"vcpu"`
	Memory             deployProjectFromImageMemoryArgs `json:"memory"`
	ServerID           string                           `json:"server_id,omitempty"`
	ClusterID          string                           `json:"cluster_id,omitempty"`
	EnvironmentID      string                           `json:"environment_id"`
	WorkspaceID        string                           `json:"workspace_id"`
	Preset             string                           `json:"preset,omitempty"`
}

type createExternalRegistryArgs struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	RegistryURL string `json:"registry_url,omitempty"`
	Region      string `json:"region,omitempty"`
	AccountID   string `json:"account_id,omitempty"`
}

type externalRegistryListArgs struct {
	WorkspaceID string `json:"workspace_id"`
	Page        int    `json:"page,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type externalRegistryArgs struct {
	RegistryID string `json:"registry_id"`
}

type externalRegistryBrowseArgs struct {
	RegistryID string `json:"registry_id"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type externalRegistryTagsArgs struct {
	RegistryID string `json:"registry_id"`
	Namespace  string `json:"namespace"`
	Repository string `json:"repository"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type publicRegistrySearchArgs struct {
	Query string `json:"query"`
	Page  int    `json:"page,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type publicRegistryTagsArgs struct {
	Namespace  string `json:"namespace"`
	Repository string `json:"repository"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type cloudProviderArgs struct {
	CloudProvider string `json:"cloud_provider"`
}

type cloudProviderInstanceTypesArgs struct {
	CloudProvider string `json:"cloud_provider"`
	InstanceClass string `json:"instance_class"`
	Region        string `json:"region"`
}

func (s *Server) listProjectsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var opts *pipeops.ProjectListOptions
	if len(args) > 0 {
		decoded := new(pipeops.ProjectListOptions)
		if err := decodeArguments(args, decoded); err != nil {
			return nil, err
		}
		opts = decoded
	}

	if opts != nil && (opts.WorkspaceID != "" || opts.WorkspaceUUID != "") {
		resp, _, err := s.client.Projects.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return jsonResult(resp)
	}

	resp, err := s.listProjectsAcrossWorkspaces(ctx, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listProjectsAcrossWorkspaces(ctx context.Context, opts *pipeops.ProjectListOptions) (*pipeops.ProjectsResponse, error) {
	workspacesResp, _, err := s.client.Workspaces.List(ctx)
	if err != nil {
		return nil, err
	}

	projectsResp := &pipeops.ProjectsResponse{
		Status:  workspacesResp.Status,
		Message: workspacesResp.Message,
	}
	seen := make(map[string]struct{})

	for _, workspace := range workspacesResp.Data.Workspaces {
		workspaceID := workspace.UUID
		if workspaceID == "" {
			workspaceID = workspace.ID
		}
		if workspaceID == "" {
			continue
		}

		wsOpts := &pipeops.ProjectListOptions{
			WorkspaceUUID: workspaceID,
			WorkspaceID:   workspaceID,
			Limit:         1000,
		}
		if opts != nil {
			wsOpts.ServerID = opts.ServerID
		}

		workspaceProjects, _, workspaceErr := s.client.Projects.List(ctx, wsOpts)
		if workspaceErr != nil {
			return nil, workspaceErr
		}
		if projectsResp.Status == "" {
			projectsResp.Status = workspaceProjects.Status
		}
		if projectsResp.Message == "" {
			projectsResp.Message = workspaceProjects.Message
		}

		for _, project := range workspaceProjects.Data.Projects {
			projectKey := project.UUID
			if projectKey == "" {
				projectKey = project.ID.String()
			}
			if projectKey == "" {
				continue
			}
			if _, ok := seen[projectKey]; ok {
				continue
			}
			seen[projectKey] = struct{}{}
			projectsResp.Data.Projects = append(projectsResp.Data.Projects, project)
		}
	}

	if opts != nil {
		projectsResp.Data.Projects = paginateProjects(projectsResp.Data.Projects, opts.Page, opts.Limit)
	}

	return projectsResp, nil
}

func paginateProjects(projects []pipeops.Project, page, limit int) []pipeops.Project {
	if limit <= 0 {
		return projects
	}
	if page <= 0 {
		page = 1
	}

	start := (page - 1) * limit
	if start >= len(projects) {
		return []pipeops.Project{}
	}

	end := start + limit
	if end > len(projects) {
		end = len(projects)
	}

	return projects[start:end]
}

func (s *Server) getProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) createProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req pipeops.CreateProjectRequest
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.ServerID == "" {
		return nil, fmt.Errorf("server_id is required")
	}
	if req.EnvironmentID == "" {
		return nil, fmt.Errorf("environment_id is required")
	}
	if req.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}
	if req.Branch == "" {
		return nil, fmt.Errorf("branch is required")
	}

	resp, _, err := s.client.Projects.Create(ctx, &req)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req updateProjectArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	resp, _, err := s.client.Projects.Update(ctx, req.ProjectID, &pipeops.UpdateProjectRequest{
		Name:         req.Name,
		Description:  req.Description,
		BuildCommand: req.BuildCommand,
		StartCommand: req.StartCommand,
		Port:         req.Port,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deleteProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Projects.Delete(ctx, projectID); err != nil {
		return nil, err
	}
	return textResult("Project deleted successfully"), nil
}

func (s *Server) deployProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Projects.Deploy(ctx, projectID); err != nil {
		return nil, err
	}
	return textResult("Deployment triggered successfully"), nil
}

func (s *Server) deployProjectFromImageTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req deployProjectFromImageArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.ContainerImage == "" {
		return nil, fmt.Errorf("container_image is required")
	}
	if req.Port == 0 {
		return nil, fmt.Errorf("port is required")
	}
	if req.VCPU == 0 {
		return nil, fmt.Errorf("vcpu is required")
	}
	if req.Memory.Value == 0 || req.Memory.Unit == "" {
		return nil, fmt.Errorf("memory is required")
	}
	serverID := req.ServerID
	if serverID == "" {
		serverID = req.ClusterID
	}
	if serverID == "" {
		return nil, fmt.Errorf("server_id is required")
	}
	if req.EnvironmentID == "" {
		return nil, fmt.Errorf("environment_id is required")
	}
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	resp, _, err := s.client.Projects.DeployFromImage(ctx, &pipeops.DeployFromImageRequest{
		Name:               req.Name,
		ContainerImage:     req.ContainerImage,
		ImageTag:           req.ImageTag,
		ExternalRegistryID: req.ExternalRegistryID,
		Port:               req.Port,
		EnvVariables:       req.EnvVariables,
		Replicas:           req.Replicas,
		VCPU:               req.VCPU,
		Memory: pipeops.DeployFromImageMemory{
			Value: req.Memory.Value,
			Unit:  req.Memory.Unit,
		},
		ClusterUUID:     serverID,
		EnvironmentUUID: req.EnvironmentID,
		WorkspaceUUID:   req.WorkspaceID,
		Preset:          req.Preset,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) restartProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Projects.Restart(ctx, projectID); err != nil {
		return nil, err
	}
	return textResult("Project restarted successfully"), nil
}

func (s *Server) stopProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Projects.Stop(ctx, projectID); err != nil {
		return nil, err
	}
	return textResult("Project stopped successfully"), nil
}

func (s *Server) getProjectLogsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var request projectLogsArgs
	if err := decodeArguments(args, &request); err != nil {
		return nil, err
	}
	if request.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	resp, _, err := s.client.Projects.GetLogs(ctx, request.ProjectID, &pipeops.LogsOptions{
		StartTime: request.StartTime,
		EndTime:   request.EndTime,
		Limit:     request.Limit,
		Search:    request.Search,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getProjectEnvVariablesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Projects.GetEnvVariables(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateProjectEnvVariablesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var request projectEnvVariablesArgs
	if err := decodeArguments(args, &request); err != nil {
		return nil, err
	}
	if request.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	resp, _, err := s.client.Projects.UpdateEnvVariables(ctx, request.ProjectID, &pipeops.EnvVariablesRequest{
		EnvVariables: request.EnvVariables,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) createExternalRegistryTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req createExternalRegistryArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	if req.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	resp, _, err := s.client.ExternalRegistries.Create(ctx, req.WorkspaceID, &pipeops.CreateExternalRegistryRequest{
		Name:        req.Name,
		Type:        req.Type,
		Username:    req.Username,
		Password:    req.Password,
		RegistryURL: req.RegistryURL,
		Region:      req.Region,
		AccountID:   req.AccountID,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listExternalRegistriesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req externalRegistryListArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	resp, _, err := s.client.ExternalRegistries.List(ctx, req.WorkspaceID, &pipeops.ExternalRegistryListOptions{
		Page:     req.Page,
		PageSize: req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getExternalRegistryTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	registryID, err := requiredString(args, "registry_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ExternalRegistries.Get(ctx, registryID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deleteExternalRegistryTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	registryID, err := requiredString(args, "registry_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.ExternalRegistries.Delete(ctx, registryID); err != nil {
		return nil, err
	}
	return textResult("External registry deleted successfully"), nil
}

func (s *Server) listExternalRegistryImagesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req externalRegistryBrowseArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.RegistryID == "" {
		return nil, fmt.Errorf("registry_id is required")
	}

	resp, _, err := s.client.ExternalRegistries.ListDockerHubImages(ctx, req.RegistryID, &pipeops.DockerHubListOptions{
		Page:     req.Page,
		PageSize: req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listExternalRegistryTagsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req externalRegistryTagsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.RegistryID == "" {
		return nil, fmt.Errorf("registry_id is required")
	}
	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if req.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	resp, _, err := s.client.ExternalRegistries.ListDockerHubTags(ctx, req.RegistryID, req.Namespace, req.Repository, &pipeops.DockerHubListOptions{
		Page:     req.Page,
		PageSize: req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) searchPublicRegistryImagesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req publicRegistrySearchArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	resp, _, err := s.client.ExternalRegistries.SearchPublicDockerHubImages(ctx, &pipeops.DockerHubSearchOptions{
		Query:    req.Query,
		Page:     req.Page,
		PageSize: req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listPublicRegistryTagsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req publicRegistryTagsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if req.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	resp, _, err := s.client.ExternalRegistries.ListPublicDockerHubTags(ctx, req.Namespace, req.Repository, &pipeops.DockerHubListOptions{
		Page:     req.Page,
		PageSize: req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listServersTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	workspaceID, err := s.resolveWorkspaceID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Servers.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getServerTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	serverID, err := requiredString(args, "server_id")
	if err != nil {
		return nil, err
	}
	workspaceID, err := s.resolveWorkspaceID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Servers.Get(ctx, serverID, workspaceID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getClusterConnectionTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	clusterID, _ := args["cluster_id"].(string)
	if clusterID == "" {
		clusterID, _ = args["server_id"].(string)
	}
	if clusterID == "" {
		return nil, fmt.Errorf("cluster_id is required")
	}

	resp, _, err := s.client.Servers.GetClusterConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getClusterCostAllocationTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	clusterID, _ := args["cluster_id"].(string)
	if clusterID == "" {
		clusterID, _ = args["server_id"].(string)
	}
	if clusterID == "" {
		return nil, fmt.Errorf("cluster_id is required")
	}

	resp, _, err := s.client.Servers.GetClusterCostAllocation(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listCloudProviderRegionsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	cloudProvider, err := requiredString(args, "cloud_provider")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.CloudProviders.ListRegions(ctx, cloudProvider)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listCloudProviderInstanceCategoriesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	cloudProvider, err := requiredString(args, "cloud_provider")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.CloudProviders.ListInstanceCategories(ctx, cloudProvider)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listCloudProviderInstanceTypesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req cloudProviderInstanceTypesArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.CloudProvider == "" {
		return nil, fmt.Errorf("cloud_provider is required")
	}
	if req.InstanceClass == "" {
		return nil, fmt.Errorf("instance_class is required")
	}
	if req.Region == "" {
		return nil, fmt.Errorf("region is required")
	}

	resp, _, err := s.client.CloudProviders.ListInstanceTypes(ctx, req.CloudProvider, &pipeops.CloudProviderInstanceTypesOptions{
		InstanceClass: req.InstanceClass,
		Region:        req.Region,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listCloudProviderServerTemplatesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	cloudProvider, err := requiredString(args, "cloud_provider")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.CloudProviders.ListServerTemplates(ctx, cloudProvider)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listEnvironmentsTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Environments.List(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getEnvironmentTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	environmentID, err := requiredString(args, "environment_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Environments.Get(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) createEnvironmentTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req pipeops.CreateEnvironmentRequest
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	resp, _, err := s.client.Environments.Create(ctx, &req)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateEnvironmentTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req updateEnvironmentArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.EnvironmentID == "" {
		return nil, fmt.Errorf("environment_id is required")
	}

	resp, _, err := s.client.Environments.Update(ctx, req.EnvironmentID, &pipeops.UpdateEnvironmentRequest{
		Name: req.Name,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deleteEnvironmentTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	environmentID, err := requiredString(args, "environment_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Environments.Delete(ctx, environmentID); err != nil {
		return nil, err
	}
	return textResult("Environment deleted successfully"), nil
}

func (s *Server) setEnvironmentVariablesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req setEnvironmentVariablesArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.EnvironmentID == "" {
		return nil, fmt.Errorf("environment_id is required")
	}

	if _, err := s.client.Environments.SetEnvVariables(ctx, req.EnvironmentID, &pipeops.SetEnvironmentVariablesRequest{
		EnvVariables: req.EnvVariables,
	}); err != nil {
		return nil, err
	}
	return textResult("Environment variables updated successfully"), nil
}

func (s *Server) listTeamsTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Teams.List(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) createTeamTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req pipeops.CreateTeamRequest
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	resp, _, err := s.client.Teams.Create(ctx, &req)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateTeamTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req updateTeamArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.TeamID == "" {
		return nil, fmt.Errorf("team_id is required")
	}

	resp, _, err := s.client.Teams.Update(ctx, req.TeamID, &pipeops.UpdateTeamRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getTeamTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	teamID, err := requiredString(args, "team_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Teams.Get(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) inviteTeamMemberTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req inviteTeamMemberArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.TeamID == "" {
		return nil, fmt.Errorf("team_id is required")
	}
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	resp, _, err := s.client.Teams.InviteMember(ctx, req.TeamID, &pipeops.InviteTeamMemberRequest{
		Email:       req.Email,
		Role:        req.Role,
		Permissions: req.Permissions,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listTeamMembersTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	teamID, err := requiredString(args, "team_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Teams.ListMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) removeTeamMemberTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req teamMemberArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.TeamID == "" {
		return nil, fmt.Errorf("team_id is required")
	}
	if req.MemberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}

	if _, err := s.client.Teams.RemoveMember(ctx, req.TeamID, req.MemberID); err != nil {
		return nil, err
	}
	return textResult("Team member removed successfully"), nil
}

func (s *Server) updateTeamMemberRoleTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req updateTeamMemberRoleArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.TeamID == "" {
		return nil, fmt.Errorf("team_id is required")
	}
	if req.MemberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}
	if req.Role == "" {
		return nil, fmt.Errorf("role is required")
	}

	if _, err := s.client.Teams.UpdateMemberRole(ctx, req.TeamID, req.MemberID, &pipeops.UpdateMemberRoleRequest{
		Role:        req.Role,
		Permissions: req.Permissions,
	}); err != nil {
		return nil, err
	}
	return textResult("Team member role updated successfully"), nil
}

func (s *Server) listWorkspacesTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Workspaces.List(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) createWorkspaceTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req pipeops.CreateWorkspaceRequest
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	resp, _, err := s.client.Workspaces.Create(ctx, &req)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateWorkspaceTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req updateWorkspaceArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	resp, _, err := s.client.Workspaces.Update(ctx, req.WorkspaceID, &pipeops.UpdateWorkspaceRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getWorkspaceTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	workspaceID, err := requiredString(args, "workspace_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Workspaces.Get(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deleteWorkspaceTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	workspaceID, err := requiredString(args, "workspace_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Workspaces.Delete(ctx, workspaceID); err != nil {
		return nil, err
	}
	return textResult("Workspace deleted successfully"), nil
}

func (s *Server) setWorkspaceBillingEmailTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req setWorkspaceBillingEmailArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	if _, err := s.client.Workspaces.SetBillingEmail(ctx, req.WorkspaceID, &pipeops.SetBillingEmailRequest{
		Email: req.Email,
	}); err != nil {
		return nil, err
	}
	return textResult("Workspace billing email updated successfully"), nil
}

func (s *Server) getCurrentUserTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Users.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listAddOnsTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.AddOns.List(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getAddOnTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	addonID, err := requiredString(args, "addon_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.AddOns.Get(ctx, addonID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deployAddOnTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req deployAddOnArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.AddOnID == "" {
		return nil, fmt.Errorf("addon_id is required")
	}

	workspaceID, err := s.resolveDefaultWorkspaceID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.AddOns.Deploy(ctx, &pipeops.DeployAddOnRequest{
		ID:        req.AddOnID,
		ProjectID: req.ProjectID,
		Server:    req.ServerID,
		Workspace: workspaceID,
		Config:    req.Config,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listAddOnDeploymentsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	workspaceID, err := s.resolveDefaultWorkspaceID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.AddOns.ListDeployments(ctx, &pipeops.ListDeploymentsOptions{WorkspaceUUID: workspaceID})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getAddOnDeploymentTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	deploymentID, err := requiredString(args, "deployment_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.AddOns.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getAddOnDeploymentSessionTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req addOnDeploymentSessionArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	resp, _, err := s.client.AddOns.GetDeploymentSession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) viewAddOnDeploymentConfigsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req addOnDeploymentArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.DeploymentID == "" {
		return nil, fmt.Errorf("deployment_id is required")
	}

	resp, _, err := s.client.AddOns.ViewDeploymentConfigs(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) addAddOnDomainTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req addOnDomainArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.AddOnID == "" {
		return nil, fmt.Errorf("addon_id is required")
	}
	if req.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	if _, err := s.client.AddOns.AddDomain(ctx, req.AddOnID, &pipeops.DomainRequest{Domain: req.Domain}); err != nil {
		return nil, err
	}
	return textResult("Add-on domain added successfully"), nil
}

func (s *Server) listAddOnCategoriesTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.AddOns.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getMyAddOnSubmissionsTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.AddOns.GetMySubmissions(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getBillingInfoTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	balanceResp, err := s.requestJSON(ctx, http.MethodGet, "billing/balance", nil)
	if err != nil {
		return nil, err
	}

	billingInfo := map[string]interface{}{
		"success": true,
		"message": "Billing information retrieved successfully",
		"data": map[string]interface{}{
			"balance": responseData(balanceResp),
		},
	}

	currentSubscriptionResp, err := s.requestJSON(ctx, http.MethodGet, "billing/subscriptions/current", nil)
	if err != nil {
		if !isHTTPStatus(err, http.StatusNotFound) {
			return nil, err
		}
		billingInfo["data"].(map[string]interface{})["current_subscription"] = nil
		return jsonResult(billingInfo)
	}

	billingInfo["data"].(map[string]interface{})["current_subscription"] = responseData(currentSubscriptionResp)
	return jsonResult(billingInfo)
}

func (s *Server) listBillingPlansTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Billing.GetPlans(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) subscribeToPlanTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req planIDArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	resp, _, err := s.client.Billing.Subscribe(ctx, &pipeops.SubscribeRequest{PlanID: req.PlanID})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) cancelSubscriptionTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	subscriptionID, err := requiredString(args, "subscription_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Billing.CancelSubscription(ctx, subscriptionID); err != nil {
		return nil, err
	}
	return textResult("Subscription cancelled successfully"), nil
}

func (s *Server) addBillingCardTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req billingCardArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	resp, _, err := s.client.Billing.AddCard(ctx, &pipeops.AddCardRequest{Token: req.Token})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deleteBillingCardTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req billingCardIDArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.CardID == "" {
		return nil, fmt.Errorf("card_id is required")
	}

	if _, err := s.client.Billing.DeleteCard(ctx, req.CardID); err != nil {
		return nil, err
	}
	return textResult("Billing card deleted successfully"), nil
}

func (s *Server) createWorkspaceCheckoutTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	if _, err := s.client.Billing.CreateWorkspaceBilling(ctx); err != nil {
		return nil, err
	}
	return textResult("Workspace billing created successfully"), nil
}

func (s *Server) startTrialTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req planIDArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	if _, err := s.client.Billing.StartTrial(ctx, &pipeops.StartTrialRequest{PlanID: req.PlanID}); err != nil {
		return nil, err
	}
	return textResult("Trial started successfully"), nil
}

func (s *Server) getBillingPortalURLTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Billing.GetPortalURL(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getBalanceTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Billing.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listWorkspaceCardsTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Billing.ListWorkspaceCards(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getActiveCardTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Billing.GetActiveCard(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listSubscriptionsTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Billing.ListSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getSubscriptionTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	subscriptionID, err := requiredString(args, "subscription_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Billing.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listInvoicesTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.Billing.ListInvoices(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listServiceAccountTokensTool(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	resp, _, err := s.client.ServiceTokens.ListServiceAccountTokens(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getServiceAccountTokenTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	tokenID, err := requiredString(args, "token_id")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ServiceTokens.GetServiceAccountToken(ctx, tokenID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) createServiceAccountTokenTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req pipeops.ServiceAccountTokenRequest
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	resp, _, err := s.client.ServiceTokens.CreateServiceAccountToken(ctx, &req)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateServiceAccountTokenTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req updateServiceAccountTokenArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.TokenID == "" {
		return nil, fmt.Errorf("token_id is required")
	}

	resp, _, err := s.client.ServiceTokens.UpdateServiceAccountToken(ctx, req.TokenID, &pipeops.ServiceAccountTokenUpdateRequest{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		IsActive:    req.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) revokeServiceAccountTokenTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	tokenID, err := requiredString(args, "token_id")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.ServiceTokens.RevokeServiceAccountToken(ctx, tokenID); err != nil {
		return nil, err
	}
	return textResult("Service account token revoked successfully"), nil
}
