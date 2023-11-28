export interface DynamicStrategy {
  createInstance(): any;
  getName(): string;
  apply(chain: string): boolean;
}
