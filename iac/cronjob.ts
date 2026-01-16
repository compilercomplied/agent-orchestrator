import * as k8s from "@pulumi/kubernetes";

// CronJob to delete Succeeded and Failed pods periodically
export function createCleanupCronJob(namespace: k8s.core.v1.Namespace) {
    // Service Account for the cleaner
    const cleanerServiceAccount = new k8s.core.v1.ServiceAccount("agent-cleaner-serviceaccount", {
        metadata: {
            namespace: namespace.metadata.name,
            name: "agent-cleaner-serviceaccount",
        },
    });

    const cleanerRole = new k8s.rbac.v1.Role("agent-cleaner-role", {
        metadata: {
            namespace: namespace.metadata.name,
            name: "agent-cleaner-role",
        },
        rules: [{
            apiGroups: [""],
            resources: ["pods"],
            verbs: ["list", "delete"],
        }],
    });

    const cleanerRoleBinding = new k8s.rbac.v1.RoleBinding("agent-cleaner-rolebinder", {
        metadata: {
            namespace: namespace.metadata.name,
            name: "agent-cleaner-rolebinder",
        },
        subjects: [{
            kind: "ServiceAccount",
            name: cleanerServiceAccount.metadata.name,
            namespace: namespace.metadata.name,
        }],
        roleRef: {
            kind: "Role",
            name: cleanerRole.metadata.name,
            apiGroup: "rbac.authorization.k8s.io",
        },
    });

    const cleanupCronJob = new k8s.batch.v1.CronJob("agent-cleanup-job", {
        metadata: {
            namespace: namespace.metadata.name,
            name: "agent-cleanup",
        },
        spec: {
            schedule: "0 0 * * *", // Run once a day at midnight
            jobTemplate: {
                spec: {
                    template: {
                        spec: {
                            serviceAccountName: cleanerServiceAccount.metadata.name,
                            containers: [{
                                name: "kubectl",
                                image: "bitnami/kubectl:latest",
                                command: ["/bin/sh", "-c"],
                                args: ["kubectl delete pods --field-selector=status.phase=Succeeded --ignore-not-found=true && kubectl delete pods --field-selector=status.phase=Failed --ignore-not-found=true"],
                            }],
                            restartPolicy: "OnFailure",
                        },
                    },
                },
            },
        },
    });

    return { cleanerServiceAccount, cleanerRole, cleanerRoleBinding, cleanupCronJob };
}
