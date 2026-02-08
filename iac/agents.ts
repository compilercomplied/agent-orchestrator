// Configuration of the agents Kubernetes resources.
// This module sets up the namespace and RBAC permissions for autonomous agent containers.

import * as k8s from "@pulumi/kubernetes";

export interface AgentsNamespaceResult {
  role: k8s.rbac.v1.Role;
  roleBinding: k8s.rbac.v1.RoleBinding;
}

/**
 * Creates the RBAC resources in the agents namespace that allow the orchestrator
 * to manage agent pods.
 *
 * @param agentsNamespaceName The name of the namespace where agents run.
 * @param controlPlaneNamespace The namespace where the orchestrator runs.
 * @param serviceAccount The service account used by the orchestrator.
 * @returns The created RBAC resources.
 */
export function createAgentsNamespace(
  agentsNamespaceName: string,
  controlPlaneNamespace: k8s.core.v1.Namespace,
  serviceAccount: k8s.core.v1.ServiceAccount
): AgentsNamespaceResult {
  const role = new k8s.rbac.v1.Role("agents-manager-role", {
    metadata: {
      namespace: agentsNamespaceName,
      name: "agents-manager",
    },
    rules: [
      {
        apiGroups: [""],
        resources: ["pods", "pods/log"],
        verbs: ["create", "list", "watch", "delete", "get"],
      },
      {
        apiGroups: [""],
        resources: ["secrets"],
        resourceNames: ["dev-environment-secrets"],
        verbs: ["get"],
      },
    ],
  });

  const roleBinding = new k8s.rbac.v1.RoleBinding("agents-manager-rb", {
    metadata: {
      namespace: agentsNamespaceName,
      name: "agents-manager-binding",
    },
    subjects: [
      {
        kind: "ServiceAccount",
        name: serviceAccount.metadata.name,
        namespace: controlPlaneNamespace.metadata.name,
      },
    ],
    roleRef: {
      kind: "Role",
      name: role.metadata.name,
      apiGroup: "rbac.authorization.k8s.io",
    },
  });

  return { role, roleBinding };
}
