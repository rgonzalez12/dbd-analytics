export type PlayerSummary = {
	steam_id: string;
	persona_name?: string;
	avatar?: string;
};

export type PlayerStats = {
	steam_id: string;
	[k: string]: unknown;
};

export type PlayerStatsSurface = {
	steam_id?: string;
	persona_name?: string;
	matches?: number;
};

export type ApiError = {
	status: number;
	message: string;
	code?: string;
};
