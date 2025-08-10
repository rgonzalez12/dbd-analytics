export type PlayerSummary = {
	steam_id: string;
	display_name?: string;
	avatar?: string;
};

export type PlayerStats = {
	steam_id: string;
	display_name: string;
	total_matches: number;
	[k: string]: unknown;
};

export type PlayerStatsSurface = {
	steam_id?: string;
	display_name?: string;
	total_matches?: number;
};

export type ApiError = {
	status: number;
	message: string;
	code?: string;
};
