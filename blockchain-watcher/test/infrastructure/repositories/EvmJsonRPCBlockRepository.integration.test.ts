import { describe, it, expect, afterEach, afterAll } from "@jest/globals";
import { EvmJsonRPCBlockRepository } from "../../../src/infrastructure/repositories";
import axios from "axios";
import nock from "nock";
import { EvmLogFilter, EvmTag } from "../../../src/domain/entities";
import { HttpClient } from "../../../src/infrastructure/http/HttpClient";

axios.defaults.adapter = "http"; // needed by nock
const rpc = "http://localhost";
const address = "0x98f3c9e6e3face36baad05fe09d375ef1464288b";
const topic = "0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2";
const txHash = "0xcbdefc83080a8f60cbde7785eb2978548fd5c1f7d0ea2c024cce537845d339c7";

let repo: EvmJsonRPCBlockRepository;

describe("EvmJsonRPCBlockRepository", () => {
  afterAll(() => {
    nock.restore();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  it("should be able to get block height", async () => {
    const expectedHeight = 1980809n;
    givenARepo();
    givenBlockHeightIs(expectedHeight, "latest");

    const result = await repo.getBlockHeight("latest");

    expect(result).toBe(expectedHeight);
  });

  it("should be able to get several blocks", async () => {
    const blockNumbers = [2n, 3n, 4n];
    givenARepo();
    givenBlocksArePresent(blockNumbers);

    const result = await repo.getBlocks(new Set(blockNumbers));

    expect(Object.keys(result)).toHaveLength(blockNumbers.length);
    blockNumbers.forEach((blockNumber) => {
      expect(result[blockHash(blockNumber)].number).toBe(blockNumber);
    });
  });

  it("should be able to get logs", async () => {
    const filter: EvmLogFilter = {
      fromBlock: "safe",
      toBlock: "latest",
      addresses: [address],
      topics: [],
    };

    givenLogsPresent(filter);

    const logs = await repo.getFilteredLogs(filter);

    expect(logs).toHaveLength(1);
    expect(logs[0].blockNumber).toBe(1n);
    expect(logs[0].blockHash).toBe(blockHash(1n));
    expect(logs[0].address).toBe(address);
  });
});

const givenARepo = () => {
  repo = new EvmJsonRPCBlockRepository(
    { rpc, timeout: 100, chain: "ethereum", chainId: 2 },
    new HttpClient()
  );
};

const givenBlockHeightIs = (height: bigint, commitment: EvmTag) => {
  nock(rpc)
    .post("/", {
      jsonrpc: "2.0",
      method: "eth_getBlockByNumber",
      params: [commitment, false],
      id: 1,
    })
    .reply(200, {
      jsonrpc: "2.0",
      id: 1,
      result: {
        number: `0x${height.toString(16)}`,
        hash: blockHash(height),
        timestamp: "0x654a892f",
      },
    });
};

const givenBlocksArePresent = (blockNumbers: bigint[]) => {
  const requests = blockNumbers.map((blockNumber) => ({
    jsonrpc: "2.0",
    method: "eth_getBlockByNumber",
    params: [`0x${blockNumber.toString(16)}`, false],
    id: blockNumber.toString(),
  }));
  const response = blockNumbers.map((blockNumber) => ({
    jsonrpc: "2.0",
    id: blockNumber.toString(),
    result: {
      number: `0x${blockNumber.toString(16)}`,
      hash: blockHash(blockNumber),
      timestamp: "0x654a892f",
    },
  }));

  nock(rpc).post("/", requests).reply(200, response);
};

const givenLogsPresent = (filter: EvmLogFilter) => {
  const response = {
    jsonrpc: "2.0",
    id: 1,
    result: [
      {
        address: filter.addresses[0],
        topics: [topic],
        data: "0x",
        blockNumber: "0x1",
        blockHash: blockHash(1n),
        transactionHash: txHash,
        transactionIndex: "0x0",
        logIndex: 0,
        removed: false,
      },
    ],
  };

  nock(rpc).post("/").reply(200, response);
};

const blockHash = (blockNumber: bigint) => `0x${blockNumber.toString(16)}`;

/* Examples:
    - blockByNumber:
            {
                "jsonrpc":"2.0",
                "method":"eth_getBlockByNumber",
                "params":[
                    "latest", 
                    true
                ],
                "id":1
            }
        -> 
           {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "baseFeePerGas": "0x78d108bbb",
                    "difficulty": "0x0",
                    "extraData": "0x406275696c64657230783639",
                    "gasLimit": "0x1c9c380",
                    "gasUsed": "0xfca882",
                    "hash": "0xb29ad2fa313f50d293fcf5679c6862ee7f4a3d641f09b227ad0ee3fba10d1cbb",
                    "logsBloom": "0x312390b5e798baf83b514972932b522118d98b9888f5880461db7e26f4bece2e8141717140d492028f887928435127083671a3488c3b6c240130a6eeb3692d908417d91c65fbfd396d0ade2b57a263a080c64f59fb4d3c4033415503e833306057524072daeae803a45cc020c1a32f436f30037b49003cd257c965d9214b441922012654b681e5202053a7d58500a64aa040cec9d90a0c9e5e3321d821503d90cfb84961594a72f02e92c7c2559d95c86504c54260c708ea63e5e4a2538f1143096c2422250a0a20b321a8814678d26e6a6d6a872e232a500a402a3a6445b85b3cf92b481e9020c20a969eac4c50ca08667cda68812f8141108908b3d175f649",
                    "miner": "0x690b9a9e9aa1c9db991c7721a92d351db4fac990",
                    "mixHash": "0xbdeea2aa4f2a026b27bc720d28c73680a35ad3e5017568cddcb066b5c12b1f60",
                    "nonce": "0x0000000000000000",
                    "number": "0x11a9fa9",
                    "parentHash": "0x7f8c4ecd8772eab825ee3c8e713c5088c6a32b41d61dbd1c0833e7d4df337713",
                    "receiptsRoot": "0x06e3cd06761468089708b204a092545576c508739f0eff936c96914da2e277e9",
                    "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
                    "size": "0x25301",
                    "stateRoot": "0x3e1333543583ec7e5fb4e337261c43ef94aeb4f77c4e7f657a538fc3c2e5b6de",
                    "timestamp": "0x654a892f",
                    "totalDifficulty": "0xc70d815d562d3cfa955",
                    "transactions": [
                        "0x5e59c0bb917e7a5a64f098cd6a370bac4f40ecdf6ca79deaccf25736fe117ef7"
                    ],
                    "transactionsRoot": "0xbf23b57ac6f6aede4d886a556b7bbee868721542b9a1d912ffa5f4ead0b8ec72",
                    "uncles": [],
                    "withdrawals": [
                        {
                            "index": "0x16b0d86",
                            "validatorIndex": "0x2a57b",
                            "address": "0xdaac5ce35ad7892d5f2dd364954066f4323c9a57",
                            "amount": "0x105fdec"
                        }
                    ]
                }
            }
    - blockByHash:
*/
