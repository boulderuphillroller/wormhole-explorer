import { FileMetadataRepository } from "../FileMetadataRepository";
import { SnsEventRepository } from "../SnsEventRepository";
import { PromStatRepository } from "../PromStatRepository";
import { SNSClient } from "@aws-sdk/client-sns";
import { Config } from "../../config";
import { EvmJsonRPCBlockRepository } from "../EvmJsonRPCBlockRepository";
import { Web3SolanaSlotRepository } from "../Web3SolanaSlotRepository";
import { Connection } from "@solana/web3.js";
import { DynamicStrategy } from "./DynamicStrategy";
import { StaticStrategy } from "./StaticStrategy";

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
      new FileMetadataRepository(this.cfg),
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
      if (!this.cfg.platforms[chain]) throw new Error(`No config for chain ${chain}`);

      const repositories: DynamicStrategy[] = [
        new EvmJsonRPCBlockRepository(this.cfg),
        new Web3SolanaSlotRepository(this.cfg),
      ];

      repositories.forEach((repository) => {
        if (repository.apply(chain))
          dynamicRepositories.set(repository.getName(), repository.createInstance());
      });
    });

    return dynamicRepositories;
  }
}
