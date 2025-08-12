// Shared types for DBD profile page

export type TabId = 'stats' | 'adepts' | 'achievements';

export interface DbdAdept {
	id: string;
	name: string;
	unlocked: boolean;
	unlockTime?: string | null;
}

export interface DbdAchievement {
	id: string;
	name: string;
	description: string;
	unlocked: boolean;
	unlockTime?: string | null;
	iconUrl?: string;
}

export interface DbdStats {
	[key: string]: number;
	matchesPlayed?: number;
	escapes?: number;
	kills?: number;
}

export interface Player {
	steamId: string;
	personaName: string;
	avatarUrl?: string;
}

export interface PlayerBundle {
	player: Player;
	stats: DbdStats;
	adepts: DbdAdept[];
	achievements: DbdAchievement[];
}
