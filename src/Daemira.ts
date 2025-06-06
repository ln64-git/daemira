import { z } from "zod";
import { DynamicServerApp } from "../core/app";

export type DaemiraState = z.infer<typeof DaemiraSchema>;
export const DaemiraSchema = z.object({
  port: z.number(),
  message: z.string(),
});

export class Daemira extends DynamicServerApp<DaemiraState> {
  schema = DaemiraSchema;
  port = 2005;

  message = "Hello, world!";

  async sampleFunction(): Promise<string> {
    return (this.message + " called from sampleFunction");
  }

}