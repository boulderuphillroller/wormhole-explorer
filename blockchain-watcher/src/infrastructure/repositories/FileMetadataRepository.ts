import { MetadataRepository } from "../../domain/repositories";
import fs from "fs";
import { StaticStrategy } from "./strategies/StaticStrategy";
import { Config } from "../config";

const UTF8 = "utf8";

export class FileMetadataRepository implements MetadataRepository<any>, StaticStrategy {
  private readonly dirPath: string;

  constructor(dirPath: string) {
    this.dirPath = dirPath;

    if (!fs.existsSync(this.dirPath)) {
      fs.mkdirSync(this.dirPath, { recursive: true });
    }
  }

  apply(): boolean {
    return this.dirPath != undefined;
  }

  getName(): string {
    return "metadata";
  }

  createInstance() {
    return new FileMetadataRepository(this.dirPath);
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
