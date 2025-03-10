import {
	isToday,
	isYesterday,
	isThisWeek,
	isThisYear,
	format,
	differenceInSeconds,
	differenceInCalendarDays
} from 'date-fns';

export function formatLastSeen(date: Date) {
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


export const ICONS = {
	USER: '👤',
	CHANNEL: '📢',
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