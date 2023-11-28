import {
  Commitment,
  Connection,
  PublicKey,
  VersionedTransactionResponse,
  SolanaJSONRPCError,
} from "@solana/web3.js";
import { solana } from "../../domain/entities";
import { SolanaSlotRepository } from "../../domain/repositories";
import { Fallible, SolanaFailure } from "../../domain/errors";
import { DynamicStrategy } from "./strategies/DynamicStrategy";
import { Config } from "../config";

const COMMITMENT_FINALIZED = "finalized";
const COMMITMENT_CONDIRMED = "confirmed";
const LEGACY_VERSION = "legacy";
const CHAIN = "solana";
const NAME = "solana-slotRepo";

export class Web3SolanaSlotRepository implements SolanaSlotRepository, DynamicStrategy {
  private connection: Connection;
  private cfg: Config;

  constructor(cfg: Config) {
    this.connection = new Connection(cfg.platforms[CHAIN].rpcs[0]);
    this.cfg = cfg;
  }

  apply(chain: string): boolean {
    return chain === CHAIN;
  }

  getName(): string {
    return NAME;
  }

  createInstance() {
    return new Web3SolanaSlotRepository(this.cfg);
  }

  getLatestSlot(commitment: string): Promise<number> {
    return this.connection.getSlot(commitment as Commitment);
  }

  getBlock(slot: number, finality?: string): Promise<Fallible<solana.Block, SolanaFailure>> {
    return this.connection
      .getBlock(slot, {
        maxSupportedTransactionVersion: 0,
        commitment:
          finality === COMMITMENT_FINALIZED || finality === COMMITMENT_CONDIRMED
            ? finality
            : undefined,
      })
      .then((block) => {
        if (block === null) {
          return Fallible.error<solana.Block, SolanaFailure>(
            new SolanaFailure(0, "Block not found")
          );
        }
        return Fallible.ok<solana.Block, SolanaFailure>({
          ...block,
          transactions: block.transactions.map((tx) => this.mapTx(tx, slot)),
        });
      })
      .catch((err) => {
        if (err instanceof SolanaJSONRPCError) {
          return Fallible.error(new SolanaFailure(err.code, err.message));
        }

        return Fallible.error(new SolanaFailure(0, err.message));
      });
  }

  getSignaturesForAddress(
    address: string,
    beforeSig: string,
    afterSig: string,
    limit: number
  ): Promise<solana.ConfirmedSignatureInfo[]> {
    return this.connection.getSignaturesForAddress(new PublicKey(address), {
      limit: limit,
      before: beforeSig,
      until: afterSig,
    });
  }

  async getTransactions(sigs: solana.ConfirmedSignatureInfo[]): Promise<solana.Transaction[]> {
    const txs = await this.connection.getTransactions(
      sigs.map((sig) => sig.signature),
      { maxSupportedTransactionVersion: 0 }
    );

    if (txs.length !== sigs.length) {
      throw new Error(`Expected ${sigs.length} transactions, but got ${txs.length} instead`);
    }

    return txs
      .filter((tx) => tx !== null)
      .map((tx, i) => {
        const message = tx?.transaction.message;
        const accountKeys =
          message?.version === LEGACY_VERSION
            ? message.accountKeys.map((key) => key.toBase58())
            : message?.staticAccountKeys.map((key) => key.toBase58());

        return {
          ...tx,
          transaction: {
            ...tx?.transaction,
            message: {
              ...tx?.transaction.message,
              accountKeys,
              compiledInstructions: message?.compiledInstructions ?? [],
            },
          },
        } as solana.Transaction;
      });
  }

  private mapTx(tx: Partial<VersionedTransactionResponse>, slot?: number): solana.Transaction {
    const message = tx?.transaction?.message;
    const accountKeys =
      message?.version === LEGACY_VERSION
        ? message.accountKeys.map((key) => key.toBase58())
        : message?.staticAccountKeys.map((key) => key.toBase58());

    return {
      ...tx,
      slot: tx.slot || slot,
      transaction: {
        ...tx.transaction,
        message: {
          ...tx?.transaction?.message,
          accountKeys,
          compiledInstructions: message?.compiledInstructions,
        },
      },
    } as solana.Transaction;
  }
}
