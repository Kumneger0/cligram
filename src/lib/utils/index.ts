import { Api } from 'telegram';
import {
	isToday,
	isYesterday,
	isThisWeek,
	isThisYear,
	format,
	differenceInSeconds,
	differenceInCalendarDays,
} from 'date-fns';

export function formatBytes(bytes: number) {
	const KB = 1024;
	const MB = KB * 1024;
	const GB = MB * 1024;

	if (bytes < KB) return `${bytes} Bytes`;
	if (bytes < MB) return `${(bytes / KB).toFixed(2)} KB`;
	if (bytes < GB) return `${(bytes / MB).toFixed(2)} MB`;

	return `${(bytes / GB).toFixed(2)} GB`;
}

export const getChannelEntity = (channelId: string, accessHash: string) => {
	return new Api.InputChannel({
		//@ts-ignore
		channelId: channelId,
		//@ts-ignore
		accessHash: accessHash
	});
};




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