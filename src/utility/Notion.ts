/**
 * Notion Utility - Integration with Notion API
 *
 * Features:
 * - Database queries with filtering
 * - Page CRUD operations (create, read, update)
 * - Append content blocks to pages
 * - Sync local files to Notion pages
 * - Retry logic with exponential backoff
 * - Integration with Logger
 */

import { Client } from "@notionhq/client";
import { Logger } from "./Logger";
import type {
	PageObjectResponse,
	QueryDatabaseResponse,
	CreatePageParameters,
} from "@notionhq/client/build/src/api-endpoints";
import { readFileSync } from "node:fs";

export interface NotionOptions {
	logLevel?: "debug" | "info" | "warn" | "error";
}

export interface PageFilter {
	property?: string;
	value?: string;
	// Add more filter types as needed
}

/**
 * Notion API client with CRUD operations and file syncing
 */
export class Notion {
	private client: Client;
	private logger = Logger.getInstance();

	constructor(token: string, options: NotionOptions = {}) {
		if (!token) {
			throw new Error("Notion token is required");
		}

		this.client = new Client({ auth: token });
		this.logger.info("Notion client initialized");
	}

	/**
	 * Query a database
	 */
	async queryDatabase(
		databaseId: string,
		filter?: PageFilter,
	): Promise<QueryDatabaseResponse> {
		this.logger.debug(`Querying database: ${databaseId}`);

		try {
			const response = await this._retryWrapper(() =>
				this.client.databases.query({
					database_id: databaseId,
					filter: filter ? this._buildFilter(filter) : undefined,
				}),
			);

			this.logger.info(
				`Retrieved ${response.results.length} pages from database`,
			);
			return response;
		} catch (error) {
			this.logger.error(`Failed to query database: ${error}`);
			throw error;
		}
	}

	/**
	 * Get a page by ID
	 */
	async getPage(pageId: string): Promise<PageObjectResponse> {
		this.logger.debug(`Fetching page: ${pageId}`);

		try {
			const response = await this._retryWrapper(() =>
				this.client.pages.retrieve({ page_id: pageId }),
			);

			this.logger.debug(`Retrieved page: ${pageId}`);
			return response as PageObjectResponse;
		} catch (error) {
			this.logger.error(`Failed to retrieve page: ${error}`);
			throw error;
		}
	}

	/**
	 * Create a new page in a database
	 */
	async createPage(
		databaseId: string,
		properties: Record<string, any>,
		content?: any[],
	): Promise<PageObjectResponse> {
		this.logger.debug(`Creating page in database: ${databaseId}`);

		try {
			const params: CreatePageParameters = {
				parent: { database_id: databaseId },
				properties,
			};

			if (content) {
				params.children = content;
			}

			const response = await this._retryWrapper(() =>
				this.client.pages.create(params),
			);

			this.logger.info(`Created page: ${response.id}`);
			return response as PageObjectResponse;
		} catch (error) {
			this.logger.error(`Failed to create page: ${error}`);
			throw error;
		}
	}

	/**
	 * Update an existing page
	 */
	async updatePage(
		pageId: string,
		properties: Record<string, any>,
	): Promise<PageObjectResponse> {
		this.logger.debug(`Updating page: ${pageId}`);

		try {
			const response = await this._retryWrapper(() =>
				this.client.pages.update({
					page_id: pageId,
					properties,
				}),
			);

			this.logger.info(`Updated page: ${pageId}`);
			return response as PageObjectResponse;
		} catch (error) {
			this.logger.error(`Failed to update page: ${error}`);
			throw error;
		}
	}

	/**
	 * Append content blocks to a page
	 */
	async appendBlocks(pageId: string, blocks: any[]): Promise<void> {
		this.logger.debug(`Appending ${blocks.length} blocks to page: ${pageId}`);

		try {
			await this._retryWrapper(() =>
				this.client.blocks.children.append({
					block_id: pageId,
					children: blocks,
				}),
			);

			this.logger.info(`Appended ${blocks.length} blocks to page`);
		} catch (error) {
			this.logger.error(`Failed to append blocks: ${error}`);
			throw error;
		}
	}

	/**
	 * Sync local file content to a Notion page as blocks
	 */
	async syncFileToPage(
		pageId: string,
		filePath: string,
		options: { overwrite?: boolean } = {},
	): Promise<void> {
		this.logger.info(`Syncing file to Notion page: ${filePath} -> ${pageId}`);

		try {
			const content = readFileSync(filePath, "utf-8");

			// Convert file content to Notion blocks
			const blocks = this._fileContentToBlocks(content, filePath);

			if (options.overwrite) {
				// TODO: Clear existing blocks first
				this.logger.warn("Overwrite mode not yet implemented");
			}

			await this.appendBlocks(pageId, blocks);
			this.logger.info("Successfully synced file to Notion");
		} catch (error) {
			this.logger.error(`Failed to sync file to Notion: ${error}`);
			throw error;
		}
	}

	/**
	 * Convert file content to Notion blocks
	 */
	private _fileContentToBlocks(content: string, filePath: string): any[] {
		const blocks: any[] = [];

		// Detect file type
		const ext = filePath.split(".").pop()?.toLowerCase();

		if (ext === "md" || ext === "markdown") {
			// Simple markdown parsing
			const lines = content.split("\n");

			for (const line of lines) {
				if (line.startsWith("# ")) {
					blocks.push({
						type: "heading_1",
						heading_1: { rich_text: [{ text: { content: line.slice(2) } }] },
					});
				} else if (line.startsWith("## ")) {
					blocks.push({
						type: "heading_2",
						heading_2: { rich_text: [{ text: { content: line.slice(3) } }] },
					});
				} else if (line.startsWith("### ")) {
					blocks.push({
						type: "heading_3",
						heading_3: { rich_text: [{ text: { content: line.slice(4) } }] },
					});
				} else if (line.trim()) {
					blocks.push({
						type: "paragraph",
						paragraph: {
							rich_text: [{ text: { content: line } }],
						},
					});
				}
			}
		} else {
			// Plain text - code block
			blocks.push({
				type: "code",
				code: {
					rich_text: [{ text: { content } }],
					language: ext || "plain text",
				},
			});
		}

		return blocks;
	}

	/**
	 * Build Notion filter from simple filter object
	 */
	private _buildFilter(filter: PageFilter): any {
		// Simple implementation - expand as needed
		if (filter.property && filter.value) {
			return {
				property: filter.property,
				rich_text: {
					contains: filter.value,
				},
			};
		}
		return undefined;
	}

	/**
	 * Retry wrapper with exponential backoff
	 */
	private async _retryWrapper<T>(
		operation: () => Promise<T>,
		maxRetries = 3,
		baseDelay = 1000,
	): Promise<T> {
		let lastError: any;

		for (let attempt = 0; attempt <= maxRetries; attempt++) {
			try {
				return await operation();
			} catch (error: any) {
				lastError = error;

				// Don't retry on auth errors or bad requests
				if (error.status === 401 || error.status === 400) {
					throw error;
				}

				if (attempt < maxRetries) {
					const delay = baseDelay * Math.pow(2, attempt);
					this.logger.warn(
						`Notion API error, retrying in ${delay}ms (attempt ${attempt + 1}/${maxRetries})`,
					);
					await new Promise((resolve) => setTimeout(resolve, delay));
				}
			}
		}

		throw lastError;
	}
}
