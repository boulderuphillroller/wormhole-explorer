import { expect, describe, it } from "@jest/globals";
import { solana } from "../../../src/domain/entities";
import { Web3SolanaSlotRepository } from "../../../src/infrastructure/repositories";

describe("Web3SolanaSlotRepository", () => {
  const chain = "solana";

  describe("strategy", () => {
    it("should be apply Web3SolanaSlotRepository", async () => {
      // Given
      const connectionMock = {};
      const repo = new Web3SolanaSlotRepository(connectionMock as any);

      // When
      const result = await repo.apply(chain);

      // Then
      expect(result).toBe(true);
    });

    it("should be get name metadata", async () => {
      // Given
      const connectionMock = {};
      const repo = new Web3SolanaSlotRepository(connectionMock as any);

      // When
      const result = await repo.getName();

      // Then
      expect(result).toBe("solana-slotRepo");
    });

    it("should be create instance", async () => {
      // Given
      const connectionMock = {};
      const repo = new Web3SolanaSlotRepository(connectionMock as any);

      // When
      const result = await repo.createInstance();

      // Then
      expect(result).toBeInstanceOf(Web3SolanaSlotRepository);
    });
  });

  describe("getLatestSlot", () => {
    it("should return the latest slot number", async () => {
      // Given
      const connectionMock = {
        getSlot: () => Promise.resolve(100),
      };
      const repository = new Web3SolanaSlotRepository(connectionMock as any);

      // When
      const latestSlot = await repository.getLatestSlot("finalized");

      // Then
      expect(latestSlot).toBe(100);
    });
  });

  describe("getBlock", () => {
    it("should return a block for a given slot number", async () => {
      // Given
      const expected = {
        blockTime: 100,
        transactions: [],
      };
      const connectionMock = {
        getBlock: (slot: number) => Promise.resolve(expected),
      };
      const repo = new Web3SolanaSlotRepository(connectionMock as any);

      // When
      const block = (await repo.getBlock(100)).getValue();

      // Then
      expect(block.blockTime).toBe(expected.blockTime);
      expect(block.transactions).toHaveLength(expected.transactions.length);
    });
  });

  describe("getSignaturesForAddress", () => {
    it("should return confirmed signature info for a given address", async () => {
      // Given
      const expected = [
        {
          signature: "signature1",
          slot: 100,
        },
        {
          signature: "signature2",
          slot: 200,
        },
      ];
      const connectionMock = {
        getSignaturesForAddress: () => Promise.resolve(expected),
      };
      const repo = new Web3SolanaSlotRepository(connectionMock as any);

      // When
      const signatures = await repo.getSignaturesForAddress(
        "BTcueXFisZiqE49Ne2xTZjHV9bT5paVZhpKc1k4L3n1c",
        "before",
        "after",
        10
      );

      // Then
      expect(signatures).toBe(expected);
    });
  });

  describe("getTransactions", () => {
    it("should return transactions for a given array of confirmed signature info", async () => {
      // Given
      const expected = [
        {
          signature: "signature1",
          slot: 100,
          transaction: {
            message: {
              version: "legacy",
              accountKeys: [],
              instructions: [],
              compiledInstructions: [],
            },
          },
        },
      ];
      const connectionMock = {
        getTransactions: (sigs: solana.ConfirmedSignatureInfo[]) => Promise.resolve(expected),
      };
      const repo = new Web3SolanaSlotRepository(connectionMock as any);

      // When
      const transactions = await repo.getTransactions([
        {
          signature: "signature1",
        },
      ]);

      // Then
      expect(transactions).toStrictEqual(expected);
    });
  });
});
