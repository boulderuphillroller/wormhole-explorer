import { BigNumber } from "ethers";
import { EvmLog, LogFoundEvent, LogMessagePublished } from "../../domain/entities";

export const evmLogMessagePublishedMapper = (
  log: EvmLog,
  parsedArgs: ReadonlyArray<any>
): LogFoundEvent<LogMessagePublished> => {
  return {
    name: "log-message-published",
    chainId: 2, // TODO: get from config
    txHash: log.transactionHash,
    blockHeight: log.blockNumber,
    blockTime: log.blockTime,
    attributes: {
      sender: parsedArgs[0], // log.topics[1]
      sequence: (parsedArgs[1] as BigNumber).toNumber(),
      payload: parsedArgs[3],
      nonce: parsedArgs[2],
      consistencyLevel: parsedArgs[4],
    },
  };
};
