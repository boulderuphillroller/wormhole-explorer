import { MetadataRepository } from "../../domain/repositories";
import fs from "fs";
import { RepositoryStrategy } from "./strategies/RepositoryStrategy";
import { Config } from "../config";

const UTF8 = "utf8";

export class FileMetadataRepository implements MetadataRepository<any>, RepositoryStrategy {
  private readonly dirPath: string;
  private readonly cfg: Config;

  constructor(cfg: Config) {
    this.cfg = cfg;
    this.dirPath = this.cfg!.metadata!.dir;

    if (!fs.existsSync(this.dirPath)) {
      fs.mkdirSync(this.dirPath, { recursive: true });
    }
  }

  apply(): boolean {
    return this.cfg.metadata?.dir != undefined;
  }

  getName(): string {
    return "metadata";
  }

  createInstance() {
    return new FileMetadataRepository(this.cfg);
  }

  async get(id: string): Promise<any> {
    const filePath = `${this.dirPath}/${id}.json`;

    return fs.promises
      .readFile(filePath, UTF8)
      .then(JSON.parse)
      .catch((err) => null);
  }

  async save(id: string, metadata: any): Promise<void> {
    const filePath = `${this.dirPath}/${id}.json`;
    return fs.promises.writeFile(filePath, JSON.stringify(metadata), UTF8);
  }
}
