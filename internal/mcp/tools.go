package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Annotations *ToolAnnotations       `json:"annotations,omitempty"`
}

// ToolAnnotations describes operational safety hints to MCP clients.
type ToolAnnotations struct {
	ReadOnlyHint    bool  `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  bool  `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
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

type listGitOpsApplicationsArgs struct {
	Page  int `json:"page,omitempty"`
	Limit int `json:"limit,omitempty"`
}


type createGitOpsApplicationArgs struct {
	Name                string `json:"name"`
	RepoURL             string `json:"repo_url"`
	ProjectID           *uint  `json:"project_id,omitempty"`
	EnvironmentID       *uint  `json:"environment_id,omitempty"`
	Branch              string `json:"branch,omitempty"`
	Path                string `json:"path,omitempty"`
	TargetRevision      string `json:"target_revision,omitempty"`
	ManifestType        string `json:"manifest_type,omitempty"`
	HealthCheckEnabled  *bool  `json:"health_check_enabled,omitempty"`
	HealthCheckInterval int    `json:"health_check_interval,omitempty"`
	AutoSyncPrune       *bool  `json:"auto_sync_prune,omitempty"`
	AutoSyncSelfHeal    *bool  `json:"auto_sync_self_heal,omitempty"`
	AutoSyncAllowEmpty  *bool  `json:"auto_sync_allow_empty,omitempty"`
}

type updateGitOpsApplicationArgs struct {
	ApplicationUUID     string `json:"application_uuid"`
	Name                string `json:"name,omitempty"`
	Branch              string `json:"branch,omitempty"`
	Path                string `json:"path,omitempty"`
	TargetRevision      string `json:"target_revision,omitempty"`
	HealthCheckEnabled  *bool  `json:"health_check_enabled,omitempty"`
	HealthCheckInterval *int   `json:"health_check_interval,omitempty"`
	AutoSyncPrune       *bool  `json:"auto_sync_prune,omitempty"`
	AutoSyncSelfHeal    *bool  `json:"auto_sync_self_heal,omitempty"`
	AutoSyncAllowEmpty  *bool  `json:"auto_sync_allow_empty,omitempty"`
}

type syncGitOpsApplicationArgs struct {
	ApplicationUUID string `json:"application_uuid"`
	Revision        string `json:"revision,omitempty"`
	Prune           bool   `json:"prune,omitempty"`
	DryRun          bool   `json:"dry_run,omitempty"`
}

type getGitOpsHistoryArgs struct {
	ApplicationUUID string `json:"application_uuid"`
	Page            int    `json:"page,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type listProjectGroupsArgs struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}


type createProjectGroupArgs struct {
	Name                   string `json:"name"`
	DefaultClusterUUID     string `json:"default_cluster_uuid,omitempty"`
	DefaultEnvironmentUUID string `json:"default_environment_uuid,omitempty"`
	WorkspaceID            string `json:"workspace_id,omitempty"`
}

type updateProjectGroupArgs struct {
	GroupUUID              string  `json:"group_uuid"`
	Name                   *string `json:"name,omitempty"`
	DefaultClusterUUID     *string `json:"default_cluster_uuid,omitempty"`
	DefaultEnvironmentUUID *string `json:"default_environment_uuid,omitempty"`
	WorkspaceID            string  `json:"workspace_id,omitempty"`
}

type attachProjectGroupMemberArgs struct {
	GroupUUID      string `json:"group_uuid"`
	MemberType     string `json:"member_type"`
	MemberUUID     string `json:"member_uuid"`
	IncludeSession *bool  `json:"include_session,omitempty"`
	Move           bool   `json:"move,omitempty"`
	WorkspaceID    string `json:"workspace_id,omitempty"`
}

type detachProjectGroupMemberArgs struct {
	GroupUUID      string `json:"group_uuid"`
	MemberType     string `json:"member_type"`
	MemberUUID     string `json:"member_uuid"`
	IncludeSession *bool  `json:"include_session,omitempty"`
	WorkspaceID    string `json:"workspace_id,omitempty"`
}

type putProjectGroupSharedEnvArgs struct {
	GroupUUID      string                             `json:"group_uuid"`
	Variables      []pipeops.ProjectGroupSharedEnvVar `json:"variables"`
	Inject         bool                               `json:"inject,omitempty"`
	Overwrite      bool                               `json:"overwrite,omitempty"`
	Redeploy       bool                               `json:"redeploy,omitempty"`
	KeepReferences bool                               `json:"keep_references,omitempty"`
	WorkspaceID    string                             `json:"workspace_id,omitempty"`
}

type injectProjectGroupSharedEnvArgs struct {
	GroupUUID      string   `json:"group_uuid"`
	Overwrite      bool     `json:"overwrite,omitempty"`
	Redeploy       bool     `json:"redeploy,omitempty"`
	MemberUUIDs    []string `json:"member_uuids,omitempty"`
	KeepReferences bool     `json:"keep_references,omitempty"`
	WorkspaceID    string   `json:"workspace_id,omitempty"`
}

type connectProjectGroupServicesArgs struct {
	GroupUUID    string `json:"group_uuid"`
	ConsumerType string `json:"consumer_type"`
	ConsumerUUID string `json:"consumer_uuid"`
	ProviderType string `json:"provider_type"`
	ProviderUUID string `json:"provider_uuid"`
	Overwrite    bool   `json:"overwrite,omitempty"`
	VariableSet  string `json:"variable_set,omitempty"`
	WorkspaceID  string `json:"workspace_id,omitempty"`
}

type resolveProjectGroupMemberArgs struct {
	MemberType  string `json:"member_type"`
	MemberUUID  string `json:"member_uuid"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type listProjectGroupCandidatesArgs struct {
	GroupUUID   string `json:"group_uuid,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

func gitOpsAutomatedSyncPolicy(prune, selfHeal, allowEmpty *bool) *pipeops.GitOpsSyncPolicyRequest {
	if prune == nil && selfHeal == nil && allowEmpty == nil {
		return nil
	}
	automated := &pipeops.GitOpsAutomatedSyncRequest{}
	if prune != nil {
		automated.Prune = *prune
	}
	if selfHeal != nil {
		automated.SelfHeal = *selfHeal
	}
	if allowEmpty != nil {
		automated.AllowEmpty = *allowEmpty
	}
	return &pipeops.GitOpsSyncPolicyRequest{Automated: automated}
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
					"project_id":   stringProperty("The project ID, UUID, name, or slug"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "project_id"),
			},
			handler: s.getProjectTool,
		},
		{
			tool: Tool{
				Name: "create_project",
				Description: "Create and queue deploy of a project (POST /project/create). " +
					"Matches the dashboard create contract. Prefer-client: any field you send is used as-is; " +
					"only empty fields get defaults (workspace→first workspace, environment→development, " +
					"source→github, PORT from port if env_vars lacks PORT, network protocol→HTTP, worker→false). " +
					"Required for a standard git web app: name, username, source, repository, branch, " +
					"cluster_uuid, environment_uuid, build_method (or build_settings), and port for web apps. " +
					"Strongly recommended: commit_sha, commit_url, language (e.g. dockerfile|nodejs). " +
					"Do not invent K8s secret names; runner creates {name}-{namespace}-secret automatically.",
				InputSchema: objectSchema(map[string]interface{}{
					"name":             stringProperty("Project name (K8s-safe; spaces become dashes server-side)"),
					"username":         stringProperty("VCS username/org that owns the repository (required for git)"),
					"source":           stringProperty("VCS provider: github | gitlab | bitbucket | image. Default github if omitted"),
					"repository":       stringProperty("Full repository URL"),
					"branch":           stringProperty("Git branch to deploy"),
					"cluster_uuid":     stringProperty("Target cluster/server UUID (maps to clusterUUID)"),
					"clusterUUID":      stringProperty("Alias for cluster_uuid"),
					"server_id":        stringProperty("Legacy alias for cluster_uuid"),
					"environment_uuid": stringProperty("Environment UUID (controller resolves namespace from this)"),
					"environment_id":   stringProperty("Legacy alias for environment_uuid"),
					"environment":      stringProperty("Environment name/slug for display. Default development if omitted"),
					"workspace_id":     stringProperty("Workspace UUID/ID. Default: first workspace if omitted"),
					"workspace_uuid":   stringProperty("Alias for workspace_id (body workspace_uuid)"),
					"commit_url":       stringProperty("Commit URL (dashboard sends real commit link; default repository URL if empty)"),
					"commit_sha":       stringProperty("Commit SHA to build (prefer real SHA from list_vcs_branches / git)"),
					"language":         stringProperty("repositoryLanguage, e.g. nodejs, go, dockerfile"),
					"framework":        stringProperty("Optional framework label"),
					"build_method":     stringProperty("buildSettings.buildMethod, e.g. nodejs, docker, go"),
					"build_command":    stringProperty("buildSettings.buildCommand"),
					"run_command":      stringProperty("buildSettings.runCommand (process start)"),
					"start_command":    stringProperty("Legacy alias for run_command"),
					"port":             integerProperty("App port → networkSettings[].Port; also seeds PORT env if env_vars has no PORT"),
					"protocol":         stringProperty("Network protocol (default HTTP if port set and protocol omitted)"),
					"env_vars":         objectProperty("Environment variables map. Client values win; PORT only added if missing and port is set", true),
					// Nested object MUST use properties (not additionalProperties). Passing a
					// property map as additionalProperties breaks JSON Schema draft 2020-12
					// (Claude rejects it): the nested key "type" collides with the schema keyword.
					"build_settings": nestedObjectProperty(
						"Nested buildSettings. Explicit fields override top-level build_method/build_command/run_command when both are set in handler merge order",
						map[string]interface{}{
							"type":             stringProperty("Build settings type (e.g. user)"),
							"build_method":     stringProperty("Build method (e.g. nodejs, docker)"),
							"buildMethod":      stringProperty("Alias for build_method"),
							"build_command":    stringProperty("Build command"),
							"buildCommand":     stringProperty("Alias for build_command"),
							"run_command":      stringProperty("Run/start command"),
							"runCommand":       stringProperty("Alias for run_command"),
							"worker":           booleanProperty("Worker project (default false if omitted)"),
							"build_path":       stringProperty("Build path"),
							"build_directory":  stringProperty("Build directory"),
							"build_version":    stringProperty("Build version / runtime image tag"),
							"docker_path":      stringProperty("Dockerfile path"),
							"docker_image_url": stringProperty("Prebuilt docker image URL"),
							"dockerImageURL":   stringProperty("Alias for docker_image_url"),
						},
					),
				}, "name", "cluster_uuid", "environment_uuid", "repository", "branch", "source", "username"),
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
				Name: "deploy_project",
				Description: "Trigger a deployment for an existing project (prefer-client: " +
					"only project_id is required; the control plane fills build, source, network, " +
					"and env from the stored project). Optional workspace_id scopes the call; " +
					"no_cache forces a clean rebuild.",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":   stringProperty("The project ID or UUID to deploy"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
					"no_cache":     booleanProperty("Force a clean rebuild without cached build layers"),
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
					"project_id":   stringProperty("The project ID, UUID, name, or slug"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
					"start_time":   stringProperty("Optional RFC3339 start time"),
					"end_time":     stringProperty("Optional RFC3339 end time"),
					"limit":        integerProperty("Optional maximum number of log entries"),
					"search":       stringProperty("Optional search text"),
				}, "project_id"),
			},
			handler: s.getProjectLogsTool,
		},
		{
			tool: Tool{
				Name:        "list_project_deployments",
				Description: "List build or git deployments for a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":   stringProperty("The project ID, UUID, name, or slug"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override for project resolution"),
					"filter_by":    stringProperty("Optional deployment view: build or git"),
					"page":         integerProperty("Optional page number"),
					"limit":        integerProperty("Optional page size"),
				}, "project_id"),
			},
			handler: s.listProjectDeploymentsTool,
		},
		{
			tool: Tool{
				Name:        "list_project_deployment_history",
				Description: "List deployment history records for a project",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":   stringProperty("The project ID, UUID, name, or slug"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override for project resolution"),
					"page":         integerProperty("Optional page number"),
					"limit":        integerProperty("Optional page size"),
				}, "project_id"),
			},
			handler: s.listProjectDeploymentHistoryTool,
		},
		{
			tool: Tool{
				Name:        "search_project_deployments",
				Description: "Search project deployments for a project by SHA, status, commit message, URL, or related values",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":   stringProperty("The project ID, UUID, name, or slug"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override for project resolution"),
					"filter_by":    stringProperty("Optional deployment view: build or git"),
					"search":       stringProperty("Search text to match against deployment values"),
					"page":         integerProperty("Optional page number"),
					"limit":        integerProperty("Optional page size"),
				}, "project_id", "search"),
			},
			handler: s.searchProjectDeploymentsTool,
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
				Name: "update_project_env_variables",
				Description: "Update project environment variables (prefer-client). " +
					"By default merge=true so keys overlay existing envs without wiping others; " +
					"set merge=false for full replace (dashboard-style). Control plane injects PORT " +
					"from network when missing.",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":    stringProperty("The project ID or UUID"),
					"env_variables": envVariablesProperty("Environment variables to set (client values win)"),
					"merge":         booleanProperty("When true (default), merge into existing envs; false = full replace"),
					"workspace_id":  stringProperty("Optional workspace ID or UUID"),
				}, "project_id", "env_variables"),
			},
			handler: s.updateProjectEnvVariablesTool,
		},
		{
			tool: Tool{
				Name: "update_project_deploy_settings",
				Description: "Update project deploy/source-control settings (prefer-client). " +
					"Only project_id is required; omit branch/repository/username/auto flags to keep " +
					"stored values. Client-provided fields always win. Triggers a settings deployment " +
					"(rebuild only if source fields change).",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":          stringProperty("The project ID or UUID"),
					"workspace_id":        stringProperty("Optional workspace ID or UUID"),
					"branch":              stringProperty("Git branch (omitted = keep current)"),
					"repository":          stringProperty("Repository name/path (omitted = keep current)"),
					"username":            stringProperty("VCS owner/username (omitted = keep current)"),
					"auto_deploy_enabled": booleanProperty("Enable auto-deploy on push (omitted = keep current)"),
					"auto_rollback":       booleanProperty("Enable auto-rollback (omitted = keep current)"),
				}, "project_id"),
			},
			handler: s.updateProjectDeploySettingsTool,
		},
		{
			tool: Tool{
				Name: "update_project_security_policy",
				Description: "Update project image-scan security policy (prefer-client partial update). " +
					"Only send fields to change; omitted thresholds keep stored values. " +
					"Explicit zeros (e.g. max_critical=0) are applied when provided.",
				InputSchema: objectSchema(map[string]interface{}{
					"project_id":      stringProperty("The project ID or UUID"),
					"workspace_id":    stringProperty("Optional workspace ID or UUID"),
					"enabled":         booleanProperty("Enable image-scan gate (omitted = keep current)"),
					"max_critical":    integerProperty("Max critical vulns allowed (omitted = keep; 0 when sent is intentional)"),
					"max_high":        integerProperty("Max high vulns allowed"),
					"max_medium":      integerProperty("Max medium vulns allowed"),
					"max_cvss_score":  numberProperty("Max CVSS score allowed"),
					"max_total_vulns": integerProperty("Max total vulns allowed"),
					"fail_on_secrets": booleanProperty("Fail when secrets are detected"),
				}, "project_id"),
			},
			handler: s.updateProjectSecurityPolicyTool,
		},
		{
			tool: Tool{
				Name:        "list_vcs_organizations",
				Description: "List linked VCS organizations or personal profiles for a provider",
				InputSchema: objectSchema(map[string]interface{}{
					"provider": stringProperty("The VCS provider: github, gitlab, bitbucket, or azuredevops"),
				}, "provider"),
			},
			handler: s.listVCSOrganizationsTool,
		},
		{
			tool: Tool{
				Name:        "list_vcs_repositories",
				Description: "List repositories for a VCS organization or personal profile",
				InputSchema: objectSchema(map[string]interface{}{
					"provider": stringProperty("The VCS provider: github, gitlab, bitbucket, or azuredevops"),
					"org_name": stringProperty("The organization or personal profile name"),
					"page":     integerProperty("Optional page number"),
				}, "provider", "org_name"),
			},
			handler: s.listVCSRepositoriesTool,
		},
		{
			tool: Tool{
				Name:        "search_vcs_repositories",
				Description: "Search repositories within a VCS organization or personal profile",
				InputSchema: objectSchema(map[string]interface{}{
					"provider":        stringProperty("The VCS provider: github, gitlab, bitbucket, or azuredevops"),
					"org_name":        stringProperty("The organization or personal profile name"),
					"repository_name": stringProperty("The repository name to search for"),
					"page":            integerProperty("Optional page number"),
				}, "provider", "org_name", "repository_name"),
			},
			handler: s.searchVCSRepositoriesTool,
		},
		{
			tool: Tool{
				Name:        "list_vcs_branches",
				Description: "List branches for a repository in a VCS provider",
				InputSchema: objectSchema(map[string]interface{}{
					"provider":      stringProperty("The VCS provider: github, gitlab, bitbucket, or azuredevops"),
					"repo_fullname": stringProperty("Repository full name such as owner/repository"),
					"visibility":    stringProperty("Optional repository visibility, such as public or private"),
					"search":        stringProperty("Optional branch name search text"),
				}, "provider", "repo_fullname"),
			},
			handler: s.listVCSBranchesTool,
		},
		{
			tool: Tool{
				Name:        "check_repository_dockerfile",
				Description: "Check whether a repository branch contains a Dockerfile",
				InputSchema: objectSchema(map[string]interface{}{
					"provider":   stringProperty("The VCS provider: github, gitlab, bitbucket, or azuredevops"),
					"owner":      stringProperty("The repository owner, namespace, or workspace"),
					"repository": stringProperty("The repository name"),
					"branch":     stringProperty("The branch name"),
				}, "provider", "owner", "repository", "branch"),
			},
			handler: s.checkRepositoryDockerfileTool,
		},
		{
			tool: Tool{
				Name:        "link_vcs_provider",
				Description: "Initiate VCS provider linking and return the authorization URL",
				InputSchema: objectSchema(map[string]interface{}{
					"provider":      stringProperty("The VCS provider: github, gitlab, bitbucket, or azuredevops"),
					"redirect_path": stringProperty("Frontend redirect path to continue the provider auth flow"),
				}, "provider", "redirect_path"),
			},
			handler: s.linkVCSProviderTool,
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
					"server_id":      stringProperty("The target server ID or UUID"),
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
				Name:        "get_server_connection",
				Description: "Get connection information for a server",
				InputSchema: objectSchema(map[string]interface{}{
					"server_id": stringProperty("The server ID or UUID"),
				}, "server_id"),
			},
			handler: s.getClusterConnectionTool,
		},
		{
			tool: Tool{
				Name:        "get_server_cost_allocation",
				Description: "Get compute cost allocation for a server",
				InputSchema: objectSchema(map[string]interface{}{
					"server_id":    stringProperty("The server ID or UUID"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
					"aggregate":    stringProperty("Optional cost aggregation, defaults to namespace"),
					"window":       stringProperty("Optional time window such as 30d, defaults to 30d"),
					"location":     stringProperty("Optional billing location such as NGR or USA, used for nova server cost fallback"),
				}, "server_id"),
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
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID or UUID filter"),
				}),
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
				Description: "List available add-ons with optional search and filters",
				InputSchema: objectSchema(map[string]interface{}{
					"page":         integerProperty("Optional page number"),
					"limit":        integerProperty("Optional page size"),
					"category":     stringProperty("Optional add-on category filter"),
					"search":       stringProperty("Optional add-on search text"),
					"featured":     booleanProperty("Optional featured add-on filter"),
					"workspace_id": stringProperty("Optional workspace ID or UUID to scope results"),
				}),
			},
			handler: s.listAddOnsTool,
		},
		{
			tool: Tool{
				Name:        "search_addons",
				Description: "Search available add-ons",
				InputSchema: objectSchema(map[string]interface{}{
					"search":       stringProperty("The add-on search text"),
					"page":         integerProperty("Optional page number"),
					"limit":        integerProperty("Optional page size"),
					"category":     stringProperty("Optional add-on category filter"),
					"featured":     booleanProperty("Optional featured add-on filter"),
					"workspace_id": stringProperty("Optional workspace ID or UUID to scope results"),
				}, "search"),
			},
			handler: s.searchAddOnsTool,
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
				Name: "deploy_addon",
				Description: "Deploy a marketplace add-on (prefer-client: only addon_id + server_id required; " +
					"control plane fills Config from catalog, environment from cluster defaults when omitted). " +
					"Optional config overlays catalog defaults without wiping them.",
				InputSchema: objectSchema(map[string]interface{}{
					"addon_id":       stringProperty("The add-on marketplace UID to deploy"),
					"server_id":      stringProperty("Cluster/server UUID to deploy onto"),
					"workspace_id":   stringProperty("Optional workspace UUID; defaults to the first available workspace"),
					"environment_id": stringProperty("Optional environment UUID; defaults to first env on the cluster"),
					"project_id":     stringProperty("Optional project ID (placement hint)"),
					"tag":            stringProperty("Optional version/tag override"),
					"config":         objectProperty("Optional partial deployment configuration (merged over catalog)", true),
				}, "addon_id", "server_id"),
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
				Name:        "list_addon_backups",
				Description: "List backup snapshots for an add-on deployment",
				InputSchema: objectSchema(map[string]interface{}{
					"deployment_id": stringProperty("The add-on deployment ID or UUID"),
				}, "deployment_id"),
			},
			handler: s.listAddonBackupsTool,
		},
		{
			tool: Tool{
				Name:        "start_addon_backup_export",
				Description: "Start an async export of an add-on backup snapshot",
				InputSchema: objectSchema(map[string]interface{}{
					"deployment_id": stringProperty("The add-on deployment ID or UUID"),
					"snapshot_id":   stringProperty("The backup snapshot ID to export"),
					"path":          stringProperty("Optional snapshot path to export"),
					"format":        stringProperty("Optional export format: auto, sql, rdb, or archive"),
				}, "deployment_id", "snapshot_id"),
			},
			handler: s.startAddonBackupExportTool,
		},
		{
			tool: Tool{
				Name:        "get_addon_backup_export",
				Description: "Get status for an add-on backup export job",
				InputSchema: objectSchema(map[string]interface{}{
					"deployment_id": stringProperty("The add-on deployment ID or UUID"),
					"export_id":     stringProperty("The backup export job ID"),
				}, "deployment_id", "export_id"),
			},
			handler: s.getAddonBackupExportTool,
		},
		{
			tool: Tool{
				Name:        "list_volumes",
				Description: "List workspace volumes with optional status and cluster filters",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID or UUID; defaults to the first available workspace"),
					"status":       stringProperty("Optional volume status filter (e.g. mounted, unattached)"),
					"cluster_uuid": stringProperty("Optional cluster/server UUID filter"),
					"limit":        integerProperty("Optional page size"),
					"offset":       integerProperty("Optional page offset"),
				}),
			},
			handler: s.listVolumesTool,
		},
		{
			tool: Tool{
				Name:        "get_volume",
				Description: "Get detailed information about a workspace volume",
				InputSchema: objectSchema(map[string]interface{}{
					"volume_uuid":  stringProperty("The volume UUID"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "volume_uuid"),
			},
			handler: s.getVolumeTool,
		},
		{
			tool: Tool{
				Name:        "remount_volume",
				Description: "Remount an unattached volume onto a project or add-on",
				InputSchema: objectSchema(map[string]interface{}{
					"volume_uuid":  stringProperty("The volume UUID to remount"),
					"target_type":  stringProperty("Remount target type: project or addon"),
					"target_uuid":  stringProperty("The project or add-on UUID to mount onto"),
					"mount_path":   stringProperty("Optional mount path inside the target"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "volume_uuid", "target_type", "target_uuid"),
			},
			handler: s.remountVolumeTool,
		},
		{
			tool: Tool{
				Name:        "delete_volume",
				Description: "Permanently delete a workspace volume",
				InputSchema: objectSchema(map[string]interface{}{
					"volume_uuid":  stringProperty("The volume UUID to delete"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "volume_uuid"),
			},
			handler: s.deleteVolumeTool,
		},
		{
			tool: Tool{
				Name:        "export_volume",
				Description: "Start an async export of a workspace volume",
				InputSchema: objectSchema(map[string]interface{}{
					"volume_uuid":  stringProperty("The volume UUID to export"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "volume_uuid"),
			},
			handler: s.exportVolumeTool,
		},
		{
			tool: Tool{
				Name:        "get_volume_export",
				Description: "Get status for a workspace volume export job",
				InputSchema: objectSchema(map[string]interface{}{
					"volume_uuid":  stringProperty("The volume UUID whose export status to fetch"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "volume_uuid"),
			},
			handler: s.getVolumeExportTool,
		},
		{
			tool: Tool{
				Name:        "get_billing_info",
				Description: "Get current billing balance and subscription information for the account",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}),
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
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}),
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
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}),
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
				Name:        "list_gitops_applications",
				Description: "List GitOps application configurations for the current workspace session",
				InputSchema: objectSchema(map[string]interface{}{
					"page":  integerProperty("Optional page number"),
					"limit": integerProperty("Optional page size"),
				}),
			},
			handler: s.listGitOpsApplicationsTool,
		},
		{
			tool: Tool{
				Name:        "get_gitops_application",
				Description: "Get a GitOps application configuration by UUID",
				InputSchema: objectSchema(map[string]interface{}{
					"application_uuid": stringProperty("The GitOps application UUID"),
				}, "application_uuid"),
			},
			handler: s.getGitOpsApplicationTool,
		},
		{
			tool: Tool{
				Name:        "create_gitops_application",
				Description: "Create a GitOps application configuration",
				InputSchema: objectSchema(map[string]interface{}{
					"name":                  stringProperty("GitOps application name"),
					"repo_url":              stringProperty("Git repository URL"),
					"project_id":            integerProperty("Optional project ID to bind"),
					"environment_id":        integerProperty("Optional environment ID to bind"),
					"branch":                stringProperty("Optional git branch"),
					"path":                  stringProperty("Optional path within the repository"),
					"target_revision":       stringProperty("Optional target revision (branch, tag, or commit)"),
					"manifest_type":         stringProperty("Optional manifest type: pipeops or kubernetes"),
					"health_check_enabled":  booleanProperty("Optional health check enable flag"),
					"health_check_interval": integerProperty("Optional health check interval in seconds"),
					"auto_sync_prune":       booleanProperty("Optional automated sync prune setting"),
					"auto_sync_self_heal":   booleanProperty("Optional automated sync self-heal setting"),
					"auto_sync_allow_empty": booleanProperty("Optional automated sync allow-empty setting"),
				}, "name", "repo_url"),
			},
			handler: s.createGitOpsApplicationTool,
		},
		{
			tool: Tool{
				Name:        "update_gitops_application",
				Description: "Update a GitOps application configuration",
				InputSchema: objectSchema(map[string]interface{}{
					"application_uuid":      stringProperty("The GitOps application UUID"),
					"name":                  stringProperty("Updated application name"),
					"branch":                stringProperty("Updated git branch"),
					"path":                  stringProperty("Updated path within the repository"),
					"target_revision":       stringProperty("Updated target revision"),
					"health_check_enabled":  booleanProperty("Updated health check enable flag"),
					"health_check_interval": integerProperty("Updated health check interval in seconds"),
					"auto_sync_prune":       booleanProperty("Updated automated sync prune setting"),
					"auto_sync_self_heal":   booleanProperty("Updated automated sync self-heal setting"),
					"auto_sync_allow_empty": booleanProperty("Updated automated sync allow-empty setting"),
				}, "application_uuid"),
			},
			handler: s.updateGitOpsApplicationTool,
		},
		{
			tool: Tool{
				Name:        "delete_gitops_application",
				Description: "Delete a GitOps application configuration",
				InputSchema: objectSchema(map[string]interface{}{
					"application_uuid": stringProperty("The GitOps application UUID to delete"),
				}, "application_uuid"),
			},
			handler: s.deleteGitOpsApplicationTool,
		},
		{
			tool: Tool{
				Name:        "sync_gitops_application",
				Description: "Trigger a manual sync for a GitOps application",
				InputSchema: objectSchema(map[string]interface{}{
					"application_uuid": stringProperty("The GitOps application UUID"),
					"revision":         stringProperty("Optional revision to sync"),
					"prune":            booleanProperty("Whether to prune resources during sync"),
					"dry_run":          booleanProperty("Whether to perform a dry-run sync"),
				}, "application_uuid"),
			},
			handler: s.syncGitOpsApplicationTool,
		},
		{
			tool: Tool{
				Name:        "get_gitops_sync_status",
				Description: "Get the current sync and health status for a GitOps application",
				InputSchema: objectSchema(map[string]interface{}{
					"application_uuid": stringProperty("The GitOps application UUID"),
				}, "application_uuid"),
			},
			handler: s.getGitOpsSyncStatusTool,
		},
		{
			tool: Tool{
				Name:        "get_gitops_diff",
				Description: "Get the git vs live-state diff for a GitOps application",
				InputSchema: objectSchema(map[string]interface{}{
					"application_uuid": stringProperty("The GitOps application UUID"),
				}, "application_uuid"),
			},
			handler: s.getGitOpsDiffTool,
		},
		{
			tool: Tool{
				Name:        "get_gitops_history",
				Description: "Get paginated sync history for a GitOps application",
				InputSchema: objectSchema(map[string]interface{}{
					"application_uuid": stringProperty("The GitOps application UUID"),
					"page":             integerProperty("Optional page number"),
					"limit":            integerProperty("Optional page size"),
				}, "application_uuid"),
			},
			handler: s.getGitOpsHistoryTool,
		},
		{
			tool: Tool{
				Name:        "list_project_groups",
				Description: "List project groups (unified project plane) for a workspace",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID or UUID; defaults to the first available workspace"),
					"limit":        integerProperty("Optional page size"),
					"offset":       integerProperty("Optional page offset"),
				}),
			},
			handler: s.listProjectGroupsTool,
		},
		{
			tool: Tool{
				Name:        "get_project_group",
				Description: "Get a project group by UUID",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":   stringProperty("The project group UUID"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid"),
			},
			handler: s.getProjectGroupTool,
		},
		{
			tool: Tool{
				Name:        "create_project_group",
				Description: "Create an empty project group",
				InputSchema: objectSchema(map[string]interface{}{
					"name":                     stringProperty("Project group name"),
					"default_cluster_uuid":     stringProperty("Optional default cluster UUID"),
					"default_environment_uuid": stringProperty("Optional default environment UUID"),
					"workspace_id":             stringProperty("Optional workspace ID or UUID override"),
				}, "name"),
			},
			handler: s.createProjectGroupTool,
		},
		{
			tool: Tool{
				Name:        "update_project_group",
				Description: "Update project group metadata",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":               stringProperty("The project group UUID"),
					"name":                     stringProperty("Updated project group name"),
					"default_cluster_uuid":     stringProperty("Updated default cluster UUID"),
					"default_environment_uuid": stringProperty("Updated default environment UUID"),
					"workspace_id":             stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid"),
			},
			handler: s.updateProjectGroupTool,
		},
		{
			tool: Tool{
				Name:        "delete_project_group",
				Description: "Delete a project group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":   stringProperty("The project group UUID to delete"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid"),
			},
			handler: s.deleteProjectGroupTool,
		},
		{
			tool: Tool{
				Name:        "attach_project_group_member",
				Description: "Attach a project or add-on deployment member to a project group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":      stringProperty("The project group UUID"),
					"member_type":     stringProperty("Member type: project or addon_deployment"),
					"member_uuid":     stringProperty("The member project or add-on deployment UUID"),
					"include_session": booleanProperty("Optional flag to include the member session"),
					"move":            booleanProperty("Whether to move the member from another group if already attached"),
					"workspace_id":    stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid", "member_type", "member_uuid"),
			},
			handler: s.attachProjectGroupMemberTool,
		},
		{
			tool: Tool{
				Name:        "detach_project_group_member",
				Description: "Detach a member from a project group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":      stringProperty("The project group UUID"),
					"member_type":     stringProperty("Member type: project or addon_deployment"),
					"member_uuid":     stringProperty("The member project or add-on deployment UUID"),
					"include_session": booleanProperty("Optional flag to include the member session when detaching"),
					"workspace_id":    stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid", "member_type", "member_uuid"),
			},
			handler: s.detachProjectGroupMemberTool,
		},
		{
			tool: Tool{
				Name:        "get_project_group_topology",
				Description: "Get the topology plane for a project group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":   stringProperty("The project group UUID"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid"),
			},
			handler: s.getProjectGroupTopologyTool,
		},
		{
			tool: Tool{
				Name:        "get_project_group_shared_env",
				Description: "Get shared environment variables for a project group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":   stringProperty("The project group UUID"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid"),
			},
			handler: s.getProjectGroupSharedEnvTool,
		},
		{
			tool: Tool{
				Name:        "put_project_group_shared_env",
				Description: "Replace shared environment variables for a project group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":      stringProperty("The project group UUID"),
					"variables":       envVariablesProperty("Shared environment variables to set"),
					"inject":          booleanProperty("Whether to inject variables into members after upsert"),
					"overwrite":       booleanProperty("Whether to overwrite existing member env values on inject"),
					"redeploy":        booleanProperty("Whether to redeploy apps after inject"),
					"keep_references": booleanProperty("Whether to keep existing env references"),
					"workspace_id":    stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid", "variables"),
			},
			handler: s.putProjectGroupSharedEnvTool,
		},
		{
			tool: Tool{
				Name:        "inject_project_group_shared_env",
				Description: "Inject stored project-group shared env into member services",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":      stringProperty("The project group UUID"),
					"overwrite":       booleanProperty("Whether to overwrite existing member env values"),
					"redeploy":        booleanProperty("Whether to redeploy apps after inject"),
					"member_uuids":    stringArrayProperty("Optional subset of member UUIDs to inject into"),
					"keep_references": booleanProperty("Whether to keep existing env references"),
					"workspace_id":    stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid"),
			},
			handler: s.injectProjectGroupSharedEnvTool,
		},
		{
			tool: Tool{
				Name:        "connect_project_group_services",
				Description: "Wire provider connection environment variables into a consumer project in a group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":    stringProperty("The project group UUID"),
					"consumer_type": stringProperty("Consumer type (typically project)"),
					"consumer_uuid": stringProperty("Consumer project UUID"),
					"provider_type": stringProperty("Provider type (typically addon_deployment)"),
					"provider_uuid": stringProperty("Provider add-on deployment UUID"),
					"overwrite":     booleanProperty("Whether to overwrite existing consumer env values"),
					"variable_set":  stringProperty("Optional variable set name"),
					"workspace_id":  stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid", "consumer_type", "consumer_uuid", "provider_type", "provider_uuid"),
			},
			handler: s.connectProjectGroupServicesTool,
		},
		{
			tool: Tool{
				Name:        "redeploy_project_group_apps",
				Description: "Queue redeploys for application (project) members in a project group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":   stringProperty("The project group UUID"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "group_uuid"),
			},
			handler: s.redeployProjectGroupAppsTool,
		},
		{
			tool: Tool{
				Name:        "resolve_project_group_member",
				Description: "Resolve which project group a service member belongs to",
				InputSchema: objectSchema(map[string]interface{}{
					"member_type":  stringProperty("Member type: project or addon_deployment"),
					"member_uuid":  stringProperty("The member project or add-on deployment UUID"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}, "member_type", "member_uuid"),
			},
			handler: s.resolveProjectGroupMemberTool,
		},
		{
			tool: Tool{
				Name:        "list_project_group_candidates",
				Description: "List projects and add-ons that can be attached to a project group",
				InputSchema: objectSchema(map[string]interface{}{
					"group_uuid":   stringProperty("Optional target project group UUID for in-group flags"),
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}),
			},
			handler: s.listProjectGroupCandidatesTool,
		},
		{
			tool: Tool{
				Name:        "list_service_account_tokens",
				Description: "List service account tokens",
				InputSchema: objectSchema(map[string]interface{}{
					"workspace_id": stringProperty("Optional workspace ID or UUID override"),
				}),
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
		if !s.toolAllowed(definition.tool.Name) {
			continue
		}
		definition.tool.Annotations = annotationsForTool(definition.tool.Name)
		tools = append(tools, definition.tool)
	}

	return map[string]interface{}{"tools": tools}
}

func annotationsForTool(name string) *ToolAnnotations {
	readOnly := hasAnyPrefix(name, "get_", "list_", "search_", "check_", "view_")
	additive := hasAnyPrefix(name, "create_", "add_", "invite_", "link_")
	destructive := !readOnly && !additive
	closedWorld := false
	annotations := &ToolAnnotations{
		ReadOnlyHint:   readOnly,
		IdempotentHint: readOnly,
		OpenWorldHint:  &closedWorld,
	}
	if !readOnly {
		annotations.DestructiveHint = boolPointer(destructive)
	}
	return annotations
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func boolPointer(value bool) *bool {
	return &value
}

// handleToolsCall handles a tool invocation.
func normalizeLegacyToolName(name string) string {
	switch name {
	case "get_cluster_connection":
		return "get_server_connection"
	case "get_cluster_cost_allocation":
		return "get_server_cost_allocation"
	default:
		return name
	}
}

func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var callParams ToolCallParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	callParams.Name = normalizeLegacyToolName(callParams.Name)
	if !s.toolAllowed(callParams.Name) {
		return nil, fmt.Errorf("tool %s is not allowed by the approved OAuth scopes", callParams.Name)
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
		// additionalProperties must be a boolean or a valid schema. Callers that
		// need named fields should use nestedObjectProperty instead of passing a
		// properties map here (that pattern breaks draft 2020-12 for Claude).
		property["additionalProperties"] = additionalProperties
	}
	return property
}

// nestedObjectProperty builds a typed object schema with fixed properties.
// Use this for nested tool arguments; do not pass property maps to objectProperty.
func nestedObjectProperty(description string, properties map[string]interface{}, required ...string) map[string]interface{} {
	schema := objectSchema(properties, required...)
	schema["description"] = description
	return schema
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

func normalizeVCSProvider(provider string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	switch normalized {
	case "github", "gitlab", "bitbucket":
		return normalized, nil
	case "azuredevops", "azure-devops", "azure_devops":
		return "azuredevops", nil
	case "":
		return "", fmt.Errorf("provider is required")
	default:
		return "", fmt.Errorf("unsupported provider %q", provider)
	}
}

func (s *Server) resolveDefaultWorkspaceID(ctx context.Context, args map[string]interface{}) (string, error) {
	// Prefer explicit args (agents may pass either key).
	if ws := firstNonEmptyString(
		optionalStringArg(args, "workspace_id"),
		optionalStringArg(args, "workspace_uuid"),
		optionalStringArg(args, "workspaceUUID"),
	); ws != "" {
		return ws, nil
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

func (s *Server) resolveDefaultWorkspaceUUID(ctx context.Context, args map[string]interface{}) (string, error) {
	workspaceID, err := s.resolveDefaultWorkspaceID(ctx, args)
	if err != nil {
		return "", err
	}

	explicitWorkspaceID := firstNonEmptyString(
		optionalStringArg(args, "workspace_id"),
		optionalStringArg(args, "workspace_uuid"),
		optionalStringArg(args, "workspaceUUID"),
	)
	if explicitWorkspaceID == "" {
		return workspaceID, nil
	}
	if isLikelyUUID(workspaceID) {
		return workspaceID, nil
	}

	isNumericWorkspaceID := true
	for _, r := range strings.TrimSpace(workspaceID) {
		if r < '0' || r > '9' {
			isNumericWorkspaceID = false
			break
		}
	}
	if !isNumericWorkspaceID {
		return workspaceID, nil
	}

	workspace, err := s.resolveWorkspaceReference(ctx, workspaceID)
	if err == nil && isLikelyUUID(workspace.UUID) {
		return workspace.UUID, nil
	}
	if err == nil && isUsableWorkspaceIdentifier(workspace.UUID) {
		return workspace.UUID, nil
	}
	return workspaceID, nil
}

func (s *Server) resolveWorkspaceUUID(ctx context.Context, workspaceID string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return "", fmt.Errorf("workspace_id is required")
	}
	return s.resolveDefaultWorkspaceUUID(ctx, map[string]interface{}{"workspace_id": workspaceID})
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

func withQueryValues(path string, values map[string]string) string {
	parsed, err := url.Parse(path)
	if err != nil {
		return path
	}

	query := parsed.Query()
	for key, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		query.Set(key, trimmed)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func withWorkspaceUUIDQuery(path, workspaceUUID string) string {
	if strings.TrimSpace(workspaceUUID) == "" {
		return path
	}
	return withQueryValues(path, map[string]string{"workspace_uuid": workspaceUUID})
}

func buildProjectLogsQueryValues(opts *pipeops.LogsOptions, includeWorkspace bool) map[string]string {
	values := map[string]string{"app": "project"}
	if opts == nil {
		return values
	}
	if app := strings.TrimSpace(opts.App); app != "" {
		values["app"] = app
	}
	if includeWorkspace {
		workspaceUUID := strings.TrimSpace(opts.WorkspaceUUID)
		if workspaceUUID == "" {
			workspaceUUID = strings.TrimSpace(opts.WorkspaceID)
		}
		if workspaceUUID != "" {
			values["workspace_uuid"] = workspaceUUID
		}
	}
	if start := strings.TrimSpace(opts.Start); start != "" {
		values["start"] = start
	} else if start := strings.TrimSpace(opts.StartTime); start != "" {
		values["start"] = start
	}
	if end := strings.TrimSpace(opts.End); end != "" {
		values["end"] = end
	} else if end := strings.TrimSpace(opts.EndTime); end != "" {
		values["end"] = end
	}
	if opts.Limit > 0 {
		values["limit"] = fmt.Sprintf("%d", opts.Limit)
	}
	if search := strings.TrimSpace(opts.Search); search != "" {
		values["search"] = search
	}
	if logMode := strings.TrimSpace(opts.Log); logMode != "" {
		values["log"] = logMode
	}
	if opts.Delay > 0 {
		values["delay"] = fmt.Sprintf("%d", opts.Delay)
	}
	return values
}

func normalizeLogsResponse(resp map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{}, len(resp))
	for key, value := range resp {
		normalized[key] = value
	}

	switch data := normalized["data"].(type) {
	case nil:
		normalized["data"] = map[string]interface{}{"logs": []interface{}{}}
	case []interface{}:
		normalized["data"] = map[string]interface{}{"logs": data}
	case map[string]interface{}:
		if _, ok := data["logs"]; ok {
			return normalized
		}
		if len(data) == 0 {
			normalized["data"] = map[string]interface{}{"logs": []interface{}{}}
			return normalized
		}
		normalized["data"] = map[string]interface{}{"logs": []interface{}{data}}
	default:
		normalized["data"] = map[string]interface{}{"logs": []interface{}{data}}
	}
	return normalized
}

func (s *Server) requestProjectLogs(ctx context.Context, projectID string, opts *pipeops.LogsOptions, includeWorkspace bool) (map[string]interface{}, error) {
	path := withQueryValues(
		fmt.Sprintf("project/logs/%s", url.PathEscape(strings.TrimSpace(projectID))),
		buildProjectLogsQueryValues(opts, includeWorkspace),
	)

	resp, err := s.requestJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}
	return normalizeLogsResponse(resp), nil
}

func buildProjectDeploymentsQueryValues(workspaceUUID, filterBy string, page, limit int) map[string]string {
	values := map[string]string{}
	if normalizedWorkspaceUUID := strings.TrimSpace(workspaceUUID); normalizedWorkspaceUUID != "" {
		values["workspace_uuid"] = normalizedWorkspaceUUID
	}
	if normalizedFilter := strings.TrimSpace(filterBy); normalizedFilter != "" {
		values["filterBy"] = normalizedFilter
	}
	if page > 0 {
		values["page"] = fmt.Sprintf("%d", page)
	}
	if limit > 0 {
		values["limit"] = fmt.Sprintf("%d", limit)
	}
	return values
}

func (s *Server) requestProjectDeployments(ctx context.Context, projectID, workspaceUUID string, filterBy string, page, limit int) (map[string]interface{}, error) {
	path := withQueryValues(
		fmt.Sprintf("project/get-deployments/%s", url.PathEscape(strings.TrimSpace(projectID))),
		buildProjectDeploymentsQueryValues(workspaceUUID, filterBy, page, limit),
	)

	resp, err := s.requestJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list project deployments: %w", err)
	}
	return normalizeCollectionResponse(resp, "deployments"), nil
}

func (s *Server) requestProjectDeploymentHistory(ctx context.Context, projectID, workspaceUUID string, page, limit int) (map[string]interface{}, error) {
	path := withQueryValues(
		fmt.Sprintf("project/deployment/%s", url.PathEscape(strings.TrimSpace(projectID))),
		buildProjectDeploymentsQueryValues(workspaceUUID, "", page, limit),
	)

	resp, err := s.requestJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list project deployment history: %w", err)
	}
	return normalizeCollectionResponse(resp, "deployments"), nil
}

func (s *Server) requestProjectDeploymentCollectionWithFallback(ctx context.Context, projectID, workspaceID string, request func(context.Context, string, string) (map[string]interface{}, error)) (map[string]interface{}, map[string]interface{}, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}

	explicitWorkspaceUUID := ""
	if strings.TrimSpace(workspaceID) != "" {
		workspace, err := s.resolveWorkspaceReference(ctx, workspaceID)
		if err != nil {
			return nil, nil, err
		}
		explicitWorkspaceUUID = projectWorkspaceUUID(nil, workspace)
	}

	tryRequest := func(projectIdentifiers []string, workspaceUUIDs []string) (map[string]interface{}, error) {
		if len(workspaceUUIDs) == 0 {
			workspaceUUIDs = []string{""}
		}

		attempted := make(map[string]struct{})
		var lastErr error
		for _, projectIdentifier := range projectIdentifiers {
			projectIdentifier = strings.TrimSpace(projectIdentifier)
			if projectIdentifier == "" {
				continue
			}
			for _, workspaceUUID := range workspaceUUIDs {
				workspaceUUID = strings.TrimSpace(workspaceUUID)
				attemptKey := projectIdentifier + "|" + workspaceUUID
				if _, ok := attempted[attemptKey]; ok {
					continue
				}
				attempted[attemptKey] = struct{}{}

				resp, err := request(ctx, projectIdentifier, workspaceUUID)
				if err == nil {
					return resp, nil
				}
				if !isProjectDeploymentRetryableError(err) {
					return nil, err
				}
				lastErr = err
			}
		}
		return nil, lastErr
	}

	var directErr error
	if isLikelyDirectProjectIdentifier(projectID) {
		directWorkspaceUUIDs := []string{}
		if explicitWorkspaceUUID != "" {
			directWorkspaceUUIDs = appendUniqueString(directWorkspaceUUIDs, explicitWorkspaceUUID)
		}
		directWorkspaceUUIDs = append(directWorkspaceUUIDs, "")

		resp, err := tryRequest([]string{projectID}, directWorkspaceUUIDs)
		if err == nil {
			return resp, nil, nil
		}
		directErr = err
	}

	workspace, project, err := s.findProjectReference(ctx, projectID, workspaceID)
	if err != nil {
		return nil, nil, err
	}
	if project == nil {
		if directErr != nil {
			return nil, nil, directErr
		}
		return nil, nil, fmt.Errorf("project %q not found", projectID)
	}

	projectIdentifiers := projectLogIdentifiers(project)
	if len(projectIdentifiers) == 0 {
		if directErr != nil {
			return nil, nil, directErr
		}
		return nil, nil, fmt.Errorf("project %q not found", projectID)
	}

	workspaceUUIDs := []string{}
	if resolvedWorkspaceUUID := projectWorkspaceUUID(project, workspace); resolvedWorkspaceUUID != "" {
		workspaceUUIDs = appendUniqueString(workspaceUUIDs, resolvedWorkspaceUUID)
	}
	if explicitWorkspaceUUID != "" {
		workspaceUUIDs = appendUniqueString(workspaceUUIDs, explicitWorkspaceUUID)
	}
	workspaceUUIDs = append(workspaceUUIDs, "")

	resp, err := tryRequest(projectIdentifiers, workspaceUUIDs)
	if err != nil {
		return nil, nil, err
	}
	return resp, normalizeProject(project), nil
}

func responseCollectionItems(resp map[string]interface{}, key string) []map[string]interface{} {
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return nil
	}
	items, ok := data[key].([]interface{})
	if !ok {
		return nil
	}

	results := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		payload, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		results = append(results, payload)
	}
	return results
}

func setCollectionItems(resp map[string]interface{}, key string, items []map[string]interface{}) map[string]interface{} {
	normalized := normalizeCollectionResponse(resp, key)
	data, ok := normalized["data"].(map[string]interface{})
	if !ok {
		data = map[string]interface{}{}
	}
	serialized := make([]interface{}, 0, len(items))
	for _, item := range items {
		serialized = append(serialized, item)
	}
	data[key] = serialized
	normalized["data"] = data

	meta, ok := normalized["meta"].(map[string]interface{})
	if !ok {
		meta = map[string]interface{}{}
	}
	meta["current_count"] = len(items)
	normalized["meta"] = meta
	return normalized
}

func attachProjectToCollectionResponse(resp map[string]interface{}, project map[string]interface{}) map[string]interface{} {
	if project == nil {
		return resp
	}
	normalized := make(map[string]interface{}, len(resp))
	for key, value := range resp {
		normalized[key] = value
	}
	data, ok := normalized["data"].(map[string]interface{})
	if !ok {
		data = map[string]interface{}{}
	}
	data["project"] = normalizeProject(project)
	normalized["data"] = data
	return normalized
}

func searchTextForValue(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []interface{}:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(searchTextForValue(item)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	case map[string]interface{}:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(searchTextForValue(item)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func deploymentMatchesSearch(item map[string]interface{}, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(searchTextForValue(item)), query)
}

func filterDeploymentItems(items []map[string]interface{}, query string) []map[string]interface{} {
	if strings.TrimSpace(query) == "" {
		return items
	}
	filtered := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if deploymentMatchesSearch(item, query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func withClusterCostAllocationQuery(path, workspaceUUID, aggregate, window string) string {
	aggregate = strings.TrimSpace(aggregate)
	if aggregate == "" {
		aggregate = "namespace"
	}
	window = strings.TrimSpace(window)
	if window == "" {
		window = "30d"
	}
	return withQueryValues(path, map[string]string{
		"workspace_uuid": workspaceUUID,
		"aggregate":      aggregate,
		"window":         window,
	})
}

func withNovaClusterCostAllocationQuery(path, workspaceUUID, aggregate, window, clusterID, location string) string {
	aggregate = strings.TrimSpace(aggregate)
	if aggregate == "" {
		aggregate = "namespace"
	}
	window = strings.TrimSpace(window)
	if window == "" {
		window = "30d"
	}
	location = normalizeCostLocation(location)
	return withQueryValues(path, map[string]string{
		"workspace_uuid": workspaceUUID,
		"aggregate":      aggregate,
		"window":         window,
		"cluster":        clusterID,
		"location":       location,
	})
}

func (s *Server) requestBillingJSONWithWorkspaceFallback(ctx context.Context, method, path string, args map[string]interface{}, body interface{}, workspaceUUID *string) (map[string]interface{}, error) {
	if workspaceUUID != nil && *workspaceUUID != "" {
		return s.requestJSON(ctx, method, withWorkspaceUUIDQuery(path, *workspaceUUID), body)
	}

	if workspaceID, ok := args["workspace_id"].(string); ok && strings.TrimSpace(workspaceID) != "" {
		resolvedWorkspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
		if err != nil {
			return nil, err
		}
		if workspaceUUID != nil {
			*workspaceUUID = resolvedWorkspaceUUID
		}
		return s.requestJSON(ctx, method, withWorkspaceUUIDQuery(path, resolvedWorkspaceUUID), body)
	}

	resp, err := s.requestJSON(ctx, method, path, body)
	if err == nil || !isBillingWorkspaceRequiredError(err) {
		return resp, err
	}

	resolvedWorkspaceUUID, resolveErr := s.resolveDefaultWorkspaceUUID(ctx, args)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if workspaceUUID != nil {
		*workspaceUUID = resolvedWorkspaceUUID
	}
	return s.requestJSON(ctx, method, withWorkspaceUUIDQuery(path, resolvedWorkspaceUUID), body)
}

func normalizeCollectionResponse(resp map[string]interface{}, key string) map[string]interface{} {
	normalized := make(map[string]interface{}, len(resp))
	for k, v := range resp {
		normalized[k] = v
	}

	switch data := normalized["data"].(type) {
	case nil:
		normalized["data"] = map[string]interface{}{key: []interface{}{}}
	case []interface{}:
		normalized["data"] = map[string]interface{}{key: data}
	case map[string]interface{}:
		if _, ok := data[key]; !ok {
			normalized["data"] = map[string]interface{}{key: []interface{}{data}}
		}
	}

	return normalized
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

type getProjectArgs struct {
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
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
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	StartTime   string `json:"start_time,omitempty"`
	EndTime     string `json:"end_time,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Search      string `json:"search,omitempty"`
}

type projectDeploymentsArgs struct {
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	FilterBy    string `json:"filter_by,omitempty"`
	Page        int    `json:"page,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type searchProjectDeploymentsArgs struct {
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	FilterBy    string `json:"filter_by,omitempty"`
	Search      string `json:"search"`
	Page        int    `json:"page,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type projectEnvVariablesArgs struct {
	ProjectID    string                `json:"project_id"`
	EnvVariables []pipeops.EnvVariable `json:"env_variables"`
	Merge        *bool                 `json:"merge,omitempty"`
	WorkspaceID  string                `json:"workspace_id,omitempty"`
}

type projectDeploySettingsArgs struct {
	ProjectID         string `json:"project_id"`
	WorkspaceID       string `json:"workspace_id,omitempty"`
	Branch            string `json:"branch,omitempty"`
	Repository        string `json:"repository,omitempty"`
	Username          string `json:"username,omitempty"`
	AutoDeployEnabled *bool  `json:"auto_deploy_enabled,omitempty"`
	AutoRollback      *bool  `json:"auto_rollback,omitempty"`
}

type projectSecurityPolicyArgs struct {
	ProjectID     string   `json:"project_id"`
	WorkspaceID   string   `json:"workspace_id,omitempty"`
	Enabled       *bool    `json:"enabled,omitempty"`
	MaxCritical   *int     `json:"max_critical,omitempty"`
	MaxHigh       *int     `json:"max_high,omitempty"`
	MaxMedium     *int     `json:"max_medium,omitempty"`
	MaxCvssScore  *float64 `json:"max_cvss_score,omitempty"`
	MaxTotalVulns *int     `json:"max_total_vulns,omitempty"`
	FailOnSecrets *bool    `json:"fail_on_secrets,omitempty"`
}

type listEnvironmentsArgs struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
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
	AddOnID       string                 `json:"addon_id"`
	ProjectID     string                 `json:"project_id,omitempty"`
	ServerID      string                 `json:"server_id,omitempty"`
	WorkspaceID   string                 `json:"workspace_id,omitempty"`
	EnvironmentID string                 `json:"environment_id,omitempty"`
	Tag           string                 `json:"tag,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
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

type listAddOnsArgs struct {
	Page        int    `json:"page,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Category    string `json:"category,omitempty"`
	Search      string `json:"search,omitempty"`
	Featured    *bool  `json:"featured,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type listVolumesArgs struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	Status      string `json:"status,omitempty"`
	ClusterUUID string `json:"cluster_uuid,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

type volumeUUIDArgs struct {
	VolumeUUID  string `json:"volume_uuid"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type remountVolumeArgs struct {
	VolumeUUID  string `json:"volume_uuid"`
	TargetType  string `json:"target_type"`
	TargetUUID  string `json:"target_uuid"`
	MountPath   string `json:"mount_path,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type listAddonBackupsArgs struct {
	DeploymentID string `json:"deployment_id"`
}

type startAddonBackupExportArgs struct {
	DeploymentID string `json:"deployment_id"`
	SnapshotID   string `json:"snapshot_id"`
	Path         string `json:"path,omitempty"`
	Format       string `json:"format,omitempty"`
}

type getAddonBackupExportArgs struct {
	DeploymentID string `json:"deployment_id"`
	ExportID     string `json:"export_id"`
}

type vcsProviderArgs struct {
	Provider string `json:"provider"`
}

type vcsRepositoriesArgs struct {
	Provider string `json:"provider"`
	OrgName  string `json:"org_name"`
	Page     int    `json:"page,omitempty"`
}

type vcsRepoSearchArgs struct {
	Provider       string `json:"provider"`
	OrgName        string `json:"org_name"`
	RepositoryName string `json:"repository_name"`
	Page           int    `json:"page,omitempty"`
}

type vcsBranchesArgs struct {
	Provider     string `json:"provider"`
	RepoFullname string `json:"repo_fullname"`
	Visibility   string `json:"visibility,omitempty"`
	Search       string `json:"search,omitempty"`
}

type vcsDockerfileArgs struct {
	Provider   string `json:"provider"`
	Owner      string `json:"owner"`
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
}

type linkVCSProviderArgs struct {
	Provider     string `json:"provider"`
	RedirectPath string `json:"redirect_path"`
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

type cloudProviderInstanceTypesArgs struct {
	CloudProvider string `json:"cloud_provider"`
	InstanceClass string `json:"instance_class"`
	Region        string `json:"region"`
}

type workspaceReference struct {
	ID       string
	UUID     string
	Projects []map[string]interface{}
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

	var (
		resp map[string]interface{}
		err  error
	)
	if opts != nil && (opts.WorkspaceID != "" || opts.WorkspaceUUID != "") {
		workspaceID := opts.WorkspaceUUID
		if workspaceID == "" {
			workspaceID = opts.WorkspaceID
		}
		resp, err = s.listProjectsForWorkspace(ctx, workspaceID, opts)
	} else {
		resp, err = s.listProjectsAcrossWorkspaces(ctx, opts)
	}
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listProjectsAcrossWorkspaces(ctx context.Context, opts *pipeops.ProjectListOptions) (map[string]interface{}, error) {
	workspaces, status, message, err := s.listWorkspaceReferences(ctx)
	if err != nil {
		return nil, err
	}

	projects := make([]map[string]interface{}, 0)
	seen := make(map[string]struct{})
	for _, workspace := range workspaces {
		workspaceProjects, workspaceStatus, workspaceMessage, workspaceErr := s.fetchProjectsForWorkspaceReference(ctx, workspace)
		if workspaceErr != nil {
			if isWorkspaceProjectsFallbackError(workspaceErr) {
				continue
			}
			return nil, workspaceErr
		}
		if status == "" {
			status = workspaceStatus
		}
		if message == "" {
			message = workspaceMessage
		}
		for _, project := range workspaceProjects {
			if opts != nil && opts.ServerID != "" && !projectMatchesServerID(project, opts.ServerID) {
				continue
			}
			projectKey := projectIdentity(project)
			if projectKey == "" {
				continue
			}
			if _, ok := seen[projectKey]; ok {
				continue
			}
			seen[projectKey] = struct{}{}
			projects = append(projects, normalizeProject(project))
		}
	}

	if opts != nil {
		projects = paginateProjects(projects, opts.Page, opts.Limit)
	}

	return map[string]interface{}{
		"status":  responseStatus(status),
		"message": message,
		"data": map[string]interface{}{
			"projects": projects,
		},
	}, nil
}

func (s *Server) listProjectsForWorkspace(ctx context.Context, workspaceID string, opts *pipeops.ProjectListOptions) (map[string]interface{}, error) {
	workspaceRef, err := s.resolveWorkspaceReference(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	projects, status, message, err := s.fetchProjectsForWorkspaceReference(ctx, workspaceRef)
	if err != nil {
		if isWorkspaceProjectsFallbackError(err) {
			return map[string]interface{}{
				"status":  "success",
				"message": err.Error(),
				"data": map[string]interface{}{
					"projects": []map[string]interface{}{},
				},
			}, nil
		}
		return nil, err
	}
	filtered := make([]map[string]interface{}, 0, len(projects))
	for _, project := range projects {
		if opts != nil && opts.ServerID != "" && !projectMatchesServerID(project, opts.ServerID) {
			continue
		}
		filtered = append(filtered, normalizeProject(project))
	}
	if opts != nil {
		filtered = paginateProjects(filtered, opts.Page, opts.Limit)
	}

	return map[string]interface{}{
		"status":  responseStatus(status),
		"message": message,
		"data": map[string]interface{}{
			"projects": filtered,
		},
	}, nil
}

func (s *Server) resolveWorkspaceReference(ctx context.Context, workspaceID string) (workspaceReference, error) {
	ref := workspaceReference{UUID: workspaceID, ID: workspaceID}
	workspaces, _, _, err := s.listWorkspaceReferences(ctx)
	if err != nil {
		return ref, nil
	}
	for _, workspace := range workspaces {
		if workspace.UUID == workspaceID || workspace.ID == workspaceID {
			return workspace, nil
		}
	}
	return ref, nil
}

func (s *Server) fetchProjectsForWorkspaceReference(ctx context.Context, workspace workspaceReference) ([]map[string]interface{}, string, string, error) {
	if len(workspace.Projects) > 0 {
		return workspace.Projects, "success", "ok", nil
	}

	identifiers := workspaceReferenceIdentifiers(workspace)
	var lastErr error
	for _, identifier := range identifiers {
		projects, status, message, err := s.fetchWorkspaceProjects(ctx, identifier)
		if err == nil {
			return projects, status, message, nil
		}
		if !isWorkspaceProjectsFallbackError(err) {
			return nil, "", "", err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", "", lastErr
	}
	return []map[string]interface{}{}, "success", "", nil
}

func (s *Server) listWorkspaceReferences(ctx context.Context) ([]workspaceReference, string, string, error) {
	req, err := s.client.NewRequest(http.MethodGet, "workspace", nil)
	if err != nil {
		return nil, "", "", err
	}

	var envelope struct {
		Success bool            `json:"success,omitempty"`
		Status  string          `json:"status,omitempty"`
		Message string          `json:"message,omitempty"`
		Data    json.RawMessage `json:"data,omitempty"`
	}
	if _, err := s.client.Do(ctx, req, &envelope); err != nil {
		return nil, "", "", err
	}

	workspaces, err := parseWorkspaceReferences(envelope.Data)
	if err != nil {
		return nil, "", "", err
	}
	return workspaces, envelopeStatus(envelope.Status, envelope.Success), envelope.Message, nil
}

func parseWorkspaceReferences(data json.RawMessage) ([]workspaceReference, error) {
	items, err := asMapSlice(data)
	if err != nil {
		var wrapped map[string]interface{}
		if err := json.Unmarshal(data, &wrapped); err != nil {
			return nil, err
		}
		items = mapSliceValue(wrapped, "workspaces", "Workspaces")
	}

	workspaces := make([]workspaceReference, 0, len(items))
	for _, item := range items {
		workspace := workspaceReference{
			ID:       extractString(lookupValue(item, "id", "ID")),
			UUID:     extractString(lookupValue(item, "uuid", "UUID", "uid", "UID")),
			Projects: mapSliceValue(item, "projects", "Projects"),
		}
		if workspace.ID == "" && workspace.UUID == "" {
			continue
		}
		workspaces = append(workspaces, workspace)
	}
	return workspaces, nil
}

func (s *Server) fetchWorkspaceProjects(ctx context.Context, workspaceID string) ([]map[string]interface{}, string, string, error) {
	req, err := s.client.NewRequest(http.MethodGet, fmt.Sprintf("workspace/fetch/%s", workspaceID), nil)
	if err != nil {
		return nil, "", "", err
	}

	var envelope struct {
		Success bool            `json:"success,omitempty"`
		Status  string          `json:"status,omitempty"`
		Message string          `json:"message,omitempty"`
		Data    json.RawMessage `json:"data,omitempty"`
	}
	if _, err := s.client.Do(ctx, req, &envelope); err != nil {
		return nil, "", "", err
	}

	projects, err := parseWorkspaceProjects(envelope.Data)
	if err != nil {
		return nil, "", "", err
	}
	return projects, envelopeStatus(envelope.Status, envelope.Success), envelope.Message, nil
}

func parseWorkspaceProjects(data json.RawMessage) ([]map[string]interface{}, error) {
	workspace, err := extractWorkspacePayload(data)
	if err != nil {
		return nil, err
	}

	projects := mapSliceValue(workspace, "projects", "Projects")
	return projects, nil
}

func parseProjectNameReferences(data json.RawMessage) ([]map[string]interface{}, error) {
	if len(data) == 0 || strings.TrimSpace(string(data)) == "" || string(data) == "null" {
		return []map[string]interface{}{}, nil
	}

	projects, err := asMapSlice(data)
	if err == nil {
		return projects, nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	projects = mapSliceValue(payload, "projects", "Projects")
	if len(projects) > 0 {
		return projects, nil
	}

	nested, ok := lookupValue(payload, "projects", "Projects").(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}
	return mapSliceValue(nested, "rows", "Rows"), nil
}

func (s *Server) fetchProjectNameReferences(ctx context.Context) ([]map[string]interface{}, string, string, error) {
	req, err := s.client.NewRequest(http.MethodGet, "project/fetch-names", nil)
	if err != nil {
		return nil, "", "", err
	}

	var envelope struct {
		Success bool            `json:"success,omitempty"`
		Status  string          `json:"status,omitempty"`
		Message string          `json:"message,omitempty"`
		Data    json.RawMessage `json:"data,omitempty"`
	}
	if _, err := s.client.Do(ctx, req, &envelope); err != nil {
		return nil, "", "", err
	}

	projects, err := parseProjectNameReferences(envelope.Data)
	if err != nil {
		return nil, "", "", err
	}

	normalized := make([]map[string]interface{}, 0, len(projects))
	for _, project := range projects {
		normalized = append(normalized, normalizeProject(project))
	}

	return normalized, envelopeStatus(envelope.Status, envelope.Success), envelope.Message, nil
}

func (s *Server) fetchEnvironmentsForWorkspaceReference(ctx context.Context, workspace workspaceReference) ([]map[string]interface{}, string, string, error) {
	identifiers := workspaceReferenceIdentifiers(workspace)
	var lastErr error
	for _, identifier := range identifiers {
		environments, status, message, err := s.fetchWorkspaceEnvironments(ctx, identifier)
		if err == nil {
			return environments, status, message, nil
		}
		if !isWorkspaceProjectsFallbackError(err) {
			return nil, "", "", err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", "", lastErr
	}
	return []map[string]interface{}{}, "success", "", nil
}

func (s *Server) fetchWorkspaceEnvironments(ctx context.Context, workspaceID string) ([]map[string]interface{}, string, string, error) {
	req, err := s.client.NewRequest(http.MethodGet, fmt.Sprintf("workspace/fetch/%s", workspaceID), nil)
	if err != nil {
		return nil, "", "", err
	}

	var envelope struct {
		Success bool            `json:"success,omitempty"`
		Status  string          `json:"status,omitempty"`
		Message string          `json:"message,omitempty"`
		Data    json.RawMessage `json:"data,omitempty"`
	}
	if _, err := s.client.Do(ctx, req, &envelope); err != nil {
		return nil, "", "", err
	}

	environments, err := parseWorkspaceEnvironments(envelope.Data)
	if err != nil {
		return nil, "", "", err
	}
	return environments, envelopeStatus(envelope.Status, envelope.Success), envelope.Message, nil
}

func parseWorkspaceEnvironments(data json.RawMessage) ([]map[string]interface{}, error) {
	workspace, err := extractWorkspacePayload(data)
	if err != nil {
		return nil, err
	}

	environments := make([]map[string]interface{}, 0)
	seen := make(map[string]struct{})
	appendEnvironment := func(environment map[string]interface{}, cluster map[string]interface{}) {
		normalized := normalizeEnvironment(environment, workspace, cluster)
		identity := environmentIdentity(normalized)
		if identity != "" {
			if _, ok := seen[identity]; ok {
				return
			}
			seen[identity] = struct{}{}
		}
		environments = append(environments, normalized)
	}

	for _, environment := range mapSliceValue(workspace, "environments", "Environments") {
		appendEnvironment(environment, nil)
	}

	for _, entry := range mapSliceValue(workspace, "clusters", "Clusters") {
		cluster := entry
		if nested, ok := lookupValue(entry, "Cluster", "cluster").(map[string]interface{}); ok {
			cluster = nested
		}
		for _, environment := range mapSliceValue(entry, "environments", "Environments") {
			appendEnvironment(environment, cluster)
		}
		for _, environment := range mapSliceValue(cluster, "environments", "Environments") {
			appendEnvironment(environment, cluster)
		}
	}

	return environments, nil
}

func extractWorkspacePayload(data json.RawMessage) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	workspaceValue := lookupValue(payload, "workspace", "Workspace")
	workspace, ok := workspaceValue.(map[string]interface{})
	if !ok {
		workspace = payload
	}
	return workspace, nil
}

func normalizeEnvironment(environment map[string]interface{}, workspace map[string]interface{}, cluster map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{}, len(environment)+5)
	for key, value := range environment {
		normalized[key] = value
	}
	if _, ok := normalized["UUID"]; !ok {
		if uuid := extractString(lookupValue(environment, "UUID", "uuid", "UID", "uid")); uuid != "" {
			normalized["UUID"] = uuid
		}
	}
	if _, ok := normalized["ID"]; !ok {
		if id := extractString(lookupValue(environment, "ID", "id")); id != "" {
			normalized["ID"] = id
		}
	}
	if _, ok := normalized["Name"]; !ok {
		if name := extractString(lookupValue(environment, "Name", "name")); name != "" {
			normalized["Name"] = name
		}
	}
	if _, ok := normalized["Namespace"]; !ok {
		if namespace := extractString(lookupValue(environment, "Namespace", "namespace")); namespace != "" {
			normalized["Namespace"] = namespace
		}
	}
	if _, ok := normalized["EnvironmentEnvs"]; !ok {
		if envs := lookupValue(environment, "EnvironmentEnvs", "environment_envs", "env_variables", "EnvVariables"); envs != nil {
			normalized["EnvironmentEnvs"] = envs
		}
	}
	if _, ok := normalized["ClusterUUID"]; !ok {
		if clusterUUID := extractString(lookupValue(environment, "ClusterUUID", "cluster_uuid")); clusterUUID != "" {
			normalized["ClusterUUID"] = clusterUUID
		} else if clusterUUID := extractString(lookupValue(cluster, "UUID", "uuid", "ClusterUUID", "cluster_uuid")); clusterUUID != "" {
			normalized["ClusterUUID"] = clusterUUID
		}
	}
	if _, ok := normalized["WorkspaceID"]; !ok {
		if workspaceID := extractString(lookupValue(environment, "WorkspaceID", "workspace_id")); workspaceID != "" {
			normalized["WorkspaceID"] = workspaceID
		} else if workspaceID := extractString(lookupValue(workspace, "UUID", "uuid", "ID", "id")); workspaceID != "" {
			normalized["WorkspaceID"] = workspaceID
		}
	}
	return normalized
}

func environmentIdentity(environment map[string]interface{}) string {
	for _, key := range []string{"UUID", "uuid", "UID", "uid", "ID", "id"} {
		if value := extractString(lookupValue(environment, key)); value != "" {
			return value
		}
	}
	clusterUUID := extractString(lookupValue(environment, "ClusterUUID", "cluster_uuid"))
	namespace := extractString(lookupValue(environment, "Namespace", "namespace"))
	name := extractString(lookupValue(environment, "Name", "name"))
	if clusterUUID != "" || namespace != "" || name != "" {
		return strings.Join([]string{clusterUUID, namespace, name}, "|")
	}
	return ""
}

func workspaceReferenceIdentifiers(workspace workspaceReference) []string {
	identifiers := make([]string, 0, 2)
	appendIdentifier := func(value string) {
		if value == "" {
			return
		}
		for _, existing := range identifiers {
			if existing == value {
				return
			}
		}
		identifiers = append(identifiers, value)
	}
	appendIdentifier(workspace.UUID)
	if isUsableWorkspaceIdentifier(workspace.ID) {
		appendIdentifier(workspace.ID)
	}
	return identifiers
}

func normalizeProject(project map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{}, len(project)+4)
	for key, value := range project {
		normalized[key] = value
	}
	if _, ok := normalized["UUID"]; !ok {
		if uuid := extractString(lookupValue(project, "UUID", "uuid")); uuid != "" {
			normalized["UUID"] = uuid
		}
	}
	if _, ok := normalized["ID"]; !ok {
		if id := extractString(lookupValue(project, "ID", "id")); id != "" {
			normalized["ID"] = id
		}
	}
	if _, ok := normalized["Name"]; !ok {
		if name := extractString(lookupValue(project, "Name", "name")); name != "" {
			normalized["Name"] = name
		}
	}
	if _, ok := normalized["NameSlug"]; !ok {
		if slug := extractString(lookupValue(project, "NameSlug", "name_slug", "Slug", "slug", "ProjectSlug", "project_slug")); slug != "" {
			normalized["NameSlug"] = slug
		} else if slug := normalizeIdentifierSlug(extractString(lookupValue(normalized, "Name", "name"))); slug != "" {
			normalized["NameSlug"] = slug
		}
	}
	return normalized
}

func projectMatchesServerID(project map[string]interface{}, serverID string) bool {
	if serverID == "" {
		return true
	}
	for _, key := range []string{"server_id", "ServerID", "cluster_uuid", "ClusterUUID"} {
		if extractString(lookupValue(project, key)) == serverID {
			return true
		}
	}
	return false
}

func projectIdentity(project map[string]interface{}) string {
	for _, key := range []string{"UUID", "uuid", "ProjectUUID", "project_uuid", "ID", "id"} {
		if value := extractString(lookupValue(project, key)); value != "" {
			return value
		}
	}
	return ""
}

func normalizeIdentifierSlug(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	lastHyphen := false
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func projectMatchesIdentifier(project map[string]interface{}, identifier string) bool {
	for _, key := range []string{"UUID", "uuid", "ProjectUUID", "project_uuid", "ID", "id", "Name", "name", "NameSlug", "name_slug", "Slug", "slug", "ProjectSlug", "project_slug"} {
		if extractString(lookupValue(project, key)) == identifier {
			return true
		}
	}
	for _, key := range []string{"Name", "name", "NameSlug", "name_slug", "Slug", "slug"} {
		if strings.EqualFold(extractString(lookupValue(project, key)), identifier) {
			return true
		}
	}

	normalizedIdentifier := normalizeIdentifierSlug(identifier)
	if normalizedIdentifier == "" {
		return false
	}
	for _, key := range []string{"Name", "name", "NameSlug", "name_slug", "Slug", "slug", "ProjectSlug", "project_slug"} {
		if normalizeIdentifierSlug(extractString(lookupValue(project, key))) == normalizedIdentifier {
			return true
		}
	}
	return false
}

func projectWorkspaceUUID(project map[string]interface{}, workspace workspaceReference) string {
	for _, candidate := range []string{
		workspace.UUID,
		workspace.ID,
		extractString(lookupValue(project, "WorkspaceUUID", "workspace_uuid")),
		extractString(lookupValue(project, "WorkspaceID", "workspace_id")),
	} {
		if isUsableWorkspaceIdentifier(candidate) {
			return candidate
		}
	}
	return ""
}

func (s *Server) resolveProjectReferenceWorkspace(ctx context.Context, project map[string]interface{}, workspaces []workspaceReference) (workspaceReference, error) {
	projectID := projectIdentity(project)
	if projectID == "" {
		return workspaceReference{}, nil
	}

	for _, workspace := range workspaces {
		workspaceUUID := projectWorkspaceUUID(project, workspace)
		if workspaceUUID == "" {
			continue
		}

		_, _, err := s.client.Projects.Get(ctx, projectID, &pipeops.ProjectGetOptions{WorkspaceUUID: workspaceUUID})
		if err == nil {
			return workspace, nil
		}
		if isWorkspaceProjectsFallbackError(err) {
			continue
		}
		return workspaceReference{}, err
	}

	return workspaceReference{}, nil
}

func isLikelyDirectProjectIdentifier(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if isLikelyUUID(trimmed) {
		return true
	}
	for _, r := range trimmed {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *Server) findProjectReference(ctx context.Context, projectID, workspaceID string) (workspaceReference, map[string]interface{}, error) {
	return s.findProjectReferenceWithFallback(ctx, projectID, workspaceID, false)
}

func (s *Server) findProjectReferenceWithFallback(ctx context.Context, projectID, workspaceID string, allowUnresolved bool) (workspaceReference, map[string]interface{}, error) {
	var (
		workspaces []workspaceReference
		err        error
	)
	if workspaceID != "" {
		workspace, resolveErr := s.resolveWorkspaceReference(ctx, workspaceID)
		if resolveErr != nil {
			return workspaceReference{}, nil, resolveErr
		}
		workspaces = []workspaceReference{workspace}
	} else {
		workspaces, _, _, err = s.listWorkspaceReferences(ctx)
		if err != nil {
			return workspaceReference{}, nil, err
		}
	}

	for _, workspace := range workspaces {
		projects, _, _, fetchErr := s.fetchProjectsForWorkspaceReference(ctx, workspace)
		if fetchErr != nil {
			if isWorkspaceProjectsFallbackError(fetchErr) {
				continue
			}
			return workspaceReference{}, nil, fetchErr
		}
		for _, project := range projects {
			if projectMatchesIdentifier(project, projectID) {
				return workspace, normalizeProject(project), nil
			}
		}
	}

	projectRefs, _, _, err := s.fetchProjectNameReferences(ctx)
	if err != nil {
		if isWorkspaceProjectsFallbackError(err) {
			return workspaceReference{}, nil, nil
		}
		return workspaceReference{}, nil, err
	}

	for _, project := range projectRefs {
		if !projectMatchesIdentifier(project, projectID) {
			continue
		}

		normalizedProject := normalizeProject(project)
		workspace, resolveErr := s.resolveProjectReferenceWorkspace(ctx, normalizedProject, workspaces)
		if resolveErr != nil {
			return workspaceReference{}, nil, resolveErr
		}
		if workspace.UUID == "" && workspace.ID == "" {
			if allowUnresolved && strings.TrimSpace(workspaceID) == "" {
				return workspaceReference{}, normalizedProject, nil
			}
			continue
		}
		return workspace, normalizedProject, nil
	}

	return workspaceReference{}, nil, nil
}

func normalizeCluster(cluster map[string]interface{}, workspace map[string]interface{}) map[string]interface{} {
	base := cluster
	if nested, ok := lookupValue(cluster, "Cluster", "cluster").(map[string]interface{}); ok {
		base = nested
	}

	normalized := make(map[string]interface{}, len(cluster)+len(base)+6)
	for key, value := range cluster {
		normalized[key] = value
	}
	for key, value := range base {
		normalized[key] = value
	}
	if _, ok := normalized["UUID"]; !ok {
		if uuid := extractString(lookupValue(base, "UUID", "uuid", "ClusterUUID", "cluster_uuid")); uuid != "" {
			normalized["UUID"] = uuid
		}
	}
	if _, ok := normalized["ID"]; !ok {
		if id := extractString(lookupValue(base, "ID", "id")); id != "" {
			normalized["ID"] = id
		}
	}
	if _, ok := normalized["Name"]; !ok {
		if name := extractString(lookupValue(base, "Name", "name")); name != "" {
			normalized["Name"] = name
		}
	}
	if _, ok := normalized["NameSlug"]; !ok {
		if nameSlug := extractString(lookupValue(base, "NameSlug", "name_slug", "Slug", "slug")); nameSlug != "" {
			normalized["NameSlug"] = nameSlug
		}
	}
	if _, ok := normalized["WorkspaceUUID"]; !ok {
		if workspaceUUID := extractString(lookupValue(base, "WorkspaceUUID", "workspace_uuid")); workspaceUUID != "" {
			normalized["WorkspaceUUID"] = workspaceUUID
		} else if workspaceUUID := extractString(lookupValue(workspace, "UUID", "uuid")); workspaceUUID != "" {
			normalized["WorkspaceUUID"] = workspaceUUID
		}
	}
	if _, ok := normalized["Location"]; !ok {
		if location := extractLocationValue(base); location != "" {
			normalized["Location"] = location
		} else if location := extractLocationValue(workspace); location != "" {
			normalized["Location"] = location
		}
	}
	if _, ok := normalized["CountryCode"]; !ok {
		if countryCode := extractString(lookupValue(base, "CountryCode", "country_code")); countryCode != "" {
			normalized["CountryCode"] = countryCode
		} else if countryCode := extractString(lookupValue(workspace, "CountryCode", "country_code")); countryCode != "" {
			normalized["CountryCode"] = countryCode
		}
	}
	return normalized
}

func normalizeServer(cluster map[string]interface{}, workspace workspaceReference) map[string]interface{} {
	workspacePayload := map[string]interface{}{}
	if workspace.UUID != "" {
		workspacePayload["UUID"] = workspace.UUID
	}
	if workspace.ID != "" {
		workspacePayload["ID"] = workspace.ID
	}

	normalizedCluster := normalizeCluster(cluster, workspacePayload)
	status := extractString(lookupValue(normalizedCluster, "Status", "status"))
	if status == "" {
		switch strings.ToLower(extractString(lookupValue(normalizedCluster, "IsActive", "is_active"))) {
		case "true":
			status = "active"
		case "false":
			status = "inactive"
		}
	}

	server := map[string]interface{}{}
	if id := extractString(lookupValue(normalizedCluster, "ID", "id")); id != "" {
		server["id"] = id
	}
	if uuid := clusterIdentity(normalizedCluster); uuid != "" {
		server["uuid"] = uuid
	}
	if name := extractString(lookupValue(normalizedCluster, "Name", "name")); name != "" {
		server["name"] = name
	}
	if provider := extractString(lookupValue(normalizedCluster, "CloudProvider", "cloudProvider", "cloud_provider", "Provider", "provider")); provider != "" {
		server["provider"] = provider
	}
	if region := extractString(lookupValue(normalizedCluster, "Region", "region")); region != "" {
		server["region"] = region
	}
	if status != "" {
		server["status"] = status
	}
	if workspaceUUID := clusterWorkspaceUUID(normalizedCluster, workspace); workspaceUUID != "" {
		server["workspace_id"] = workspaceUUID
	}
	if createdAt := lookupValue(normalizedCluster, "CreatedAt", "created_at"); createdAt != nil {
		server["created_at"] = createdAt
	}
	if updatedAt := lookupValue(normalizedCluster, "UpdatedAt", "updated_at"); updatedAt != nil {
		server["updated_at"] = updatedAt
	}
	return server
}

func clusterIdentity(cluster map[string]interface{}) string {
	for _, key := range []string{"UUID", "uuid", "ClusterUUID", "cluster_uuid", "ID", "id"} {
		if value := extractString(lookupValue(cluster, key)); value != "" {
			return value
		}
	}
	return ""
}

func clusterMatchesIdentifier(cluster map[string]interface{}, identifier string) bool {
	for _, key := range []string{"UUID", "uuid", "ClusterUUID", "cluster_uuid", "ID", "id", "Name", "name", "NameSlug", "name_slug", "Slug", "slug", "ServerCode", "server_code"} {
		if extractString(lookupValue(cluster, key)) == identifier {
			return true
		}
	}
	for _, key := range []string{"Name", "name", "NameSlug", "name_slug", "Slug", "slug"} {
		if strings.EqualFold(extractString(lookupValue(cluster, key)), identifier) {
			return true
		}
	}

	normalizedIdentifier := normalizeIdentifierSlug(identifier)
	if normalizedIdentifier == "" {
		return false
	}
	for _, key := range []string{"Name", "name", "NameSlug", "name_slug", "Slug", "slug", "ServerCode", "server_code"} {
		if normalizeIdentifierSlug(extractString(lookupValue(cluster, key))) == normalizedIdentifier {
			return true
		}
	}
	return false
}

func clusterWorkspaceUUID(cluster map[string]interface{}, workspace workspaceReference) string {
	candidates := []string{
		extractString(lookupValue(cluster, "WorkspaceUUID", "workspace_uuid")),
		workspace.UUID,
		workspace.ID,
		extractString(lookupValue(cluster, "WorkspaceID", "workspace_id")),
	}
	for _, candidate := range candidates {
		if isLikelyUUID(candidate) {
			return candidate
		}
	}
	for _, candidate := range candidates {
		if isUsableWorkspaceIdentifier(candidate) {
			return candidate
		}
	}
	return ""
}

func normalizeCostLocation(value string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	switch trimmed {
	case "":
		return ""
	case "NG", "NGA", "NGR":
		return "NGR"
	case "US", "USA":
		return "USA"
	default:
		return trimmed
	}
}

func extractLocationValue(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	for _, key := range []string{"Location", "location", "CountryCode", "country_code", "Country", "country"} {
		if value := normalizeCostLocation(extractString(lookupValue(payload, key))); value != "" {
			return value
		}
	}
	for _, key := range []string{"User", "user", "Owner", "owner"} {
		nested, ok := lookupValue(payload, key).(map[string]interface{})
		if !ok {
			continue
		}
		if value := extractLocationValue(nested); value != "" {
			return value
		}
	}
	return ""
}

func normalizeClusterCostResponse(resp map[string]interface{}) map[string]interface{} {
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return resp
	}
	if _, ok := data["costs"]; ok {
		return resp
	}
	if total, ok := data["total"]; ok {
		data["costs"] = total
	}
	return resp
}

func isClusterCostAllocationFetchError(err error) bool {
	var apiErr *pipeops.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	if apiErr.Response.StatusCode != http.StatusBadRequest {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(apiErr.Message))
	return strings.Contains(message, "error fetching cost alloction") || strings.Contains(message, "error fetching cost allocation")
}

func (s *Server) fetchClustersForWorkspaceReference(ctx context.Context, workspace workspaceReference) ([]map[string]interface{}, string, string, error) {
	identifiers := workspaceReferenceIdentifiers(workspace)
	var lastErr error
	for _, identifier := range identifiers {
		clusters, status, message, err := s.fetchWorkspaceClusters(ctx, identifier)
		if err == nil {
			return clusters, status, message, nil
		}
		if !isWorkspaceProjectsFallbackError(err) {
			return nil, "", "", err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", "", lastErr
	}
	return []map[string]interface{}{}, "success", "", nil
}

func (s *Server) fetchWorkspaceClusters(ctx context.Context, workspaceID string) ([]map[string]interface{}, string, string, error) {
	req, err := s.client.NewRequest(http.MethodGet, fmt.Sprintf("workspace/fetch/%s", workspaceID), nil)
	if err != nil {
		return nil, "", "", err
	}

	var envelope struct {
		Success bool            `json:"success,omitempty"`
		Status  string          `json:"status,omitempty"`
		Message string          `json:"message,omitempty"`
		Data    json.RawMessage `json:"data,omitempty"`
	}
	if _, err := s.client.Do(ctx, req, &envelope); err != nil {
		return nil, "", "", err
	}

	clusters, err := parseWorkspaceClusters(envelope.Data)
	if err != nil {
		return nil, "", "", err
	}
	return clusters, envelopeStatus(envelope.Status, envelope.Success), envelope.Message, nil
}

func parseWorkspaceClusters(data json.RawMessage) ([]map[string]interface{}, error) {
	workspace, err := extractWorkspacePayload(data)
	if err != nil {
		return nil, err
	}

	clusters := mapSliceValue(workspace, "clusters", "Clusters")
	normalized := make([]map[string]interface{}, 0, len(clusters))
	for _, cluster := range clusters {
		normalized = append(normalized, normalizeCluster(cluster, workspace))
	}
	return normalized, nil
}

func (s *Server) findClusterReference(ctx context.Context, clusterID, workspaceID string) (workspaceReference, map[string]interface{}, error) {
	var (
		workspaces []workspaceReference
		err        error
	)
	if workspaceID != "" {
		workspace, resolveErr := s.resolveWorkspaceReference(ctx, workspaceID)
		if resolveErr != nil {
			return workspaceReference{}, nil, resolveErr
		}
		workspaces = []workspaceReference{workspace}
	} else {
		workspaces, _, _, err = s.listWorkspaceReferences(ctx)
		if err != nil {
			return workspaceReference{}, nil, err
		}
	}

	for _, workspace := range workspaces {
		clusters, _, _, fetchErr := s.fetchClustersForWorkspaceReference(ctx, workspace)
		if fetchErr != nil {
			if isWorkspaceProjectsFallbackError(fetchErr) {
				continue
			}
			return workspaceReference{}, nil, fetchErr
		}
		for _, cluster := range clusters {
			if clusterMatchesIdentifier(cluster, clusterID) {
				return workspace, cluster, nil
			}
		}
	}

	return workspaceReference{}, nil, nil
}

func paginateProjects(projects []map[string]interface{}, page, limit int) []map[string]interface{} {
	if limit <= 0 {
		return projects
	}
	if page <= 0 {
		page = 1
	}

	start := (page - 1) * limit
	if start >= len(projects) {
		return []map[string]interface{}{}
	}

	end := start + limit
	if end > len(projects) {
		end = len(projects)
	}

	return projects[start:end]
}

func mapSliceValue(payload map[string]interface{}, keys ...string) []map[string]interface{} {
	value := lookupValue(payload, keys...)
	slice, ok := value.([]interface{})
	if !ok {
		return []map[string]interface{}{}
	}

	items := make([]map[string]interface{}, 0, len(slice))
	for _, item := range slice {
		if asMap, ok := item.(map[string]interface{}); ok {
			items = append(items, asMap)
		}
	}
	return items
}

func asMapSlice(data json.RawMessage) ([]map[string]interface{}, error) {
	var slice []map[string]interface{}
	if err := json.Unmarshal(data, &slice); err != nil {
		return nil, err
	}
	return slice, nil
}

func lookupValue(payload map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value
		}
	}
	return nil
}

func extractString(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func isUsableWorkspaceIdentifier(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "0" || trimmed == "0.0" || trimmed == "<nil>" {
		return false
	}
	return true
}

func isLikelyUUID(value string) bool {
	trimmed := strings.TrimSpace(value)
	return len(trimmed) >= 32 && strings.Count(trimmed, "-") == 4
}

func responseStatus(status string) string {
	if status != "" {
		return status
	}
	return "success"
}

func envelopeStatus(status string, success bool) string {
	if status != "" {
		return status
	}
	if success {
		return "success"
	}
	return "error"
}

func isWorkspaceProjectsFallbackError(err error) bool {
	var apiErr *pipeops.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	switch apiErr.Response.StatusCode {
	case http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound:
		return true
	default:
		return false
	}
}

func isServerFetchFallbackError(err error) bool {
	if isWorkspaceProjectsFallbackError(err) {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "no cluster data returned")
}

func isProjectLogsRetryableError(err error) bool {
	if isWorkspaceProjectsFallbackError(err) {
		return true
	}
	var apiErr *pipeops.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	return apiErr.Response.StatusCode >= http.StatusInternalServerError
}

func isProjectDeploymentRetryableError(err error) bool {
	if isWorkspaceProjectsFallbackError(err) {
		return true
	}
	var apiErr *pipeops.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	return apiErr.Response.StatusCode >= http.StatusInternalServerError
}

func projectLogIdentifiers(project map[string]interface{}) []string {
	identifiers := make([]string, 0, 6)
	for _, key := range []string{"UUID", "uuid", "ID", "id", "NameSlug", "name_slug", "Slug", "slug", "ProjectSlug", "project_slug", "Name", "name"} {
		identifiers = appendUniqueString(identifiers, extractString(lookupValue(project, key)))
	}
	return identifiers
}

func isBillingWorkspaceRequiredError(err error) bool {
	var apiErr *pipeops.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	if apiErr.Response.StatusCode != http.StatusBadRequest {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(apiErr.Message))
	return strings.Contains(message, "workspace")
}

func appendUniqueString(values []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return values
	}
	for _, existing := range values {
		if existing == trimmed {
			return values
		}
	}
	return append(values, trimmed)
}

func (s *Server) getProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req getProjectArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	workspaceUUID := ""
	if req.WorkspaceID != "" {
		workspace, err := s.resolveWorkspaceReference(ctx, req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		workspaceUUID = projectWorkspaceUUID(nil, workspace)
	}

	var directErr error
	if isLikelyDirectProjectIdentifier(req.ProjectID) {
		var (
			resp *pipeops.ProjectResponse
			err  error
		)
		if workspaceUUID != "" {
			resp, _, err = s.client.Projects.Get(ctx, req.ProjectID, &pipeops.ProjectGetOptions{WorkspaceUUID: workspaceUUID})
		} else {
			resp, _, err = s.client.Projects.Get(ctx, req.ProjectID)
		}
		if err == nil {
			return jsonResult(resp)
		}
		if !isWorkspaceProjectsFallbackError(err) {
			return nil, err
		}
		directErr = err
	}

	workspace, project, err := s.findProjectReferenceWithFallback(ctx, req.ProjectID, req.WorkspaceID, true)
	if err != nil {
		return nil, err
	}
	if project == nil {
		if directErr != nil {
			return nil, directErr
		}
		return nil, fmt.Errorf("project %q not found", req.ProjectID)
	}

	resolvedProjectID := projectIdentity(project)
	resolvedWorkspaceUUID := projectWorkspaceUUID(project, workspace)
	if resolvedProjectID != "" {
		var (
			resp *pipeops.ProjectResponse
			err  error
		)
		if resolvedWorkspaceUUID != "" {
			resp, _, err = s.client.Projects.Get(ctx, resolvedProjectID, &pipeops.ProjectGetOptions{WorkspaceUUID: resolvedWorkspaceUUID})
		} else {
			resp, _, err = s.client.Projects.Get(ctx, resolvedProjectID)
		}
		if err == nil {
			return jsonResult(resp)
		}
		if !isWorkspaceProjectsFallbackError(err) {
			return nil, err
		}
	}

	return jsonResult(map[string]interface{}{
		"status":  "success",
		"message": "ok",
		"data": map[string]interface{}{
			"project": project,
		},
	})
}

func (s *Server) createProjectTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name, err := requiredString(args, "name")
	if err != nil {
		return nil, err
	}
	username, err := requiredString(args, "username")
	if err != nil {
		return nil, err
	}
	source, err := requiredString(args, "source")
	if err != nil {
		return nil, err
	}
	repository, err := requiredString(args, "repository")
	if err != nil {
		return nil, err
	}
	branch, err := requiredString(args, "branch")
	if err != nil {
		return nil, err
	}

	clusterUUID := firstNonEmptyString(
		optionalStringArg(args, "cluster_uuid"),
		optionalStringArg(args, "clusterUUID"),
		optionalStringArg(args, "server_id"), // legacy alias
	)
	if clusterUUID == "" {
		return nil, fmt.Errorf("cluster_uuid is required (server_id is accepted as a legacy alias)")
	}

	envUUID := firstNonEmptyString(
		optionalStringArg(args, "environment_uuid"),
		optionalStringArg(args, "environment_id"), // legacy alias
	)
	if envUUID == "" {
		return nil, fmt.Errorf("environment_uuid is required (environment_id is accepted as a legacy alias)")
	}

	// Prefer client-supplied environment; SDK ApplyCreateProjectDefaults fills "development" if empty.
	environment := optionalStringArg(args, "environment")

	// workspace_uuid is required by POST /project/create — never omit.
	// Resolve from workspace_id / workspace_uuid args, else first workspace.
	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}
	if strings.TrimSpace(workspaceUUID) == "" {
		return nil, fmt.Errorf("workspace_uuid is required")
	}

	buildMethod := optionalStringArg(args, "build_method")
	buildCommand := optionalStringArg(args, "build_command")
	runCommand := firstNonEmptyString(
		optionalStringArg(args, "run_command"),
		optionalStringArg(args, "start_command"), // legacy alias
	)

	buildSettings := pipeops.CreateProjectBuildSettings{
		BuildMethod:  buildMethod,
		BuildCommand: buildCommand,
		RunCommand:   runCommand,
	}
	if nested, ok := args["build_settings"].(map[string]interface{}); ok && nested != nil {
		// Nested object fields override top-level build_* when set (client explicit wins).
		mergeCreateProjectBuildSettings(&buildSettings, nested)
	}

	var networkSettings []pipeops.CreateProjectNetworkSetting
	if port, ok := optionalInt32Arg(args, "port"); ok && port > 0 {
		// Protocol default applied by ApplyCreateProjectDefaults if empty.
		networkSettings = []pipeops.CreateProjectNetworkSetting{
			{Port: port, Protocol: optionalStringArg(args, "protocol")},
		}
	}

	// Client env_vars win; SDK injects PORT only if missing and network port is set.
	envVariables := createProjectEnvVarsFromArgs(args)

	req := &pipeops.CreateProjectRequest{
		Name:               name,
		Username:           username,
		Source:             source, // empty → github via ApplyCreateProjectDefaults
		Repository:         repository,
		Branch:             branch,
		ClusterUUID:        clusterUUID,
		EnvironmentUUID:    envUUID,
		Environment:        environment,
		WorkspaceUUID:      workspaceUUID,
		CommitURL:          optionalStringArg(args, "commit_url"),
		CommitSha:          optionalStringArg(args, "commit_sha"),
		RepositoryLanguage: firstNonEmptyString(optionalStringArg(args, "language"), optionalStringArg(args, "repository_language")),
		RawLanguage:        optionalStringArg(args, "raw_language"),
		Framework:          optionalStringArg(args, "framework"),
		GitlabID:           optionalStringArg(args, "gitlab_id"),
		ClusterVersion:     optionalStringArg(args, "cluster_version"),
		CustomDomainName:   optionalStringArg(args, "custom_domain_name"),
		PostStart:          optionalStringArg(args, "post_start"),
		WorkerRunCommand:   optionalStringArg(args, "worker_run_command"),
		BuildSettings:      buildSettings,
		NetworkSettings:    networkSettings,
		EnvVariables:       envVariables,
		Kind:               optionalStringArg(args, "kind"),
		ProjectType:        firstNonEmptyString(optionalStringArg(args, "project_type"), optionalStringArg(args, "projectType")),
	}

	// Create applies prefer-client defaults (PORT, worker, protocol, source, environment, …).
	resp, _, err := s.client.Projects.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func optionalStringArg(args map[string]interface{}, key string) string {
	if args == nil {
		return ""
	}
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func optionalInt32Arg(args map[string]interface{}, key string) (int32, bool) {
	if args == nil {
		return 0, false
	}
	value, ok := args[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	case float64:
		return int32(v), true
	case float32:
		return int32(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int32(i), true
	default:
		return 0, false
	}
}

func createProjectEnvVarsFromArgs(args map[string]interface{}) []pipeops.CreateProjectEnvVar {
	// Prefer map form env_vars; also accept env_variables array of {key,value}.
	if raw, ok := args["env_vars"]; ok && raw != nil {
		switch m := raw.(type) {
		case map[string]interface{}:
			out := make([]pipeops.CreateProjectEnvVar, 0, len(m))
			for k, v := range m {
				out = append(out, pipeops.CreateProjectEnvVar{
					Key:   k,
					Value: fmt.Sprint(v),
				})
			}
			return out
		case map[string]string:
			out := make([]pipeops.CreateProjectEnvVar, 0, len(m))
			for k, v := range m {
				out = append(out, pipeops.CreateProjectEnvVar{Key: k, Value: v})
			}
			return out
		}
	}

	if raw, ok := args["env_variables"].([]interface{}); ok {
		out := make([]pipeops.CreateProjectEnvVar, 0, len(raw))
		for _, item := range raw {
			entry, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			key := firstNonEmptyString(optionalStringArg(entry, "key"), optionalStringArg(entry, "Key"))
			if key == "" {
				continue
			}
			value := firstNonEmptyString(optionalStringArg(entry, "value"), optionalStringArg(entry, "Value"))
			out = append(out, pipeops.CreateProjectEnvVar{Key: key, Value: value})
		}
		return out
	}

	// Never omit: empty slice is fine (SDK also normalizes nil → []).
	return []pipeops.CreateProjectEnvVar{}
}

func mergeCreateProjectBuildSettings(dst *pipeops.CreateProjectBuildSettings, nested map[string]interface{}) {
	if dst == nil || nested == nil {
		return
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "type"), optionalStringArg(nested, "Type")); v != "" {
		dst.Type = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "build_method"), optionalStringArg(nested, "buildMethod")); v != "" {
		dst.BuildMethod = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "build_command"), optionalStringArg(nested, "buildCommand")); v != "" {
		dst.BuildCommand = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "run_command"), optionalStringArg(nested, "runCommand")); v != "" {
		dst.RunCommand = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "build_path"), optionalStringArg(nested, "buildPath")); v != "" {
		dst.BuildPath = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "build_directory"), optionalStringArg(nested, "buildDirectory")); v != "" {
		dst.BuildDirectory = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "builder_host"), optionalStringArg(nested, "builderHost")); v != "" {
		dst.BuilderHost = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "builder_id"), optionalStringArg(nested, "builderID")); v != "" {
		dst.BuilderID = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "build_version"), optionalStringArg(nested, "buildVersion")); v != "" {
		dst.BuildVersion = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "docker_image_url"), optionalStringArg(nested, "dockerImageURL")); v != "" {
		dst.DockerImageURL = v
	}
	if v := firstNonEmptyString(optionalStringArg(nested, "docker_path"), optionalStringArg(nested, "dockerPath")); v != "" {
		dst.DockerPath = v
	}
	if worker, ok := nested["worker"].(bool); ok {
		dst.Worker = &worker
	}
	if skip, ok := nested["skip_build"].(bool); ok {
		dst.SkipBuild = skip
	} else if skip, ok := nested["skipBuild"].(bool); ok {
		dst.SkipBuild = skip
	}
	if skip, ok := nested["skip_commit"].(bool); ok {
		dst.SkipCommit = skip
	} else if skip, ok := nested["skipCommit"].(bool); ok {
		dst.SkipCommit = skip
	}
	if use, ok := nested["use_docker_image"].(bool); ok {
		dst.UseDockerImage = use
	} else if use, ok := nested["useDockerImage"].(bool); ok {
		dst.UseDockerImage = use
	}
	if noCache, ok := nested["no_cache"].(bool); ok {
		dst.NoCache = noCache
	} else if noCache, ok := nested["noCache"].(bool); ok {
		dst.NoCache = noCache
	}
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

	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}

	noCache := false
	if value, ok := args["no_cache"]; ok {
		var valid bool
		noCache, valid = value.(bool)
		if !valid {
			return nil, fmt.Errorf("no_cache must be a boolean")
		}
	}

	if _, err := s.client.Projects.Deploy(ctx, projectID, &pipeops.ProjectDeployOptions{
		WorkspaceUUID: workspaceUUID,
		NoCache:       noCache,
	}); err != nil {
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

	workspaceUUID, err := s.resolveWorkspaceUUID(ctx, req.WorkspaceID)
	if err != nil {
		return nil, err
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
		WorkspaceUUID:   workspaceUUID,
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

	buildLogsOptions := func(workspaceUUID string) *pipeops.LogsOptions {
		return &pipeops.LogsOptions{
			App:           "project",
			WorkspaceUUID: workspaceUUID,
			StartTime:     request.StartTime,
			EndTime:       request.EndTime,
			Limit:         request.Limit,
			Search:        request.Search,
		}
	}

	tryLogs := func(projectIdentifiers []string, workspaceUUIDs []string) (interface{}, error) {
		if len(workspaceUUIDs) == 0 {
			workspaceUUIDs = []string{""}
		}

		attempted := make(map[string]struct{})
		var lastErr error
		for _, projectIdentifier := range projectIdentifiers {
			projectIdentifier = strings.TrimSpace(projectIdentifier)
			if projectIdentifier == "" {
				continue
			}
			for _, workspaceUUID := range workspaceUUIDs {
				workspaceUUID = strings.TrimSpace(workspaceUUID)
				attemptKey := projectIdentifier + "|" + workspaceUUID
				if _, ok := attempted[attemptKey]; ok {
					continue
				}
				attempted[attemptKey] = struct{}{}

				resp, err := s.requestProjectLogs(ctx, projectIdentifier, buildLogsOptions(workspaceUUID), workspaceUUID != "")
				if err == nil {
					return jsonResult(resp)
				}
				if !isProjectLogsRetryableError(err) {
					return nil, err
				}
				lastErr = err
			}
		}
		return nil, lastErr
	}

	explicitWorkspaceUUID := ""
	if request.WorkspaceID != "" {
		workspace, err := s.resolveWorkspaceReference(ctx, request.WorkspaceID)
		if err != nil {
			return nil, err
		}
		explicitWorkspaceUUID = projectWorkspaceUUID(nil, workspace)
	}

	var directErr error
	if isLikelyDirectProjectIdentifier(request.ProjectID) {
		directWorkspaceUUIDs := []string{}
		if explicitWorkspaceUUID != "" {
			directWorkspaceUUIDs = append(directWorkspaceUUIDs, explicitWorkspaceUUID)
		}
		directWorkspaceUUIDs = append(directWorkspaceUUIDs, "")

		result, err := tryLogs([]string{request.ProjectID}, directWorkspaceUUIDs)
		if err == nil && result != nil {
			return result, nil
		}
		if err != nil {
			if !isProjectLogsRetryableError(err) {
				return nil, err
			}
			directErr = err
		}
	}

	workspace, project, err := s.findProjectReference(ctx, request.ProjectID, request.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		if directErr != nil {
			return nil, directErr
		}
		return nil, fmt.Errorf("project %q not found", request.ProjectID)
	}

	workspaceUUIDs := []string{}
	if resolvedWorkspaceUUID := projectWorkspaceUUID(project, workspace); resolvedWorkspaceUUID != "" {
		workspaceUUIDs = append(workspaceUUIDs, resolvedWorkspaceUUID)
	}
	if explicitWorkspaceUUID != "" {
		workspaceUUIDs = append(workspaceUUIDs, explicitWorkspaceUUID)
	}
	workspaceUUIDs = append(workspaceUUIDs, "")

	result, err := tryLogs(projectLogIdentifiers(project), workspaceUUIDs)
	if err == nil && result != nil {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if directErr != nil {
		return nil, directErr
	}
	return nil, fmt.Errorf("project %q not found", request.ProjectID)
}

func (s *Server) listProjectDeploymentsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req projectDeploymentsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}

	resp, project, err := s.requestProjectDeploymentCollectionWithFallback(ctx, req.ProjectID, req.WorkspaceID, func(ctx context.Context, projectID, workspaceUUID string) (map[string]interface{}, error) {
		return s.requestProjectDeployments(ctx, projectID, workspaceUUID, req.FilterBy, req.Page, req.Limit)
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(attachProjectToCollectionResponse(resp, project))
}

func (s *Server) listProjectDeploymentHistoryTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req projectDeploymentsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}

	resp, project, err := s.requestProjectDeploymentCollectionWithFallback(ctx, req.ProjectID, req.WorkspaceID, func(ctx context.Context, projectID, workspaceUUID string) (map[string]interface{}, error) {
		return s.requestProjectDeploymentHistory(ctx, projectID, workspaceUUID, req.Page, req.Limit)
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(attachProjectToCollectionResponse(resp, project))
}

func (s *Server) searchProjectDeploymentsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req searchProjectDeploymentsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Search) == "" {
		return nil, fmt.Errorf("search is required")
	}

	resp, project, err := s.requestProjectDeploymentCollectionWithFallback(ctx, req.ProjectID, req.WorkspaceID, func(ctx context.Context, projectID, workspaceUUID string) (map[string]interface{}, error) {
		return s.requestProjectDeployments(ctx, projectID, workspaceUUID, req.FilterBy, req.Page, req.Limit)
	})
	if err != nil {
		return nil, err
	}
	filtered := filterDeploymentItems(responseCollectionItems(resp, "deployments"), req.Search)
	resp = setCollectionItems(resp, "deployments", filtered)
	resp = attachProjectToCollectionResponse(resp, project)
	if meta, ok := resp["meta"].(map[string]interface{}); ok {
		meta["search"] = strings.TrimSpace(req.Search)
		resp["meta"] = meta
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

	// Prefer-client default: merge so agents can set keys without wiping the full set.
	merge := true
	if request.Merge != nil {
		merge = *request.Merge
	}

	var workspaceUUID string
	if request.WorkspaceID != "" {
		ws, err := s.resolveWorkspaceUUID(ctx, request.WorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace: %w", err)
		}
		workspaceUUID = ws
	}

	resp, _, err := s.client.Projects.UpdateEnvVariables(ctx, request.ProjectID, &pipeops.EnvVariablesRequest{
		EnvVariables:  request.EnvVariables,
		Merge:         merge,
		WorkspaceUUID: workspaceUUID,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateProjectDeploySettingsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var request projectDeploySettingsArgs
	if err := decodeArguments(args, &request); err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.ProjectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	var workspaceUUID string
	if request.WorkspaceID != "" {
		ws, err := s.resolveWorkspaceUUID(ctx, request.WorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace: %w", err)
		}
		workspaceUUID = ws
	} else {
		ws, err := s.resolveDefaultWorkspaceUUID(ctx, args)
		if err == nil {
			workspaceUUID = ws
		}
	}

	resp, _, err := s.client.Projects.UpdateDeploySettings(ctx, request.ProjectID, &pipeops.DeploySettingsRequest{
		AutoDeployEnabled: request.AutoDeployEnabled,
		Branch:            request.Branch,
		AutoRollback:      request.AutoRollback,
		UserName:          request.Username,
		Repository:        request.Repository,
		WorkspaceUUID:     workspaceUUID,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateProjectSecurityPolicyTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var request projectSecurityPolicyArgs
	if err := decodeArguments(args, &request); err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.ProjectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	var workspaceUUID string
	if request.WorkspaceID != "" {
		ws, err := s.resolveWorkspaceUUID(ctx, request.WorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace: %w", err)
		}
		workspaceUUID = ws
	} else {
		ws, err := s.resolveDefaultWorkspaceUUID(ctx, args)
		if err == nil {
			workspaceUUID = ws
		}
	}

	resp, _, err := s.client.Projects.UpdateSecurityPolicy(ctx, request.ProjectID, &pipeops.SecurityPolicyRequest{
		Enabled:       request.Enabled,
		MaxCritical:   request.MaxCritical,
		MaxHigh:       request.MaxHigh,
		MaxMedium:     request.MaxMedium,
		MaxCvssScore:  request.MaxCvssScore,
		MaxTotalVulns: request.MaxTotalVulns,
		FailOnSecrets: request.FailOnSecrets,
		WorkspaceUUID: workspaceUUID,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listVCSOrganizationsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req vcsProviderArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}

	provider, err := normalizeVCSProvider(req.Provider)
	if err != nil {
		return nil, err
	}

	resp, err := s.requestJSON(ctx, http.MethodGet, fmt.Sprintf("project/%s/organisations", provider), nil)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listVCSRepositoriesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req vcsRepositoriesArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	provider, err := normalizeVCSProvider(req.Provider)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.OrgName) == "" {
		return nil, fmt.Errorf("org_name is required")
	}

	path := fmt.Sprintf("project/%s/organisations/repos", provider)
	if req.Page > 0 {
		path = withQueryValues(path, map[string]string{"page": fmt.Sprintf("%d", req.Page)})
	}

	resp, err := s.requestJSON(ctx, http.MethodPost, path, map[string]interface{}{
		"org_name": strings.TrimSpace(req.OrgName),
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) searchVCSRepositoriesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req vcsRepoSearchArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	provider, err := normalizeVCSProvider(req.Provider)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.OrgName) == "" {
		return nil, fmt.Errorf("org_name is required")
	}
	if strings.TrimSpace(req.RepositoryName) == "" {
		return nil, fmt.Errorf("repository_name is required")
	}

	path := fmt.Sprintf("project/%s/repo-search", provider)
	if req.Page > 0 {
		path = withQueryValues(path, map[string]string{"page": fmt.Sprintf("%d", req.Page)})
	}

	resp, err := s.requestJSON(ctx, http.MethodPost, path, map[string]interface{}{
		"org_name":        strings.TrimSpace(req.OrgName),
		"repository_name": strings.TrimSpace(req.RepositoryName),
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listVCSBranchesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req vcsBranchesArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	provider, err := normalizeVCSProvider(req.Provider)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.RepoFullname) == "" {
		return nil, fmt.Errorf("repo_fullname is required")
	}

	path := fmt.Sprintf("project/%s/branches", provider)
	if strings.TrimSpace(req.Search) != "" {
		path = withQueryValues(path, map[string]string{"search": strings.TrimSpace(req.Search)})
	}

	body := map[string]interface{}{
		"repo_fullname": strings.TrimSpace(req.RepoFullname),
	}
	if strings.TrimSpace(req.Visibility) != "" {
		body["visibility"] = strings.TrimSpace(req.Visibility)
	}

	resp, err := s.requestJSON(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) checkRepositoryDockerfileTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req vcsDockerfileArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	provider, err := normalizeVCSProvider(req.Provider)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Owner) == "" {
		return nil, fmt.Errorf("owner is required")
	}
	if strings.TrimSpace(req.Repository) == "" {
		return nil, fmt.Errorf("repository is required")
	}
	if strings.TrimSpace(req.Branch) == "" {
		return nil, fmt.Errorf("branch is required")
	}

	path := fmt.Sprintf(
		"project/check-dockerfile/%s/%s/%s/%s",
		provider,
		url.PathEscape(strings.TrimSpace(req.Owner)),
		url.PathEscape(strings.TrimSpace(req.Repository)),
		url.PathEscape(strings.TrimSpace(req.Branch)),
	)
	resp, err := s.requestJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) linkVCSProviderTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req linkVCSProviderArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	provider, err := normalizeVCSProvider(req.Provider)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.RedirectPath) == "" {
		return nil, fmt.Errorf("redirect_path is required")
	}

	resp, err := s.requestJSON(ctx, http.MethodPost, fmt.Sprintf("project/link/%s", provider), map[string]interface{}{
		"redirectPath": strings.TrimSpace(req.RedirectPath),
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
	workspaceUUID, err := s.resolveWorkspaceUUID(ctx, req.WorkspaceID)
	if err != nil {
		return nil, err
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

	resp, _, err := s.client.ExternalRegistries.Create(ctx, workspaceUUID, &pipeops.CreateExternalRegistryRequest{
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

	workspaceUUID, err := s.resolveWorkspaceUUID(ctx, req.WorkspaceID)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ExternalRegistries.List(ctx, workspaceUUID, &pipeops.ExternalRegistryListOptions{
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
	workspaceID, _ := args["workspace_id"].(string)

	var (
		resp map[string]interface{}
		err  error
	)
	if strings.TrimSpace(workspaceID) != "" {
		resp, err = s.listServersForWorkspace(ctx, workspaceID)
	} else {
		resp, err = s.listServersAcrossWorkspaces(ctx)
	}
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listServersAcrossWorkspaces(ctx context.Context) (map[string]interface{}, error) {
	workspaces, status, message, err := s.listWorkspaceReferences(ctx)
	if err != nil {
		return nil, err
	}

	servers := make([]map[string]interface{}, 0)
	seen := make(map[string]struct{})
	for _, workspace := range workspaces {
		workspaceClusters, workspaceStatus, workspaceMessage, workspaceErr := s.fetchClustersForWorkspaceReference(ctx, workspace)
		if workspaceErr != nil {
			if isWorkspaceProjectsFallbackError(workspaceErr) {
				continue
			}
			return nil, workspaceErr
		}
		if status == "" {
			status = workspaceStatus
		}
		if message == "" {
			message = workspaceMessage
		}
		for _, cluster := range workspaceClusters {
			clusterKey := clusterIdentity(cluster)
			if clusterKey != "" {
				if _, ok := seen[clusterKey]; ok {
					continue
				}
				seen[clusterKey] = struct{}{}
			}
			servers = append(servers, normalizeServer(cluster, workspace))
		}
	}

	return map[string]interface{}{
		"status":  responseStatus(status),
		"message": message,
		"data": map[string]interface{}{
			"servers": servers,
		},
	}, nil
}

func (s *Server) listServersForWorkspace(ctx context.Context, workspaceID string) (map[string]interface{}, error) {
	workspaceRef, err := s.resolveWorkspaceReference(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	clusters, status, message, err := s.fetchClustersForWorkspaceReference(ctx, workspaceRef)
	if err != nil {
		if isWorkspaceProjectsFallbackError(err) {
			return map[string]interface{}{
				"status":  "success",
				"message": err.Error(),
				"data": map[string]interface{}{
					"servers": []map[string]interface{}{},
				},
			}, nil
		}
		return nil, err
	}

	servers := make([]map[string]interface{}, 0, len(clusters))
	for _, cluster := range clusters {
		servers = append(servers, normalizeServer(cluster, workspaceRef))
	}

	return map[string]interface{}{
		"status":  responseStatus(status),
		"message": message,
		"data": map[string]interface{}{
			"servers": servers,
		},
	}, nil
}

func (s *Server) getServerTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	serverID, err := requiredString(args, "server_id")
	if err != nil {
		return nil, err
	}
	workspaceID, _ := args["workspace_id"].(string)

	var directErr error
	if isLikelyDirectProjectIdentifier(serverID) && strings.TrimSpace(workspaceID) != "" {
		resolvedWorkspaceUUID, resolveErr := s.resolveWorkspaceUUID(ctx, workspaceID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		resp, _, err := s.client.Servers.Get(ctx, serverID, resolvedWorkspaceUUID)
		if err == nil {
			return jsonResult(resp)
		}
		if !isServerFetchFallbackError(err) {
			return nil, err
		}
		directErr = err
	}

	workspace, cluster, err := s.findClusterReference(ctx, serverID, workspaceID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		if directErr != nil {
			return nil, directErr
		}
		return nil, fmt.Errorf("server %q not found", serverID)
	}

	resolvedClusterID := clusterIdentity(cluster)
	resolvedWorkspaceUUID := clusterWorkspaceUUID(cluster, workspace)
	if resolvedClusterID != "" && resolvedWorkspaceUUID != "" {
		resp, _, err := s.client.Servers.Get(ctx, resolvedClusterID, resolvedWorkspaceUUID)
		if err == nil {
			return jsonResult(resp)
		}
		if !isServerFetchFallbackError(err) {
			return nil, err
		}
	}

	return jsonResult(map[string]interface{}{
		"status":  "success",
		"message": "ok",
		"data": map[string]interface{}{
			"server": normalizeServer(cluster, workspace),
		},
	})
}

func (s *Server) getClusterConnectionTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	clusterID, _ := args["server_id"].(string)
	if clusterID == "" {
		clusterID, _ = args["cluster_id"].(string)
	}
	if clusterID == "" {
		return nil, fmt.Errorf("server_id is required")
	}

	var directErr error
	if isLikelyDirectProjectIdentifier(clusterID) {
		resp, _, err := s.client.Servers.GetClusterConnection(ctx, clusterID)
		if err == nil {
			return jsonResult(resp)
		}
		if !isWorkspaceProjectsFallbackError(err) {
			return nil, err
		}
		directErr = err
	}

	_, cluster, err := s.findClusterReference(ctx, clusterID, "")
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		if directErr != nil {
			return nil, directErr
		}
		return nil, fmt.Errorf("server %q not found", clusterID)
	}

	resolvedClusterID := clusterIdentity(cluster)
	if resolvedClusterID == "" {
		if directErr != nil {
			return nil, directErr
		}
		return nil, fmt.Errorf("server %q not found", clusterID)
	}

	resp, _, err := s.client.Servers.GetClusterConnection(ctx, resolvedClusterID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getClusterCostAllocationTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	clusterID, _ := args["server_id"].(string)
	if clusterID == "" {
		clusterID, _ = args["cluster_id"].(string)
	}
	if clusterID == "" {
		return nil, fmt.Errorf("server_id is required")
	}
	workspaceID, _ := args["workspace_id"].(string)
	aggregate, _ := args["aggregate"].(string)
	window, _ := args["window"].(string)
	location, _ := args["location"].(string)

	resolvedClusterID := clusterID
	candidateWorkspaceUUIDs := make([]string, 0, 2)
	attemptedWorkspaceUUIDs := make(map[string]struct{})
	var discoveryErr error
	path := fmt.Sprintf("cluster/%s/cost/allocation/compute", resolvedClusterID)

	if workspaceID != "" {
		resolvedWorkspaceUUID, resolveErr := s.resolveDefaultWorkspaceUUID(ctx, map[string]interface{}{"workspace_id": workspaceID})
		if resolveErr != nil {
			if discoveryErr == nil {
				discoveryErr = resolveErr
			}
			if isLikelyUUID(workspaceID) {
				candidateWorkspaceUUIDs = appendUniqueString(candidateWorkspaceUUIDs, workspaceID)
			}
		} else {
			candidateWorkspaceUUIDs = appendUniqueString(candidateWorkspaceUUIDs, resolvedWorkspaceUUID)
		}
	}

	workspace, cluster, err := s.findClusterReference(ctx, clusterID, workspaceID)
	if err != nil {
		if discoveryErr == nil {
			discoveryErr = err
		}
	} else {
		if clusterIdentityValue := clusterIdentity(cluster); clusterIdentityValue != "" {
			resolvedClusterID = clusterIdentityValue
			path = fmt.Sprintf("cluster/%s/cost/allocation/compute", resolvedClusterID)
		}
		candidateWorkspaceUUIDs = appendUniqueString(candidateWorkspaceUUIDs, clusterWorkspaceUUID(cluster, workspace))
		if strings.TrimSpace(location) == "" {
			location = extractLocationValue(cluster)
		}
	}

	tryWorkspaceCandidates := func(workspaceUUIDs []string) (interface{}, error) {
		var lastErr error
		for _, workspaceUUID := range workspaceUUIDs {
			workspaceUUID = strings.TrimSpace(workspaceUUID)
			if workspaceUUID == "" {
				continue
			}
			if _, ok := attemptedWorkspaceUUIDs[workspaceUUID]; ok {
				continue
			}
			attemptedWorkspaceUUIDs[workspaceUUID] = struct{}{}

			resp, requestErr := s.requestJSON(ctx, http.MethodGet, withClusterCostAllocationQuery(path, workspaceUUID, aggregate, window), nil)
			if requestErr == nil {
				return jsonResult(normalizeClusterCostResponse(resp))
			}
			if isClusterCostAllocationFetchError(requestErr) {
				fallbackResp, fallbackErr := s.requestJSON(ctx, http.MethodGet, withNovaClusterCostAllocationQuery("cluster/cost/allocation/compute", workspaceUUID, aggregate, window, resolvedClusterID, location), nil)
				if fallbackErr == nil {
					return jsonResult(normalizeClusterCostResponse(fallbackResp))
				}
				if !isWorkspaceProjectsFallbackError(fallbackErr) && !isClusterCostAllocationFetchError(fallbackErr) {
					return nil, fallbackErr
				}
				lastErr = fallbackErr
				continue
			}
			if !isWorkspaceProjectsFallbackError(requestErr) {
				return nil, requestErr
			}
			lastErr = requestErr
		}
		return nil, lastErr
	}

	result, err := tryWorkspaceCandidates(candidateWorkspaceUUIDs)
	if err == nil && result != nil {
		return result, nil
	}
	lastErr := err

	if len(attemptedWorkspaceUUIDs) == 0 || lastErr != nil {
		workspaces, _, _, listErr := s.listWorkspaceReferences(ctx)
		if listErr != nil {
			if discoveryErr == nil {
				discoveryErr = listErr
			}
		} else {
			fallbackWorkspaceUUIDs := make([]string, 0, len(workspaces))
			for _, workspace := range workspaces {
				fallbackWorkspaceUUIDs = appendUniqueString(fallbackWorkspaceUUIDs, workspace.UUID)
			}
			result, err = tryWorkspaceCandidates(fallbackWorkspaceUUIDs)
			if err == nil && result != nil {
				return result, nil
			}
			if err != nil {
				lastErr = err
			}
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	if discoveryErr != nil {
		return nil, discoveryErr
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("server %q not found", clusterID)
	}
	return nil, fmt.Errorf("workspace not found for server %q", clusterID)
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

func (s *Server) listEnvironmentsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req listEnvironmentsArgs
	if len(args) > 0 {
		if err := decodeArguments(args, &req); err != nil {
			return nil, err
		}
	}

	var (
		resp map[string]interface{}
		err  error
	)
	if req.WorkspaceID != "" {
		resp, err = s.listEnvironmentsForWorkspace(ctx, req.WorkspaceID)
	} else {
		resp, err = s.listEnvironmentsAcrossWorkspaces(ctx)
	}
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listEnvironmentsAcrossWorkspaces(ctx context.Context) (map[string]interface{}, error) {
	workspaces, status, message, err := s.listWorkspaceReferences(ctx)
	if err != nil {
		return nil, err
	}

	environments := make([]map[string]interface{}, 0)
	seen := make(map[string]struct{})
	for _, workspace := range workspaces {
		workspaceEnvironments, workspaceStatus, workspaceMessage, workspaceErr := s.fetchEnvironmentsForWorkspaceReference(ctx, workspace)
		if workspaceErr != nil {
			if isWorkspaceProjectsFallbackError(workspaceErr) {
				continue
			}
			return nil, workspaceErr
		}
		if status == "" {
			status = workspaceStatus
		}
		if message == "" {
			message = workspaceMessage
		}
		for _, environment := range workspaceEnvironments {
			environmentKey := environmentIdentity(environment)
			if environmentKey != "" {
				if _, ok := seen[environmentKey]; ok {
					continue
				}
				seen[environmentKey] = struct{}{}
			}
			environments = append(environments, environment)
		}
	}

	return map[string]interface{}{
		"status":  responseStatus(status),
		"message": message,
		"data": map[string]interface{}{
			"environments": environments,
		},
	}, nil
}

func (s *Server) listEnvironmentsForWorkspace(ctx context.Context, workspaceID string) (map[string]interface{}, error) {
	workspaceRef, err := s.resolveWorkspaceReference(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	environments, status, message, err := s.fetchEnvironmentsForWorkspaceReference(ctx, workspaceRef)
	if err != nil {
		if isWorkspaceProjectsFallbackError(err) {
			return map[string]interface{}{
				"status":  "success",
				"message": err.Error(),
				"data": map[string]interface{}{
					"environments": []map[string]interface{}{},
				},
			}, nil
		}
		return nil, err
	}

	return map[string]interface{}{
		"status":  responseStatus(status),
		"message": message,
		"data": map[string]interface{}{
			"environments": environments,
		},
	}, nil
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

	workspaceUUID, err := s.resolveWorkspaceUUID(ctx, req.WorkspaceID)
	if err != nil {
		return nil, err
	}
	req.WorkspaceUUID = workspaceUUID

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

func (s *Server) listAddOnsResponse(ctx context.Context, req listAddOnsArgs) (interface{}, error) {
	values := map[string]string{}
	if req.Page > 0 {
		values["page"] = fmt.Sprintf("%d", req.Page)
	}
	if req.Limit > 0 {
		values["limit"] = fmt.Sprintf("%d", req.Limit)
	}
	if strings.TrimSpace(req.Category) != "" {
		values["category"] = strings.TrimSpace(req.Category)
	}
	if strings.TrimSpace(req.Search) != "" {
		values["s"] = strings.TrimSpace(req.Search)
	}
	if req.Featured != nil {
		values["featured"] = fmt.Sprintf("%t", *req.Featured)
	}
	if strings.TrimSpace(req.WorkspaceID) != "" {
		workspaceUUID, err := s.resolveWorkspaceUUID(ctx, req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		values["workspace"] = workspaceUUID
	}

	resp, err := s.requestJSON(ctx, http.MethodGet, withQueryValues("addons", values), nil)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listAddOnsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req listAddOnsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	return s.listAddOnsResponse(ctx, req)
}

func (s *Server) searchAddOnsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req listAddOnsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Search) == "" {
		return nil, fmt.Errorf("search is required")
	}
	return s.listAddOnsResponse(ctx, req)
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
	if strings.TrimSpace(req.ServerID) == "" {
		return nil, fmt.Errorf("server_id is required")
	}

	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.AddOns.Deploy(ctx, &pipeops.DeployAddOnRequest{
		ID:          req.AddOnID,
		ProjectID:   req.ProjectID,
		Server:      req.ServerID,
		Workspace:   workspaceUUID,
		Environment: req.EnvironmentID,
		Tag:         req.Tag,
		Config:      req.Config,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listAddOnDeploymentsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.AddOns.ListDeployments(ctx, &pipeops.ListDeploymentsOptions{WorkspaceUUID: workspaceUUID})
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
	resp, err := s.requestJSON(ctx, http.MethodGet, "addons/categories", nil)
	if err != nil {
		return nil, err
	}
	return jsonResult(normalizeCollectionResponse(resp, "categories"))
}

func (s *Server) volumeListOptions(ctx context.Context, args map[string]interface{}, status, clusterUUID string, limit, offset int) (*pipeops.VolumeListOptions, error) {
	opts := &pipeops.VolumeListOptions{
		Status:      strings.TrimSpace(status),
		ClusterUUID: strings.TrimSpace(clusterUUID),
		Limit:       limit,
		Offset:      offset,
	}
	if workspaceID, ok := args["workspace_id"].(string); ok && strings.TrimSpace(workspaceID) != "" {
		workspaceUUID, err := s.resolveWorkspaceUUID(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		opts.WorkspaceUUID = workspaceUUID
	} else {
		workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
		if err != nil {
			return nil, err
		}
		opts.WorkspaceUUID = workspaceUUID
	}
	return opts, nil
}

func (s *Server) listVolumesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req listVolumesArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}

	opts, err := s.volumeListOptions(ctx, args, req.Status, req.ClusterUUID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Volumes.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getVolumeTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req volumeUUIDArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.VolumeUUID) == "" {
		return nil, fmt.Errorf("volume_uuid is required")
	}

	opts, err := s.volumeListOptions(ctx, args, "", "", 0, 0)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Volumes.Get(ctx, req.VolumeUUID, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) remountVolumeTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req remountVolumeArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.VolumeUUID) == "" {
		return nil, fmt.Errorf("volume_uuid is required")
	}
	if strings.TrimSpace(req.TargetType) == "" {
		return nil, fmt.Errorf("target_type is required")
	}
	if strings.TrimSpace(req.TargetUUID) == "" {
		return nil, fmt.Errorf("target_uuid is required")
	}

	opts, err := s.volumeListOptions(ctx, args, "", "", 0, 0)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Volumes.Remount(ctx, req.VolumeUUID, &pipeops.RemountVolumeRequest{
		TargetType: strings.TrimSpace(req.TargetType),
		TargetUUID: strings.TrimSpace(req.TargetUUID),
		MountPath:  strings.TrimSpace(req.MountPath),
	}, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deleteVolumeTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req volumeUUIDArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.VolumeUUID) == "" {
		return nil, fmt.Errorf("volume_uuid is required")
	}

	opts, err := s.volumeListOptions(ctx, args, "", "", 0, 0)
	if err != nil {
		return nil, err
	}

	if _, err := s.client.Volumes.Delete(ctx, req.VolumeUUID, opts); err != nil {
		return nil, err
	}
	return textResult("Volume deleted successfully"), nil
}

func (s *Server) exportVolumeTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req volumeUUIDArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.VolumeUUID) == "" {
		return nil, fmt.Errorf("volume_uuid is required")
	}

	opts, err := s.volumeListOptions(ctx, args, "", "", 0, 0)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Volumes.StartExport(ctx, req.VolumeUUID, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getVolumeExportTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req volumeUUIDArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.VolumeUUID) == "" {
		return nil, fmt.Errorf("volume_uuid is required")
	}

	opts, err := s.volumeListOptions(ctx, args, "", "", 0, 0)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.Volumes.GetExport(ctx, req.VolumeUUID, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listAddonBackupsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req listAddonBackupsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.DeploymentID) == "" {
		return nil, fmt.Errorf("deployment_id is required")
	}

	resp, _, err := s.client.AddOns.ListAddonBackups(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) startAddonBackupExportTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req startAddonBackupExportArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.DeploymentID) == "" {
		return nil, fmt.Errorf("deployment_id is required")
	}
	if strings.TrimSpace(req.SnapshotID) == "" {
		return nil, fmt.Errorf("snapshot_id is required")
	}

	resp, _, err := s.client.AddOns.StartAddonBackupExport(ctx, req.DeploymentID, &pipeops.AddonBackupExportRequest{
		SnapshotID: strings.TrimSpace(req.SnapshotID),
		Path:       strings.TrimSpace(req.Path),
		Format:     strings.TrimSpace(req.Format),
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getAddonBackupExportTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req getAddonBackupExportArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.DeploymentID) == "" {
		return nil, fmt.Errorf("deployment_id is required")
	}
	if strings.TrimSpace(req.ExportID) == "" {
		return nil, fmt.Errorf("export_id is required")
	}

	resp, _, err := s.client.AddOns.GetAddonBackupExport(ctx, req.DeploymentID, req.ExportID)
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

func (s *Server) getBillingInfoTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	workspaceUUID := ""
	balanceResp, err := s.requestBillingJSONWithWorkspaceFallback(ctx, http.MethodGet, "billing/balance", args, nil, &workspaceUUID)
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

	currentSubscriptionResp, err := s.requestBillingJSONWithWorkspaceFallback(ctx, http.MethodGet, "billing/subscriptions/current", args, nil, &workspaceUUID)
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

func (s *Server) getBalanceTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	resp, err := s.requestBillingJSONWithWorkspaceFallback(ctx, http.MethodGet, "billing/balance", args, nil, nil)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var balanceResp pipeops.BalanceResponse
	if err := json.Unmarshal(payload, &balanceResp); err != nil {
		return nil, err
	}
	return jsonResult(balanceResp)
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

func (s *Server) listSubscriptionsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	resp, err := s.requestBillingJSONWithWorkspaceFallback(ctx, http.MethodGet, "billing/subscriptions", args, nil, nil)
	if err != nil {
		return nil, err
	}
	return jsonResult(normalizeCollectionResponse(resp, "subscriptions"))
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
	resp, err := s.requestJSON(ctx, http.MethodGet, "billing/history", nil)
	if err != nil {
		return nil, err
	}
	return jsonResult(normalizeCollectionResponse(resp, "invoices"))
}

func (s *Server) listServiceAccountTokensTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, err := s.requestJSON(ctx, http.MethodGet, withWorkspaceUUIDQuery("api/v1/service-account-tokens", workspaceUUID), nil)
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

func (s *Server) listGitOpsApplicationsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req listGitOpsApplicationsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}

	resp, _, err := s.client.GitOps.List(ctx, &pipeops.GitOpsListOptions{
		Page:  req.Page,
		Limit: req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getGitOpsApplicationTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	applicationUUID, err := requiredString(args, "application_uuid")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.GitOps.Get(ctx, applicationUUID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) createGitOpsApplicationTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req createGitOpsApplicationArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.RepoURL) == "" {
		return nil, fmt.Errorf("repo_url is required")
	}

	resp, _, err := s.client.GitOps.Create(ctx, &pipeops.CreateGitOpsConfigRequest{
		Name:                strings.TrimSpace(req.Name),
		ProjectID:           req.ProjectID,
		EnvironmentID:       req.EnvironmentID,
		RepoURL:             strings.TrimSpace(req.RepoURL),
		Branch:              strings.TrimSpace(req.Branch),
		Path:                strings.TrimSpace(req.Path),
		TargetRevision:      strings.TrimSpace(req.TargetRevision),
		ManifestType:        strings.TrimSpace(req.ManifestType),
		SyncPolicy:          gitOpsAutomatedSyncPolicy(req.AutoSyncPrune, req.AutoSyncSelfHeal, req.AutoSyncAllowEmpty),
		HealthCheckEnabled:  req.HealthCheckEnabled,
		HealthCheckInterval: req.HealthCheckInterval,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateGitOpsApplicationTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req updateGitOpsApplicationArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ApplicationUUID) == "" {
		return nil, fmt.Errorf("application_uuid is required")
	}

	resp, _, err := s.client.GitOps.Update(ctx, strings.TrimSpace(req.ApplicationUUID), &pipeops.UpdateGitOpsConfigRequest{
		Name:                strings.TrimSpace(req.Name),
		Branch:              strings.TrimSpace(req.Branch),
		Path:                strings.TrimSpace(req.Path),
		TargetRevision:      strings.TrimSpace(req.TargetRevision),
		SyncPolicy:          gitOpsAutomatedSyncPolicy(req.AutoSyncPrune, req.AutoSyncSelfHeal, req.AutoSyncAllowEmpty),
		HealthCheckEnabled:  req.HealthCheckEnabled,
		HealthCheckInterval: req.HealthCheckInterval,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deleteGitOpsApplicationTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	applicationUUID, err := requiredString(args, "application_uuid")
	if err != nil {
		return nil, err
	}

	if _, err := s.client.GitOps.Delete(ctx, applicationUUID); err != nil {
		return nil, err
	}
	return textResult("GitOps application deleted successfully"), nil
}

func (s *Server) syncGitOpsApplicationTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req syncGitOpsApplicationArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ApplicationUUID) == "" {
		return nil, fmt.Errorf("application_uuid is required")
	}

	resp, _, err := s.client.GitOps.TriggerSync(ctx, strings.TrimSpace(req.ApplicationUUID), &pipeops.TriggerGitOpsSyncRequest{
		Revision: strings.TrimSpace(req.Revision),
		Prune:    req.Prune,
		DryRun:   req.DryRun,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getGitOpsSyncStatusTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	applicationUUID, err := requiredString(args, "application_uuid")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.GitOps.GetSyncStatus(ctx, applicationUUID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getGitOpsDiffTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	applicationUUID, err := requiredString(args, "application_uuid")
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.GitOps.GetDiff(ctx, applicationUUID)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getGitOpsHistoryTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req getGitOpsHistoryArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ApplicationUUID) == "" {
		return nil, fmt.Errorf("application_uuid is required")
	}

	resp, _, err := s.client.GitOps.GetHistory(ctx, strings.TrimSpace(req.ApplicationUUID), &pipeops.GitOpsListOptions{
		Page:  req.Page,
		Limit: req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) projectGroupWorkspaceOptions(ctx context.Context, args map[string]interface{}) (*pipeops.ProjectGroupWorkspaceOptions, error) {
	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, err
	}
	return &pipeops.ProjectGroupWorkspaceOptions{WorkspaceUUID: workspaceUUID}, nil
}

func (s *Server) listProjectGroupsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req listProjectGroupsArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}

	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.List(ctx, &pipeops.ProjectGroupListOptions{
		WorkspaceUUID: workspaceUUID,
		Limit:         req.Limit,
		Offset:        req.Offset,
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getProjectGroupTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	groupUUID, err := requiredString(args, "group_uuid")
	if err != nil {
		return nil, err
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.Get(ctx, groupUUID, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) createProjectGroupTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req createProjectGroupArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	body := &pipeops.CreateProjectGroupRequest{
		Name: strings.TrimSpace(req.Name),
	}
	if clusterUUID := strings.TrimSpace(req.DefaultClusterUUID); clusterUUID != "" {
		body.DefaultClusterUUID = &clusterUUID
	}
	if environmentUUID := strings.TrimSpace(req.DefaultEnvironmentUUID); environmentUUID != "" {
		body.DefaultEnvironmentUUID = &environmentUUID
	}

	resp, _, err := s.client.ProjectGroups.Create(ctx, body, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) updateProjectGroupTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req updateProjectGroupArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GroupUUID) == "" {
		return nil, fmt.Errorf("group_uuid is required")
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.Update(ctx, strings.TrimSpace(req.GroupUUID), &pipeops.UpdateProjectGroupRequest{
		Name:                   req.Name,
		DefaultClusterUUID:     req.DefaultClusterUUID,
		DefaultEnvironmentUUID: req.DefaultEnvironmentUUID,
	}, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) deleteProjectGroupTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	groupUUID, err := requiredString(args, "group_uuid")
	if err != nil {
		return nil, err
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	if _, err := s.client.ProjectGroups.Delete(ctx, groupUUID, opts); err != nil {
		return nil, err
	}
	return textResult("Project group deleted successfully"), nil
}

func (s *Server) attachProjectGroupMemberTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req attachProjectGroupMemberArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GroupUUID) == "" {
		return nil, fmt.Errorf("group_uuid is required")
	}
	if strings.TrimSpace(req.MemberType) == "" {
		return nil, fmt.Errorf("member_type is required")
	}
	if strings.TrimSpace(req.MemberUUID) == "" {
		return nil, fmt.Errorf("member_uuid is required")
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.AttachMember(ctx, strings.TrimSpace(req.GroupUUID), &pipeops.AttachProjectGroupMemberRequest{
		MemberType:     strings.TrimSpace(req.MemberType),
		MemberUUID:     strings.TrimSpace(req.MemberUUID),
		IncludeSession: req.IncludeSession,
		Move:           req.Move,
	}, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) detachProjectGroupMemberTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req detachProjectGroupMemberArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GroupUUID) == "" {
		return nil, fmt.Errorf("group_uuid is required")
	}
	if strings.TrimSpace(req.MemberType) == "" {
		return nil, fmt.Errorf("member_type is required")
	}
	if strings.TrimSpace(req.MemberUUID) == "" {
		return nil, fmt.Errorf("member_uuid is required")
	}

	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, err
	}

	if _, err := s.client.ProjectGroups.DetachMember(ctx, strings.TrimSpace(req.GroupUUID), strings.TrimSpace(req.MemberType), strings.TrimSpace(req.MemberUUID), &pipeops.ProjectGroupDetachOptions{
		WorkspaceUUID:  workspaceUUID,
		IncludeSession: req.IncludeSession,
	}); err != nil {
		return nil, err
	}
	return textResult("Project group member detached successfully"), nil
}

func (s *Server) getProjectGroupTopologyTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	groupUUID, err := requiredString(args, "group_uuid")
	if err != nil {
		return nil, err
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.GetTopology(ctx, groupUUID, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) getProjectGroupSharedEnvTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	groupUUID, err := requiredString(args, "group_uuid")
	if err != nil {
		return nil, err
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.GetSharedEnv(ctx, groupUUID, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) putProjectGroupSharedEnvTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req putProjectGroupSharedEnvArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GroupUUID) == "" {
		return nil, fmt.Errorf("group_uuid is required")
	}
	if req.Variables == nil {
		return nil, fmt.Errorf("variables is required")
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.PutSharedEnv(ctx, strings.TrimSpace(req.GroupUUID), &pipeops.UpsertProjectGroupSharedEnvRequest{
		Variables:      req.Variables,
		Inject:         req.Inject,
		Overwrite:      req.Overwrite,
		Redeploy:       req.Redeploy,
		KeepReferences: req.KeepReferences,
	}, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) injectProjectGroupSharedEnvTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req injectProjectGroupSharedEnvArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GroupUUID) == "" {
		return nil, fmt.Errorf("group_uuid is required")
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.InjectSharedEnv(ctx, strings.TrimSpace(req.GroupUUID), &pipeops.InjectProjectGroupSharedEnvRequest{
		Overwrite:      req.Overwrite,
		Redeploy:       req.Redeploy,
		MemberUUIDs:    req.MemberUUIDs,
		KeepReferences: req.KeepReferences,
	}, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) connectProjectGroupServicesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req connectProjectGroupServicesArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GroupUUID) == "" {
		return nil, fmt.Errorf("group_uuid is required")
	}
	if strings.TrimSpace(req.ConsumerType) == "" {
		return nil, fmt.Errorf("consumer_type is required")
	}
	if strings.TrimSpace(req.ConsumerUUID) == "" {
		return nil, fmt.Errorf("consumer_uuid is required")
	}
	if strings.TrimSpace(req.ProviderType) == "" {
		return nil, fmt.Errorf("provider_type is required")
	}
	if strings.TrimSpace(req.ProviderUUID) == "" {
		return nil, fmt.Errorf("provider_uuid is required")
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.ConnectServices(ctx, strings.TrimSpace(req.GroupUUID), &pipeops.ConnectProjectGroupServicesRequest{
		ConsumerType: strings.TrimSpace(req.ConsumerType),
		ConsumerUUID: strings.TrimSpace(req.ConsumerUUID),
		ProviderType: strings.TrimSpace(req.ProviderType),
		ProviderUUID: strings.TrimSpace(req.ProviderUUID),
		Overwrite:    req.Overwrite,
		VariableSet:  strings.TrimSpace(req.VariableSet),
	}, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) redeployProjectGroupAppsTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	groupUUID, err := requiredString(args, "group_uuid")
	if err != nil {
		return nil, err
	}

	opts, err := s.projectGroupWorkspaceOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.RedeployApps(ctx, groupUUID, opts)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) resolveProjectGroupMemberTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req resolveProjectGroupMemberArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.MemberType) == "" {
		return nil, fmt.Errorf("member_type is required")
	}
	if strings.TrimSpace(req.MemberUUID) == "" {
		return nil, fmt.Errorf("member_uuid is required")
	}

	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.ResolveMember(ctx, &pipeops.ProjectGroupResolveOptions{
		WorkspaceUUID: workspaceUUID,
		MemberType:    strings.TrimSpace(req.MemberType),
		MemberUUID:    strings.TrimSpace(req.MemberUUID),
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}

func (s *Server) listProjectGroupCandidatesTool(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var req listProjectGroupCandidatesArgs
	if err := decodeArguments(args, &req); err != nil {
		return nil, err
	}

	workspaceUUID, err := s.resolveDefaultWorkspaceUUID(ctx, args)
	if err != nil {
		return nil, err
	}

	resp, _, err := s.client.ProjectGroups.ListCandidates(ctx, &pipeops.ProjectGroupCandidatesOptions{
		WorkspaceUUID: workspaceUUID,
		GroupUUID:     strings.TrimSpace(req.GroupUUID),
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}
