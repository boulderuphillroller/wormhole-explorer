import { LogFoundEvent } from "../../domain/entities";
import crypto from "node:crypto";
import {
  SNSClient,
  PublishBatchCommand,
  PublishBatchCommandInput,
  PublishBatchRequestEntry,
} from "@aws-sdk/client-sns";
import winston from "../log";
import { SnsEvent } from "../events/SnsEvent";
import { StaticStrategy } from "./strategies/StaticStrategy";
import { Config } from "../config";
import { SnsRepository } from "../../domain/repositories";

const BLOCKCHAIN_WATCHER = "blockchain-watcher";
const CLASS_NAME = "SnsEventRepository";
const FULFILLED_STATUS = "fulfilled";
const SUCCESS_STATUS = "success";
const ERROR_STATUS = "error";
const CHUNK_SIZE = 10;

export class SnsEventRepository implements SnsRepository, StaticStrategy {
  private logger: winston.Logger;
  private client: SNSClient;
  private snsCfg: SnsConfig;
  private cfg: Config;

  constructor(snsClient: SNSClient, cfg: Config) {
    this.client = snsClient;
    this.snsCfg = cfg.sns;
    this.cfg = cfg;
    this.logger = winston.child({ module: CLASS_NAME });
    this.logger.info(`Created for topic ${this.snsCfg.topicArn}`);
  }

  apply(): boolean {
    return true;
  }

  getName(): string {
    return "sns";
  }

  createInstance(): SnsEventRepository {
    return this;
  }

  async publish(events: LogFoundEvent<any>[]): Promise<SnsPublishResult> {
    if (!events.length) {
      this.logger.warn("No events to publish, continuing...");
      return {
        status: SUCCESS_STATUS,
      };
    }

    const batches: PublishBatchCommandInput[] = [];

    const inputs: PublishBatchRequestEntry[] = events
      .map(SnsEvent.fromLogFoundEvent)
      .map((event) => ({
        Id: crypto.randomUUID(),
        Subject: this.snsCfg.subject ?? BLOCKCHAIN_WATCHER,
        Message: JSON.stringify(event),
        MessageGroupId: this.snsCfg.groupId ?? BLOCKCHAIN_WATCHER,
        MessageDeduplicationId: event.trackId,
      }));

    // PublishBatchCommand: only supports max 10 items per batch
    for (let i = 0; i < inputs.length; i += CHUNK_SIZE) {
      const batch: PublishBatchCommandInput = {
        TopicArn: this.snsCfg.topicArn,
        PublishBatchRequestEntries: inputs.slice(i, i + CHUNK_SIZE),
      };

      batches.push(batch);
    }

    try {
      const promises = [];
      const errors = [];

      for (const batch of batches) {
        const command = new PublishBatchCommand(batch);
        promises.push(this.client.send(command));
      }

      const results = await Promise.allSettled(promises);

      for (const result of results) {
        if (result.status !== FULFILLED_STATUS) {
          this.logger.error(result.reason);
          errors.push(result.reason);
        }
      }

      if (errors.length > 0) {
        return {
          status: ERROR_STATUS,
          reasons: errors,
        };
      }
    } catch (error: unknown) {
      this.logger.error(error);

      return {
        status: ERROR_STATUS,
      };
    }

    return {
      status: SUCCESS_STATUS,
    };
  }

  async asTarget(): Promise<(events: LogFoundEvent<any>[]) => Promise<void>> {
    return async (events: LogFoundEvent<any>[]) => {
      const result = await this.publish(events);

      if (result.status === ERROR_STATUS) {
        this.logger.error(`Error publishing events to SNS: ${result.reason ?? result.reasons}`);
        throw new Error(`Error publishing events to SNS: ${result.reason}`);
      }

      this.logger.info(`Published ${events.length} events to SNS`);
    };
  }
}

export type SnsConfig = {
  region: string;
  topicArn: string;
  subject?: string;
  groupId: string;
  credentials?: {
    accessKeyId: string;
    secretAccessKey: string;
    url: string;
  };
};

export type SnsPublishResult = {
  status: "success" | "error";
  reason?: string;
  reasons?: string[];
};
