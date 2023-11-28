import { describe, expect, it } from "@jest/globals";
import { RepositoriesBuilder } from "../../../src/infrastructure/RepositoriesBuilder";
import { configMock } from "../../mock/configMock";
import {
  EvmJsonRPCBlockRepository,
  FileMetadataRepository,
  PromStatRepository,
  RateLimitedSolanaSlotRepository,
  SnsEventRepository,
} from "../../../src/infrastructure/repositories";

describe("RepositoriesBuilder", () => {
  it("should be error because dose not have any chain", async () => {
    try {
      // When
      new RepositoriesBuilder(configMock([]));
    } catch (e) {
      // Then
      expect(e).toBeInstanceOf(Error);
    }
  });

  it("should be error because dose not support test chain", async () => {
    try {
      // When
      new RepositoriesBuilder(configMock(["test"]));
    } catch (e) {
      // Then
      expect(e).toBeInstanceOf(Error);
    }
  });

  it("should be return all repositories instances", async () => {
    // When
    const repos = new RepositoriesBuilder(configMock(["ethereum", "solana"]));
    // Then
    const job = repos.getJobsRepository();
    expect(job).toBeTruthy();

    expect(repos.getEvmBlockRepository("ethereum")).toBeInstanceOf(EvmJsonRPCBlockRepository);
    expect(repos.getMetadataRepository()).toBeInstanceOf(FileMetadataRepository);
    expect(repos.getSnsEventRepository()).toBeInstanceOf(SnsEventRepository);
    expect(repos.getStatsRepository()).toBeInstanceOf(PromStatRepository);
    expect(repos.getSolanaSlotRepository()).toBeInstanceOf(RateLimitedSolanaSlotRepository);
  });
});
