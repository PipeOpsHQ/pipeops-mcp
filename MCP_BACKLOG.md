# MCP Expansion Backlog

This document tracks the next MCP batches after the current tool expansion.

## P0: Verify route alignment

- Confirm the SDK-backed routes used by `list_projects`, `get_project`, `deploy_project`, `list_servers`, `get_server`, `list_environments`, `get_current_user`, and `get_billing_info` match the live controller behavior.
- Normalize any remaining `server` vs `cluster` and `project` vs `project/fetch` differences before widening the MCP surface further.

## Completed: Resource lifecycle tools

- **Environments**: `create_environment`, `update_environment`, `delete_environment`, `set_environment_variables`
- **Workspaces**: `create_workspace`, `update_workspace`, `delete_workspace`, `set_workspace_billing_email`
- **Teams**: `create_team`, `update_team`, `invite_team_member`, `list_team_members`, `remove_team_member`, `update_team_member_role`

## Completed: Add-on deployment workflows

- `deploy_addon`
- `list_addon_deployments`
- `get_addon_deployment`
- `get_addon_deployment_session`
- `view_addon_deployment_configs`
- `add_addon_domain`
- `list_addon_categories`
- `get_my_addon_submissions`

## Completed: Billing workflows

- `list_billing_plans`
- `subscribe_to_plan`
- `cancel_subscription`
- `add_billing_card`
- `delete_billing_card`
- `create_workspace_checkout`
- `start_trial`
- `get_billing_portal_url`

These stay behind clear, explicit tool names because they change account state or billing context.

## Completed: Service token and cluster operations

- `create_service_account_token`
- `update_service_account_token`
- `revoke_service_account_token`
- `get_cluster_connection`
- `get_cluster_cost_allocation`

## Completed: BYOI, registries, and cloud discovery

- `deploy_project_from_image`
- `create_external_registry`
- `list_external_registries`
- `get_external_registry`
- `delete_external_registry`
- `list_external_registry_images`
- `list_external_registry_tags`
- `search_public_registry_images`
- `list_public_registry_tags`
- `list_cloud_provider_regions`
- `list_cloud_provider_instance_categories`
- `list_cloud_provider_instance_types`
- `list_cloud_provider_server_templates`

## SDK-first gaps

These controller APIs still appear important, but should be added to the SDK before exposing them through MCP:

- Event streaming and stats endpoints
- Cluster metrics and broader cost/usage drilldowns
- Additional cloud-provider sizing and cost-estimate endpoints

## Suggested implementation order

1. Finish route verification for the existing MCP tools.
2. Add SDK support for controller-only APIs, then expose them through MCP.
