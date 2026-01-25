// Configuration of the agents Kubernetes resources.
// This module sets up the namespace and RBAC permissions for autonomous agent containers.

import * as k8s from "@pulumi/kubernetes";

export interface AgentsNamespaceResult {
  namespace: k8s.core.v1.Namespace;
  role: k8s.rbac.v1.Role;
  roleBinding: k8s.rbac.v1.RoleBinding;
}

/**
 * Creates the agents namespace and RBAC resources that allow the orchestrator
 * to manage agent pods.
 *
 * @param controlPlaneNamespace The namespace where the orchestrator runs.
 * @param serviceAccount The service account used by the orchestrator.
 * @returns The created namespace and RBAC resources.
 */
export function createAgentsNamespace(
  controlPlaneNamespace: k8s.core.v1.Namespace,
  serviceAccount: k8s.core.v1.ServiceAccount
): AgentsNamespaceResult {
  const namespace = new k8s.core.v1.Namespace("ns-agents", {
    metadata: { name: "agents" },
  });

  const role = new k8s.rbac.v1.Role("agents-manager-role", {
    metadata: {
      namespace: namespace.metadata.name,
      name: "agents-manager",
    },
    rules: [
      {
        apiGroups: [""],
        resources: ["pods", "pods/log"],
        verbs: ["create", "list", "watch", "delete", "get"],
      },
    ],
  });

  const roleBinding = new k8s.rbac.v1.RoleBinding("agents-manager-rb", {
    metadata: {
      namespace: namespace.metadata.name,
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

  return { namespace, role, roleBinding };
}
