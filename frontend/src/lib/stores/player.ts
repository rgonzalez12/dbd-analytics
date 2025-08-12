import { writable, derived, type Writable } from 'svelte/store';
import type { Player } from '$lib/api/types';
import type { TabId, DbdAdept, DbdAchievement } from '$lib/types';

const ALLOWED_TABS: TabId[] = ['stats', 'adepts', 'achievements'];

export const player = writable<Player | null>(null);

export const selectedTab = writable<TabId>('stats');

export const counts = derived(player, ($p) => {
	if (!$p) {
		return { adeptsUnlocked: 0, achievementsUnlocked: 0, achievementsTotal: 0 };
	}

	const survivorUnlocked = Object.values($p.achievements?.adepts?.survivors || {}).filter(Boolean).length;
	const killerUnlocked = Object.values($p.achievements?.adepts?.killers || {}).filter(Boolean).length;
	const adeptsUnlocked = survivorUnlocked + killerUnlocked;

	const achievementsUnlocked = $p.achievements?.unlocked ?? 
		$p.achievements?.mapped?.filter(a => a.unlocked).length ?? 0;
	const achievementsTotal = $p.achievements?.total ?? 
		$p.achievements?.mapped?.length ?? 0;

	return { adeptsUnlocked, achievementsUnlocked, achievementsTotal };
});

export const adeptsList = derived(player, ($p): DbdAdept[] => {
	if (!$p?.achievements?.adepts) {
		return [];
	}

	const survivors = Object.entries($p.achievements.adepts.survivors || {}).map(([character, unlocked]) => ({
		id: `adept_survivor_${character}`,
		name: `Adept ${character}`,
		unlocked,
		unlockTime: null
	}));

	const killers = Object.entries($p.achievements.adepts.killers || {}).map(([character, unlocked]) => ({
		id: `adept_killer_${character}`,
		name: `Adept ${character}`,
		unlocked,
		unlockTime: null
	}));

	return [...survivors, ...killers];
});

export const achievementsList = derived(player, ($p): DbdAchievement[] => {
	if (!$p?.achievements?.mapped) {
		return [];
	}

	return $p.achievements.mapped.map(achievement => ({
		id: achievement.id,
		name: achievement.character || achievement.id,
		description: `${achievement.type} achievement for ${achievement.character}`,
		unlocked: achievement.unlocked,
		unlockTime: null
	}));
});

export function syncTabWithUrl(tabStore: Writable<TabId>): () => void {
	if (typeof window === 'undefined') {
		return () => {};
	}

	const urlParams = new URLSearchParams(window.location.search);
	const urlTab = urlParams.get('tab') as TabId;
	if (urlTab && ALLOWED_TABS.includes(urlTab)) {
		tabStore.set(urlTab);
	}

	const unsubscribe = tabStore.subscribe((tab) => {
		const url = new URL(window.location.href);
		url.searchParams.set('tab', tab);
		window.history.replaceState({}, '', url.toString());
	});

	return unsubscribe;
}

export function setPlayer(p: Player): void {
	player.set(p);
}

export function resetPlayer(): void {
	player.set(null);
}
