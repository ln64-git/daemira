import { App } from "../core/app";
import { spawn } from "child_process";

export class Daemira extends App {
  systemLog: string[] = [];

  setSystemMessage(message: string): void {
    this.systemLog.push(message);
    console.log(`🔹 ${message}`);
  }

  async keepSystemUpdated(): Promise<string> {
    const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
    const SIX_HOURS = 6 * 60 * 60 * 1000;

    (async () => {
      while (true) {
        this.setSystemMessage("🕐 Starting scheduled system update...");
        await this.updateSystem();
        this.setSystemMessage(`⏰ Next update scheduled in 6 hours.`);
        await sleep(SIX_HOURS);
      }
    })();

    return "System update scheduler started (runs every 6 hours).";
  }

  private async updateSystem(): Promise<void> {
    const steps = [
      { name: "Refreshing mirrorlist", cmd: "sudo pacman-mirrors --fasttrack", optional: true },
      { name: "Updating keyrings", cmd: "sudo pacman -Sy --needed --noconfirm archlinux-keyring cachyos-keyring" },
      { name: "Updating package databases", cmd: "sudo pacman -Syy --noconfirm" },
      { name: "Upgrading packages", cmd: "sudo pacman -Syu --noconfirm" },
      { name: "Updating AUR packages", cmd: "yay -Sua --noconfirm --answerclean All --answerdiff None --answeredit None --removemake --cleanafter" },
      { name: "Updating firmware", cmd: "sudo fwupdmgr refresh --force && sudo fwupdmgr update -y", optional: true },
      { name: "Removing orphaned packages", cmd: "orphans=$(pacman -Qdtq 2>/dev/null); [ -z \"$orphans\" ] || sudo pacman -Rns --noconfirm $orphans" },
      { name: "Cleaning package cache", cmd: "sudo paccache -rk2" },
      { name: "Cleaning uninstalled cache", cmd: "sudo paccache -ruk0" },
      { name: "Cleaning yay cache", cmd: "yay -Sc --noconfirm --answerclean All" },
      { name: "Optimizing pacman database", cmd: "sudo pacman-optimize", optional: true },
      { name: "Updating GRUB", cmd: "sudo grub-mkconfig -o /boot/grub/grub.cfg" },
      { name: "Reloading systemd daemon", cmd: "sudo systemctl daemon-reload" },
    ];

    for (const step of steps) {
      this.systemLog.push(`🔹 ${step.name}...`);
      try {
        const exitCode = await runCommand(step.cmd, (line) => this.setSystemMessage(line));
        if (exitCode === 0) {
          this.systemLog.push(`✅ Done: ${step.name}`);
        } else {
          if (step.optional) {
            this.systemLog.push(`⚠️ Skipped: ${step.name} (optional, exit code ${exitCode})`);
          } else {
            this.systemLog.push(`⚠️ Warning: ${step.name} exited with code ${exitCode}`);
          }
        }
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error);
        if (step.optional) {
          this.systemLog.push(`⚠️ Skipped: ${step.name} (optional) - ${errorMsg}`);
        } else {
          this.systemLog.push(`⚠️ Skipped: ${step.name} - ${errorMsg}`);
        }
      }
    }

    await this.checkPacnewFiles();
    await this.checkRebootRequired();
    this.setSystemMessage("✅ System update completed successfully.");
  }

  private async checkPacnewFiles(): Promise<void> {
    try {
      const pacnewFiles: string[] = [];
      const proc = spawn("bash", ["-c", "find /etc -name '*.pacnew' 2>/dev/null"]);

      proc.stdout.on("data", (data) => {
        const files = data.toString().split("\n").filter((line: string) => line.trim());
        pacnewFiles.push(...files);
      });

      await new Promise((resolve) => proc.on("close", resolve));

      if (pacnewFiles.length > 0) {
        this.setSystemMessage(`⚠️ Found ${pacnewFiles.length} .pacnew file(s) that may need manual merging:`);
        pacnewFiles.forEach((file) => this.setSystemMessage(`   ${file}`));
        this.setSystemMessage("   Consider using 'pacdiff' to merge configuration changes.");
      }
    } catch (error) {
      this.systemLog.push("⚠️ Could not check for .pacnew files");
    }
  }

  private async checkRebootRequired(): Promise<void> {
    try {
      const checkKernel = async (): Promise<boolean> => {
        return new Promise((resolve) => {
          const proc = spawn("bash", ["-c", "[ -f /usr/lib/modules/$(uname -r)/modules.dep ]"]);
          proc.on("close", (code) => resolve(code !== 0));
        });
      };

      const needsReboot = await checkKernel();
      if (needsReboot) {
        this.setSystemMessage("🔄 Kernel update detected - reboot recommended for changes to take effect.");
      }
    } catch (error) {
      this.systemLog.push("⚠️ Could not check reboot status");
    }
  }
}

async function runCommand(cmd: string, onData: (line: string) => void): Promise<number> {
  return new Promise((resolve, reject) => {
    const proc = spawn("bash", ["-c", cmd], {
      stdio: ["ignore", "pipe", "pipe"],
      env: { ...process.env, DEBIAN_FRONTEND: "noninteractive" }
    });
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