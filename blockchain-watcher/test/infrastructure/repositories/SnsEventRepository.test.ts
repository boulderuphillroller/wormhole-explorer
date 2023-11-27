import { describe, expect, it, jest } from "@jest/globals";
import { SnsEventRepository, SnsConfig } from "../../../src/infrastructure/repositories";
import { SNSClient } from "@aws-sdk/client-sns";
import { configMock } from "../../mock/configMock";

const cfg = configMock();

let repo: SnsEventRepository;
let snsClient: SNSClient;

describe("SnsEventRepository", () => {
  it("should be apply SnsEventRepository", async () => {
    // Given
    givenSnsEventRepository();

    // When
    const result = await repo.apply();

    // Then
    expect(result).toBe(true);
  });

  it("should be get name metadata", async () => {
    // Given
    givenSnsEventRepository();

    // When
    const result = await repo.getName();

    // Then
    expect(result).toBe("sns");
  });

  it("should be create instance", async () => {
    // Given
    givenSnsEventRepository();

    // When
    const result = await repo.createInstance();

    // Then
    expect(result).toBeInstanceOf(SnsEventRepository);
  });

  it("should not call sns client when no events given", async () => {
    // Given
    givenSnsEventRepository();

    // When
    const result = await repo.publish([]);

    // Then
    expect(result).toEqual({ status: "success" });
    expect(snsClient.send).not.toHaveBeenCalled();
  });

  it("should publish", async () => {
    // Given
    givenSnsEventRepository();

    // When
    const result = await repo.publish([
      {
        chainId: 1,
        address: "0x123456",
        txHash: "0x123",
        blockHeight: 123n,
        blockTime: 0,
        name: "LogMessagePublished",
        attributes: {},
      },
    ]);

    // Then
    expect(result).toEqual({ status: "success" });
    expect(snsClient.send).toHaveBeenCalledTimes(1);
  });
});

const givenSnsEventRepository = () => {
  snsClient = {
    send: jest.fn().mockReturnThis(),
  } as unknown as SNSClient;
  repo = new SnsEventRepository(snsClient, cfg);
};
