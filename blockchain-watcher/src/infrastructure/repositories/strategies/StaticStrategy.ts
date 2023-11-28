export interface StaticStrategy {
  createInstance(): any;
  getName(): string;
  apply(): boolean;
}
