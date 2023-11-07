import { EvmLog } from "../entities";
import { EvmBlockRepository, MetadataRepository } from "../repositories";
import { setTimeout } from "timers/promises";

const ID = "watch-evm-logs";

/**
 * PollEvmLogs is an action that watches for new blocks and extracts logs from them.
 */
export class PollEvmLogs {
  private readonly blockRepo: EvmBlockRepository;
  private readonly metadataRepo: MetadataRepository<PollEvmLogsMetadata>;
  private latestBlockHeight: bigint = 0n;
  private blockHeightCursor: bigint = 0n;
  private cfg: PollEvmLogsConfig;
  private started: boolean = false;

  constructor(
    blockRepo: EvmBlockRepository,
    metadataRepo: MetadataRepository<PollEvmLogsMetadata>,
    cfg: PollEvmLogsConfig
  ) {
    this.blockRepo = blockRepo;
    this.metadataRepo = metadataRepo;
    this.cfg = cfg;
  }

  public async start(handlers: ((logs: EvmLog[]) => Promise<void>)[]): Promise<void> {
    const metadata = await this.metadataRepo.get(ID);
    if (metadata) {
      this.blockHeightCursor = metadata.lastBlock;
    }

    this.started = true;
    this.watch(handlers);
  }

  private async watch(handlers: ((logs: EvmLog[]) => Promise<void>)[]): Promise<void> {
    while (this.started) {
      this.latestBlockHeight = await this.blockRepo.getBlockHeight(this.cfg.getCommitment());

      const range = this.getBlockRange(this.latestBlockHeight);
      if (this.cfg.hasFinished(range.fromBlock)) {
        // TODO: log
        await this.stop();
        continue;
      }

      const logs = await this.blockRepo.getFilteredLogs({
        fromBlock: range.fromBlock,
        toBlock: range.toBlock,
        addresses: this.cfg.addresses, // Works when sending multiple addresses, but not multiple topics.
        topics: [], // this.cfg.topics => will be applied by handlers
      });

      const blockNumbers = new Set(logs.map((log) => log.blockNumber));
      const blocks = await this.blockRepo.getBlocks(blockNumbers);
      logs.forEach((log) => {
        const block = blocks[log.blockHash];
        log.blockTime = block.timestamp;
      });

      // TODO: add error handling.
      await Promise.all(handlers.map((handler) => handler(logs)));

      await this.metadataRepo.save(ID, { lastBlock: range.toBlock });
      this.blockHeightCursor = range.toBlock;

      await setTimeout(this.cfg.interval ?? 1_000, undefined, { ref: false });
    }
  }

  /**
   * Get the block range to extract.
   * @param latestBlockHeight - the latest known height of the chain
   * @returns an always valid range, in the sense from is always <= to
   */
  private getBlockRange(latestBlockHeight: bigint): {
    fromBlock: bigint;
    toBlock: bigint;
  } {
    let fromBlock = this.blockHeightCursor + 1n;
    // fromBlock is configured and is greater than current block height, then we allow to skip blocks.
    if (this.cfg.fromBlock && this.cfg.fromBlock > this.blockHeightCursor) {
      fromBlock = this.cfg.fromBlock;
    }

    if (fromBlock > latestBlockHeight) {
      return { fromBlock: latestBlockHeight, toBlock: latestBlockHeight };
    }

    let toBlock = this.cfg.toBlock ?? this.blockHeightCursor + BigInt(this.cfg.getBlockBatchSize());
    // limit toBlock to obtained block height
    if (toBlock > fromBlock && toBlock > latestBlockHeight) {
      toBlock = latestBlockHeight;
    }

    return { fromBlock, toBlock };
  }

  public async stop(): Promise<void> {
    this.started = false;
  }

  // TODO: schedule getting latest block height in chain or use the value from poll to keep metrics updated
  // this.latestBlockHeight = await this.blockRepo.getBlockHeight(this.commitment);
}

export type PollEvmLogsMetadata = {
  lastBlock: bigint;
};

export class PollEvmLogsConfig {
  fromBlock?: bigint;
  toBlock?: bigint;
  blockBatchSize?: number;
  commitment?: string;
  interval?: number;
  addresses: string[] = [];
  topics: string[] = [];

  public getBlockBatchSize() {
    return this.blockBatchSize ?? 100;
  }

  public getCommitment() {
    return this.commitment ?? "latest";
  }

  public hasFinished(currentFromBlock: bigint) {
    return this.toBlock && currentFromBlock > this.toBlock;
  }

  static fromBlock(fromBlock: bigint) {
    const cfg = new PollEvmLogsConfig();
    cfg.fromBlock = fromBlock;
    return cfg;
  }
}
