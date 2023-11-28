import { setTimeout } from "timers/promises";
import winston from "winston";
import { Handler } from "../entities";
import { StatRepository } from "../repositories";

export abstract class RunPollingJob {
  private interval: number;
  private id: string;
  private statRepo?: StatRepository;
  private running: boolean = false;
  protected abstract logger: winston.Logger;
  protected abstract preHook(): Promise<void>;
  protected abstract hasNext(): Promise<boolean>;
  protected abstract get(): Promise<any[]>;
  protected abstract persist(): Promise<void>;

  constructor(interval: number, id: string, statRepo?: StatRepository) {
    this.interval = interval;
    this.id = id;
    this.running = true;
    this.statRepo = statRepo;
  }

  public async run(handlers: Handler[]): Promise<void> {
    this.logger.info("Starting polling job");
    await this.preHook();
    while (this.running) {
      if (!(await this.hasNext())) {
        this.logger.info("Finished processing");
        await this.stop();
        break;
      }

      let items: any[];

      try {
        items = await this.get();
        await Promise.all(handlers.map((handler) => handler(items)));
      } catch (e: Error | any) {
        this.logger.error("Error processing items", e, e.stack);
        this.statRepo?.count("job_runs_total", { id: this.id, status: "error" });
        await setTimeout(this.interval);
        continue;
      }

      await this.persist();
      this.statRepo?.count("job_runs_total", { id: this.id, status: "success" });
      await setTimeout(this.interval);
    }
  }

  public async stop(): Promise<void> {
    this.running = false;
    this.statRepo?.count("job_runs_stopped", { id: this.id });
  }
}
