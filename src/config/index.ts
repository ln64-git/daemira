/**
 * Daemira Configuration
 *
 * Personal system utility daemon configuration
 */

import { z } from "zod";
import * as dotenv from "dotenv";

dotenv.config();

const DaemiraConfigSchema = z.object({
	// Environment
	nodeEnv: z
		.enum(["development", "production", "test"])
		.default("development"),
	port: z.number().default(3000),

	// Logging
	logLevel: z.enum(["debug", "info", "warn", "error"]).default("info"),

	// Google Drive / rclone
	rcloneRemoteName: z.string().default("gdrive"),

	// Notion Integration
	notionToken: z.string().optional(),
	notionDatabaseId: z.string().optional(),
	notionPageIds: z.string().optional(), // Comma-separated page IDs

	// AI Providers
	openaiApiKey: z.string().optional(),
	geminiApiKey: z.string().optional(),
	grokApiKey: z.string().optional(),
});

export type DaemiraConfig = z.infer<typeof DaemiraConfigSchema>;

function loadConfig(): DaemiraConfig {
	const rawConfig = {
		nodeEnv: process.env.NODE_ENV || "development",
		port: Number.parseInt(process.env.PORT || "3000", 10),
		logLevel: process.env.LOG_LEVEL || "info",
		rcloneRemoteName: process.env.RCLONE_REMOTE_NAME || "gdrive",
		notionToken: process.env.NOTION_TOKEN,
		notionDatabaseId: process.env.NOTION_DATABASE_ID,
		notionPageIds: process.env.NOTION_PAGE_IDS,
		openaiApiKey: process.env.OPENAI_API_KEY,
		geminiApiKey: process.env.GEMINI_API_KEY,
		grokApiKey: process.env.GROK_API_KEY,
	};

	try {
		return DaemiraConfigSchema.parse(rawConfig);
	} catch (error) {
		if (error instanceof z.ZodError) {
			console.error("Configuration validation failed:");
			for (const issue of error.issues) {
				console.error(`  - ${issue.path.join(".")}: ${issue.message}`);
			}
		}
		throw error;
	}
}

export const config = loadConfig();

// Helper functions
export const isDevelopment = config.nodeEnv === "development";
export const isProduction = config.nodeEnv === "production";
export const isTest = config.nodeEnv === "test";
