export interface RepositoryStrategy {
  apply(): boolean;
  getName(): string;
  createInstance(): any;
}
