import { FileMetadataRepository } from "../FileMetadataRepository";
import { SnsEventRepository } from "../SnsEventRepository";
import { PromStatRepository } from "../PromStatRepository";
import { SNSClient } from "@aws-sdk/client-sns";
import { Config } from "../../config";
import { EvmJsonRPCBlockRepository } from "../EvmJsonRPCBlockRepository";
import { Connection } from "@solana/web3.js";
import { DynamicStrategy } from "./DynamicStrategy";
import { StaticStrategy } from "./StaticStrategy";
import { RateLimitedSolanaSlotRepository, Web3SolanaSlotRepository } from "..";

export class RepositoriesStrategy {
  private snsClient?: SNSClient;
  private cfg: Config;

  constructor(snsClient: SNSClient, cfg: Config) {
    this.snsClient = snsClient;
    this.cfg = cfg;
  }

  executeStatic(): Map<string, any> {
    let staticRepositories = new Map();

    const repositories: StaticStrategy[] = [
      new SnsEventRepository(this.snsClient!, this.cfg),
      new FileMetadataRepository(this.cfg.metadata!.dir!),
      new PromStatRepository(),
    ];

    repositories.forEach((repository) => {
      if (repository.apply())
        staticRepositories.set(repository.getName(), repository.createInstance());
    });

    return staticRepositories;
  }

  executeDynamic(): Map<string, any> {
    let dynamicRepositories = new Map();

    this.cfg.supportedChains.forEach((chain) => {
      const platform = this.cfg.platforms[chain];
      if (!platform) throw new Error(`No config for chain ${chain}`);

      const repositories = [
        new EvmJsonRPCBlockRepository(this.cfg),
        new RateLimitedSolanaSlotRepository(
          new Web3SolanaSlotRepository(
            new Connection(platform.rpcs[0], { disableRetryOnRateLimit: true })
          ),
          platform.rateLimit
        )
      ];

      repositories.forEach((repository) => {
        if (repository.apply(chain))
          dynamicRepositories.set(repository.getName(), repository.createInstance());
      });
    });

    return dynamicRepositories;
  }
}
