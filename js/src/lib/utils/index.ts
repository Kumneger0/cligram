import { DialogInfo } from '@/telegram/client.types';
import {
	differenceInCalendarDays,
	differenceInSeconds,
	format,
	isThisWeek,
	isThisYear,
	isToday,
	isYesterday
} from 'date-fns';
import { LRUCache } from 'lru-cache';

export const ICONS = {
	USER: '👤',
	CHANNEL: '📢',
	GROUP: '👥',
	MESSAGE: '💬',
	SEARCH: '🔍',
	CHECK: '✓',
	CROSS: '✗',
	ARROW: '→',
	STAR: '⭐',
	WARNING: '⚠️',
	ERROR: '❌',
	SUCCESS: '✅',
	LOADING: '⏳',
	FOLDER: '📁',
	FILE: '📄',
	LINK: '🔗',
	CLOCK: '🕐',
	HEART: '❤️',
	PIN: '📌',
	LOCK: '🔒',
	UNLOCK: '🔓'
};

export const cache = new LRUCache<string, DialogInfo[]>({
	max: 100,
	ttl: 1000 * 60 * 5
});

export const entityCache = new LRUCache<string, any>({
	max: 100,
	ttl: 1000 * 60 * 5
});

type LastSeen =
	| {
			type: 'time';
			value: Date;
	  }
	| {
			type: 'status';
			value: string;
	  };

export function formatLastSeen(lastSeen: LastSeen) {
	if (lastSeen?.type === 'status') {
		return lastSeen.value;
	}
	const date = lastSeen?.value;

	if (!(date instanceof Date)) {
		return 'Invalid date';
	}

	const secondsDiff = differenceInSeconds(new Date(), date);
	if (secondsDiff < 60) {
		return 'last seen just now';
	}

	if (isToday(date)) {
		return `last seen at ${format(date, 'h:mm a')}`;
	}

	if (isYesterday(date)) {
		return `last seen yesterday at ${format(date, 'h:mm a')}`;
	}

	const daysDiff = differenceInCalendarDays(new Date(), date);
	if (daysDiff < 7 && isThisWeek(date)) {
		return `last seen on ${format(date, 'EEEE')} at ${format(date, 'h:mm a')}`;
	}
	if (isThisYear(date)) {
		return `last seen on ${format(date, 'MMM d')} at ${format(date, 'h:mm a')}`;
	}
	return `last seen on ${format(date, 'MMM d, yyyy')} at ${format(date, 'h:mm a')}`;
}

export { default as logger } from './logger';
