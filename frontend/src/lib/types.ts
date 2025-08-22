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
	iconGrayUrl?: string;
	hidden?: boolean;
	character?: string;
	type?: string;
	rarity?: number; // 0-100 global completion percentage
}
