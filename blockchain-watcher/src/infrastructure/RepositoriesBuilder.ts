import { SNSClient, SNSClientConfig } from "@aws-sdk/client-sns";
import { Config } from "./config";
import {
  SnsEventRepository,
  EvmJsonRPCBlockRepository,
  FileMetadataRepository,
  PromStatRepository,
  StaticJobRepository,
  Web3SolanaSlotRepository,
  RateLimitedSolanaSlotRepository,
} from "./repositories";
import { JobRepository } from "../domain/repositories";
import { RepositoriesStrategy } from "./repositories/strategies/RepositoriesStrategy";

export class RepositoriesBuilder {
  private cfg: Config;
  private snsClient?: SNSClient;
  private repositories = new Map<string, any>();

  constructor(cfg: Config) {
    this.cfg = cfg;
    this.build();
  }

  private build(): void {
    this.snsClient = this.createSnsClient();

    const repositoryStrategy = new RepositoriesStrategy(this.snsClient, this.cfg);

    const staticRepositories = repositoryStrategy.executeStatic();
    const dynamicRepositories = repositoryStrategy.executeDynamic();

    staticRepositories.forEach((instance, name) => {
      this.repositories.set(name, instance);
    });
    dynamicRepositories.forEach((instance, name) => {
      this.repositories.set(name, instance);
    });

    this.repositories.set(
      "jobs",
      new StaticJobRepository(this.cfg, (chain: string) => this.getEvmBlockRepository(chain), {
        metadataRepo: this.getMetadataRepository(),
        statsRepo: this.getStatsRepository(),
        snsRepo: this.getSnsEventRepository(),
        solanaSlotRepo: this.getSolanaSlotRepository(),
      })
    );
  }

  public getEvmBlockRepository(chain: string): EvmJsonRPCBlockRepository {
    return this.getRepo(`${chain}-evmRepo`);
  }

  public getSnsEventRepository(): SnsEventRepository {
    return this.getRepo("sns");
  }

  public getMetadataRepository(): FileMetadataRepository {
    return this.getRepo("metadata");
  }

  public getStatsRepository(): PromStatRepository {
    return this.getRepo("metrics");
  }

  public getJobsRepository(): JobRepository {
    return this.getRepo("jobs");
  }

  public getSolanaSlotRepository(): Web3SolanaSlotRepository {
    return this.getRepo("solana-slotRepo");
  }

  private getRepo(name: string): any {
    const repo = this.repositories.get(name);
    if (!repo) throw new Error(`No repository ${name}`);

    return repo;
  }

  public close(): void {
    this.snsClient?.destroy();
  }

  private createSnsClient(): SNSClient {
    const snsCfg: SNSClientConfig = { region: this.cfg.sns.region };
    if (this.cfg.sns.credentials) {
      snsCfg.credentials = {
        accessKeyId: this.cfg.sns.credentials.accessKeyId,
        secretAccessKey: this.cfg.sns.credentials.secretAccessKey,
      };
      snsCfg.endpoint = this.cfg.sns.credentials.url;
    }

    return new SNSClient(snsCfg);
  }
}
