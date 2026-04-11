import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";
import { getAppConfig } from "./libraries/configuration";

const APP_ID = "agent-orchestrator";
const ENV_PREFIX = "AO_";
const config = new pulumi.Config(APP_ID);

const CONTROL_PLANE_NAMESPACE = "agents-control-plane";
const SERVICE_ACCOUNT_NAME = "agent-orchestrator-serviceaccount";

// Reference existing resources managed by another repository
const controlPlaneNamespace = k8s.core.v1.Namespace.get("control-plane-ns", "agents-control-plane");
const orchestratorServiceAccount = k8s.core.v1.ServiceAccount.get("orchestrator-sa", "agents-control-plane/agent-orchestrator-serviceaccount");

const appConfig = getAppConfig(
	config, ENV_PREFIX, pulumi.runtime.allConfig(), config.name
);

const configMap = new k8s.core.v1.ConfigMap(`${APP_ID}-config`, {
  metadata: { 
    namespace: controlPlaneNamespace.metadata.name,
  },
  data: appConfig.plainConfig,
});

// 5. Deployment
const appLabels = { app: APP_ID };

const deployment = new k8s.apps.v1.Deployment(`${APP_ID}-deployment`, {
  metadata: {
    namespace: controlPlaneNamespace.metadata.name,
    labels: appLabels,
  },
  spec: {
    replicas: 1,
    selector: { matchLabels: appLabels },
    template: {
      metadata: { labels: appLabels },
      spec: {
        serviceAccountName: orchestratorServiceAccount.metadata.name,
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
    namespace: controlPlaneNamespace.metadata.name,
    name: APP_ID,
  },
  spec: {
    selector: appLabels,
    ports: [{ port: 8080, targetPort: 8080 }],
    type: "ClusterIP",
  },
});

// Export the internal URL
export const internalUrl = pulumi.interpolate`http://${service.metadata.name}.${CONTROL_PLANE_NAMESPACE}.svc.cluster.local:8080`;
