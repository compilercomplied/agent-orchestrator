import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";
import { createCleanupCronJob } from "./cronjob";
import { createAgentsNamespace } from "./agents";
import { getAppConfig } from "./libraries/configuration";

const APP_ID = "agent-orchestrator";
const ENV_PREFIX = "AO_";
const config = new pulumi.Config(APP_ID);

const AGENTS_NAMESPACE = "agents";

const nsControlPlane = new k8s.core.v1.Namespace("ns-agents-control-plane", {
  metadata: { name: "agents-control-plane" },
});

// The orchestrator needs to manage pods in the 'agents' namespace.
const serviceAccount = new k8s.core.v1.ServiceAccount(`${APP_ID}-serviceaccount`, {
  metadata: {
    namespace: nsControlPlane.metadata.name,
    name: `${APP_ID}-serviceaccount`
  },
});

createAgentsNamespace(AGENTS_NAMESPACE, nsControlPlane, serviceAccount);
createCleanupCronJob(AGENTS_NAMESPACE);

const appConfig = getAppConfig(
	config, ENV_PREFIX, pulumi.runtime.allConfig(), config.name
);

const configMap = new k8s.core.v1.ConfigMap(`${APP_ID}-config`, {
  metadata: { namespace: nsControlPlane.metadata.name },
  data: appConfig.plainConfig,
});

// 5. Deployment
const appLabels = { app: APP_ID };

const deployment = new k8s.apps.v1.Deployment(`${APP_ID}-deployment`, {
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
          name: APP_ID,
          image: `ghcr.io/compilercomplied/${APP_ID}:latest`,
          imagePullPolicy: "Always",
          ports: [{ containerPort: 8080 }],
          envFrom: [
            { configMapRef: { name: configMap.metadata.name } },
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
    name: APP_ID,
  },
  spec: {
    selector: appLabels,
    ports: [{ port: 8080, targetPort: 8080 }],
    type: "ClusterIP",
  },
});

// Export the internal URL
export const internalUrl = pulumi.interpolate`http://${service.metadata.name}.${nsControlPlane.metadata.name}.svc.cluster.local:8080`;
