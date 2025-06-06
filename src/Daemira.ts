import { z } from "zod";
import { DynamicServerApp } from "../core/app";
import { spawn } from "child_process";

export type DaemiraState = z.infer<typeof DaemiraSchema>;
export const DaemiraSchema = z.object({
  port: z.number(),
});

export class Daemira extends DynamicServerApp<DaemiraState> {
  schema = DaemiraSchema;
  port = 2005;

  async updateSystem(): Promise<void> {
    const steps = [
      { name: "Updating package databases", cmd: "sudo pacman -Sy" },
      { name: "Upgrading packages", cmd: "sudo pacman -Su --noconfirm" },
      { name: "Cleaning up", cmd: "sudo paccache -r" },
    ];
    for (const step of steps) {
      console.log(`🔹 ${step.name}...`);
      await runCommand(step.cmd, (line) => {
        console.log(line);
      });
      console.log(`🔹 Done: ${step.name}\n`);
    }
  }

}

export async function runCommand(cmd: string, onData: (line: string) => void): Promise<number> {
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
