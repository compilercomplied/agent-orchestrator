import * as pulumi from "@pulumi/pulumi";

export interface AppConfig {
  secrets: { [key: string]: pulumi.Output<string> };
  plainConfig: { [key: string]: string };
}

/**
 * Retrieves configuration values matching a specific prefix from the Pulumi configuration,
 * separating secret values from plain configuration values.
 * 
 * @param config The Pulumi Config object for the current project.
 * @param prefix The prefix to filter configuration keys (e.g., "AO_").
 * @param allConfig The full configuration map obtained from `pulumi.runtime.allConfig()`.
 * @param configNamespace The namespace/project name of the configuration (usually `config.name`).
 * @returns An object containing separate maps for secrets and plain config values.
 */
export function getAppConfig(
  config: pulumi.Config,
  prefix: string,
  allConfig: { [key: string]: string },
  configNamespace: string
): AppConfig {
  const secrets: { [key: string]: pulumi.Output<string> } = {};
  const plainConfig: { [key: string]: string } = {};

  for (const key of Object.keys(allConfig)) {
    // The keys in allConfig are fully qualified (e.g., "project:key")
    // We check if the key starts with the project name and the provided prefix
    if (key.startsWith(`${configNamespace}:${prefix}`)) {
      const varName = key.substring(`${configNamespace}:`.length);
      const secretValue = config.getSecret(varName);

      if (secretValue !== undefined) {
        secrets[varName] = secretValue;
      } else {
        plainConfig[varName] = config.require(varName);
      }

    }
  }

  return { secrets, plainConfig };
}
