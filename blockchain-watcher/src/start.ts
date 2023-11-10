import { HandleEvmLogs } from "./domain/actions/HandleEvmLogs";
import { PollEvmLogs, PollEvmLogsConfig } from "./domain/actions/PollEvmLogs";
import { LogFoundEvent } from "./domain/entities";
import { configuration } from "./infrastructure/config";
import { evmLogMessagePublishedMapper } from "./infrastructure/mappers/evmLogMessagePublishedMapper";
import { RepositoriesBuilder } from "./infrastructure/RepositoriesBuilder";

let repos: RepositoriesBuilder;

async function run(): Promise<void> {
  console.log(`Starting: dryRunEnabled -> ${configuration.dryRun}`);

  repos = new RepositoriesBuilder(configuration);

  /** Job definition is hardcoded, but should be loaded from cfg or a data store soon enough */
  const jobs = [
    {
      id: "poll-log-message-published-ethereum",
      chain: "ethereum",
      source: {
        action: "PollEvmLogs",
        config: {
          fromBlock: 10012499n,
          toBlock: 10012999n,
          blockBatchSize: 100,
          commitment: "latest",
          interval: 15_000,
          addresses: ["0x706abc4E45D419950511e474C7B9Ed348A4a716c"],
          topics: [],
        },
      },
      handlers: [
        {
          action: "HandleEvmLogs",
          target: "sns",
          mapper: "evmLogMessagePublishedMapper",
          config: {
            abi: "event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel)",
            filter: {
              addresses: ["0x706abc4E45D419950511e474C7B9Ed348A4a716c"],
              topics: ["0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2"],
            },
          },
        },
      ],
    },
  ];

  const pollEvmLogs = new PollEvmLogs(
    repos.getEvmBlockRepository("ethereum"),
    repos.getMetadataRepository(),
    new PollEvmLogsConfig({ ...jobs[0].source.config, id: jobs[0].id })
  );

  const snsTarget = async (events: LogFoundEvent<any>[]) => {
    const result = await repos.getSnsEventRepository().publish(events);
    if (result.status === "error") {
      console.error(`Error publishing events to SNS: ${result.reason ?? result.reasons}`);
      throw new Error(`Error publishing events to SNS: ${result.reason}`);
    }
    console.log(`Published ${events.length} events to SNS`);
  };
  const handleEvmLogs = new HandleEvmLogs<LogFoundEvent<any>>(
    jobs[0].handlers[0].config,
    evmLogMessagePublishedMapper,
    configuration.dryRun ? async (events) => console.log(`Got ${events.length} events`) : snsTarget
  );

  pollEvmLogs.start([handleEvmLogs.handle.bind(handleEvmLogs)]);

  // Just keep this running until killed
  setInterval(() => {
    console.log("Still running");
  }, 20_000);

  console.log("Started");
  // Handle shutdown
  process.on("SIGINT", handleShutdown);
  process.on("SIGTERM", handleShutdown);
}

const handleShutdown = async () => {
  try {
    await Promise.allSettled([
      repos.close(),
      // call stop() on all the things
    ]);

    process.exit();
  } catch (error: unknown) {
    process.exit(1);
  }
};

run().catch((e) => {
  console.error(e);
  console.error("Fatal error caused process to exit");
});
