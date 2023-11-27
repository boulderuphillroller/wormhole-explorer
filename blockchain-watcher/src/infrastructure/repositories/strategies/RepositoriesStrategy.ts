import { FileMetadataRepository } from "../FileMetadataRepository";
import { SnsEventRepository } from "../SnsEventRepository";
import { PromStatRepository } from "../PromStatRepository";
import { SNSClient } from "@aws-sdk/client-sns";
import { Config } from "../../config";
import { EvmJsonRPCBlockRepository } from "../EvmJsonRPCBlockRepository";
import { Web3SolanaSlotRepository } from "../Web3SolanaSlotRepository";
import { Connection } from "@solana/web3.js";

export class RepositoriesStrategy {
  private repositoriesMap = new Map<string, any>();
  private snsClient?: SNSClient;
  private cfg: Config;

  constructor(snsClient: SNSClient, cfg: Config) {
    this.snsClient = snsClient;
    this.cfg = cfg;
  }

  executeStatic(): Map<string, any> {
    const repositories = [
      new SnsEventRepository(this.snsClient!, this.cfg),
      new FileMetadataRepository(this.cfg),
      new PromStatRepository(),
    ];

    repositories.forEach((repository) => {
      if (repository.apply())
        this.repositoriesMap.set(repository.getName(), repository.createInstance());
    });

    return this.repositoriesMap;
  }

  executeChain(chain: string): Map<string, any> {
    const config = this.cfg.platforms[chain];

    const repositories = [
      new EvmJsonRPCBlockRepository(this.cfg, chain),
      new Web3SolanaSlotRepository(new Connection(config.rpcs[0]), this.cfg, chain),
    ];

    repositories.forEach((repository) => {
      if (repository.apply())
        this.repositoriesMap.set(repository.getName(), repository.createInstance());
    });

    return this.repositoriesMap;
  }
}
