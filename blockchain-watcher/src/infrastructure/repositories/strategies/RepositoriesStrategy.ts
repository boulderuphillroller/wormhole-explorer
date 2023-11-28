import { FileMetadataRepository } from "../FileMetadataRepository";
import { SnsEventRepository } from "../SnsEventRepository";
import { PromStatRepository } from "../PromStatRepository";
import { SNSClient } from "@aws-sdk/client-sns";
import { Config } from "../../config";
import { EvmJsonRPCBlockRepository } from "../EvmJsonRPCBlockRepository";
import { Web3SolanaSlotRepository } from "../Web3SolanaSlotRepository";
import { Connection } from "@solana/web3.js";

export class RepositoriesStrategy {
  private snsClient?: SNSClient;
  private cfg: Config;

  constructor(snsClient: SNSClient, cfg: Config) {
    this.snsClient = snsClient;
    this.cfg = cfg;
  }

  executeStatic(): Map<string, any> {
    let staticRepositories = new Map();

    const repositories = [
      new SnsEventRepository(this.snsClient!, this.cfg),
      new FileMetadataRepository(this.cfg, this.cfg.metadata?.dir!),
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
      const config = this.cfg.platforms[chain];

      const repositories = [
        new EvmJsonRPCBlockRepository(this.cfg, chain),
        new Web3SolanaSlotRepository(new Connection(config.rpcs[0]), this.cfg, chain),
      ];

      if (!this.cfg.platforms[chain]) throw new Error(`No config for chain ${chain}`);

      repositories.forEach((repository) => {
        if (repository.apply())
          dynamicRepositories.set(repository.getName(), repository.createInstance());
      });
    });

    return dynamicRepositories;
  }
}
