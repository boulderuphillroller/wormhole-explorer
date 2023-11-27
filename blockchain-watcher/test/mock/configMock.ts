import { SnsConfig } from "../../src/infrastructure/repositories";
import { Config, PlatformConfig } from "../../src/infrastructure/config";

export const configMock = (): Config => {
  const platformRecord: Record<string, PlatformConfig> = {
    latest: {
      name: "test",
      network: "ETH",
      chainId: 1222341,
      rpcs: ["http://localhost"],
      timeout: 100,
    },
    ethereum: {
      name: "test",
      network: "ETH",
      chainId: 1222341,
      rpcs: ["http://localhost"],
      timeout: 100,
    },
  };

  const snsConfig: SnsConfig = {
    region: "string",
    topicArn: "string",
    subject: "string",
    groupId: "string",
    credentials: {
      accessKeyId: "string",
      secretAccessKey: "string",
      url: "string",
    },
  };

  const cfg: Config = {
    environment: "mainnet",
    port: 999,
    logLevel: "info",
    dryRun: false,
    sns: snsConfig,
    metadata: {
      dir: "./metadata-repo/jobs",
    },
    jobs: {
      dir: "./metadata-repo/jobs",
    },
    platforms: platformRecord,
    supportedChains: [],
  };

  return cfg;
};
