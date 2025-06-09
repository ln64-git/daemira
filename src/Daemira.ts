import { z } from "zod";
import { DynamicServerApp } from "../core/app";
import { spawn } from "child_process";

export type DaemiraState = z.infer<typeof DaemiraSchema>;
export const DaemiraSchema = z.object({
  port: z.number(),
  systemMessage: z.string().optional(),
});

export class Daemira extends DynamicServerApp<DaemiraState> {
  schema = DaemiraSchema;
  port = 2005;

  async keepSystemUpdated(): Promise<string> {
    const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
    let hasUpdatedToday = false;

    this.setSystemMessage("System update scheduler started.");

    (async () => {
      while (true) {
        const now = new Date();
        const hour = now.getHours();
        const minute = now.getMinutes();
        if (hour === 2 && minute === 9 && !hasUpdatedToday) {
          await this.updateSystem();
          hasUpdatedToday = true;
        }
        if (hour === 8 && hasUpdatedToday) {
          hasUpdatedToday = false;
        }
        await sleep(60_000);
      }
    })();

    return "System update scheduler started.";
  }

  private async updateSystem(): Promise<void> {
    const steps = [
      { name: "Updating package databases", cmd: "sudo pacman -Sy" },
      { name: "Upgrading packages", cmd: "sudo pacman -Su --noconfirm" },
      { name: "Cleaning up", cmd: "sudo paccache -r" },
    ];
    for (const step of steps) {
      this.systemLog.push(`🔹 ${step.name}...`);
      await runCommand(step.cmd, (line) => this.setSystemMessage(line));
      this.systemLog.push(`🔹 Done: ${step.name}`);
    }
    this.setSystemMessage("✅ System update completed successfully.");
  }
}

async function runCommand(cmd: string, onData: (line: string) => void): Promise<number> {
  return new Promise((resolve, reject) => {
    const parts = cmd.split(" ");
    if (!parts[0]) {
      reject(new Error("No command specified"));
      return;
    }
    const proc = spawn(parts[0], parts.slice(1), { stdio: ["ignore", "pipe", "pipe"] });
    const handle = (data: Buffer) => {
      data.toString().split("\n").forEach((line) => {
        if (line.trim()) onData(line);
      });
    };
    proc.stdout.on("data", handle);
    proc.stderr.on("data", handle);
    proc.on("error", reject);
    proc.on("close", resolve);
  });
}