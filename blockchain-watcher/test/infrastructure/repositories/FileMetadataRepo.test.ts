import { describe, expect, it, beforeEach, afterEach } from "@jest/globals";
import fs from "fs";
import { FileMetadataRepository } from "../../../src/infrastructure/repositories";
import { configMock } from "../../mock/configMock";

describe("FileMetadataRepository", () => {
  const dirPath = "./metadata-repo";
  const cfg = configMock();
  const repo = new FileMetadataRepository(dirPath);

  beforeEach(() => {
    if (!fs.existsSync(dirPath)) {
      fs.mkdirSync(dirPath);
    }
  });

  afterEach(() => {
    fs.rm(dirPath, () => {});
  });

  describe("FileMetadataRepository", () => {
    it("should be apply FileMetadataRepository", async () => {
      // When
      const result = await repo.apply();

      // Then
      expect(result).toBe(true);
    });

    it("should be get name metadata", async () => {
      // When
      const result = await repo.getName();

      // Then
      expect(result).toBe("metadata");
    });

    it("should be create instance", async () => {
      // When
      const result = await repo.createInstance();

      // Then
      expect(result).toBeInstanceOf(FileMetadataRepository);
    });

    it("should return null if the file does not exist", async () => {
      // When
      const metadata = await repo.get("non-existent-file");
      // Then
      expect(metadata).toBeNull();
    });

    it("should return the metadata if the file exists", async () => {
      // Given
      const id = "test-file";
      const metadata = { foo: "bar" };
      // When
      await repo.save(id, metadata);

      // Then
      const retrievedMetadata = await repo.get(id);
      expect(retrievedMetadata).toEqual(metadata);
    });
  });

  describe("save", () => {
    it("should create a new file with the given metadata", async () => {
      // Given
      const id = "test-file";
      const metadata = { baz: "qux" };

      // When
      await repo.save(id, metadata);

      // Then
      const fileExists = fs.existsSync(`${dirPath}/${id}.json`);
      expect(fileExists).toBe(true);

      const fileContents = fs.readFileSync(`${dirPath}/${id}.json`, "utf8");
      expect(JSON.parse(fileContents)).toEqual(metadata);
    });

    it("should overwrite an existing file with the given metadata", async () => {
      // Given
      const id = "test-file";
      const initialMetadata = { foo: "bar" };
      const updatedMetadata = { baz: "qux" };

      // When
      await repo.save(id, initialMetadata);
      await repo.save(id, updatedMetadata);

      // Then
      const fileContents = fs.readFileSync(`${dirPath}/${id}.json`, "utf8");
      expect(JSON.parse(fileContents)).toEqual(updatedMetadata);
    });
  });
});
