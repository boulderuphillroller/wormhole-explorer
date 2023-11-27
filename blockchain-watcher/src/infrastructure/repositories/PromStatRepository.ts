import prometheus from "prom-client";
import { StatRepository } from "../../domain/repositories";
import { RepositoryStrategy } from "./strategies/RepositoryStrategy";

export class PromStatRepository implements StatRepository, RepositoryStrategy {
  private counters: Map<string, prometheus.Counter<string>> = new Map();
  private gauges: Map<string, prometheus.Gauge<string>> = new Map();
  private readonly registry: prometheus.Registry;

  constructor(registry?: prometheus.Registry) {
    this.registry = registry ?? new prometheus.Registry();
  }

  apply(): boolean {
    return true;
  }

  getName(): string {
    return "metrics";
  }

  createInstance() {
    return new PromStatRepository();
  }

  public report() {
    return this.registry.metrics();
  }

  public count(id: string, labels: Record<string, any>): void {
    const counter = this.getCounter(id, labels);
    counter.inc(labels);
  }

  public measure(id: string, value: bigint, labels: Record<string, any>): void {
    const gauge = this.getGauge(id, labels);
    gauge.set(labels, Number(value));
  }

  private getCounter(id: string, labels: Record<string, any>): prometheus.Counter {
    this.counters.get(id) ??
      this.counters.set(
        id,
        new prometheus.Counter({
          name: id,
          help: id,
          registers: [this.registry],
          labelNames: Object.keys(labels),
        })
      );

    return this.counters.get(id) as prometheus.Counter<string>;
  }

  private getGauge(id: string, labels: Record<string, any>): prometheus.Gauge {
    this.gauges.get(id) ??
      this.gauges.set(
        id,
        new prometheus.Gauge({
          name: id,
          help: id,
          registers: [this.registry],
          labelNames: Object.keys(labels),
        })
      );

    return this.gauges.get(id) as prometheus.Gauge;
  }
}
