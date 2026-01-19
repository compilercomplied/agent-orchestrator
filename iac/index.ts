import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";
import { createCleanupCronJob } from "./cronjob";

const config = new pulumi.Config("agent-orchestrator");
const githubToken = config.requireSecret("GITHUB_TOKEN");
const anthropicApiKey = config.requireSecret("ANTHROPIC_API_KEY");
const kubeconfigContent = config.requireSecret("KUBECONFIG");

const nsControlPlane = new k8s.core.v1.Namespace("ns-control-plane", {
    metadata: { name: "agents-control-plane" },
});

const nsAgents = new k8s.core.v1.Namespace("ns-agents", {
    metadata: { name: "agents" },
});

createCleanupCronJob(nsAgents);

// The orchestrator needs to manage pods in the 'agents' namespace.
const serviceAccount = new k8s.core.v1.ServiceAccount("agent-orchestrator-serviceaccount", {
    metadata: { 
        namespace: nsControlPlane.metadata.name,
        name: "agent-orchestrator-serviceaccount"
    },
});

const role = new k8s.rbac.v1.Role("agents-manager-role", {
    metadata: { 
        namespace: nsAgents.metadata.name,
        name: "agents-manager"
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
        namespace: nsAgents.metadata.name,
        name: "agents-manager-binding"
    },
    subjects: [{
        kind: "ServiceAccount",
        name: serviceAccount.metadata.name,
        namespace: nsControlPlane.metadata.name,
    }],
    roleRef: {
        kind: "Role",
        name: role.metadata.name,
        apiGroup: "rbac.authorization.k8s.io",
    },
});

// 4. Secrets
const secret = new k8s.core.v1.Secret("orchestrator-secrets", {
    metadata: { namespace: nsControlPlane.metadata.name },
    stringData: {
        "KUBECONFIG": kubeconfigContent,
        "GITHUB_TOKEN": githubToken,
        "ANTHROPIC_API_KEY": anthropicApiKey,
    },
});

// 5. Deployment
const appLabels = { app: "agent-orchestrator" };

const deployment = new k8s.apps.v1.Deployment("orchestrator-dep", {
    metadata: { 
        namespace: nsControlPlane.metadata.name,
        labels: appLabels,
    },
    spec: {
        replicas: 1,
        selector: { matchLabels: appLabels },
        template: {
            metadata: { labels: appLabels },
            spec: {
                serviceAccountName: serviceAccount.metadata.name,
                containers: [{
                    name: "agent-orchestrator",
                    image: "ghcr.io/compilercomplied/agent-orchestrator:latest",
                    imagePullPolicy: "Always",
                    ports: [{ containerPort: 8080 }],
                    env: [
                        { name: "PORT", value: "8080" }, // App constant, but safe to set
                        { name: "KUBECONFIG", valueFrom: { secretKeyRef: { name: secret.metadata.name, key: "KUBECONFIG" } } },
                        { name: "GITHUB_TOKEN", valueFrom: { secretKeyRef: { name: secret.metadata.name, key: "GITHUB_TOKEN" } } },
                        { name: "ANTHROPIC_API_KEY", valueFrom: { secretKeyRef: { name: secret.metadata.name, key: "ANTHROPIC_API_KEY" } } },
                    ],
                }],
            },
        },
    },
});

// 6. Service
const service = new k8s.core.v1.Service("orchestrator-svc", {
    metadata: { 
        namespace: nsControlPlane.metadata.name,
        name: "agent-orchestrator", // Stable name
    },
    spec: {
        selector: appLabels,
        ports: [{ port: 8080, targetPort: 8080 }],
        type: "ClusterIP",
    },
});

// Export the internal URL
export const internalUrl = pulumi.interpolate`http://${service.metadata.name}.${nsControlPlane.metadata.name}.svc.cluster.local:8080`;
