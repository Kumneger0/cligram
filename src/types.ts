import { TelegramClient } from 'telegram';

export interface ChatUser {
	firstName: string;
	isBot: boolean;
	peerId: bigInt.BigInteger;
	accessHash: bigInt.BigInteger;
	unreadCount: number;
	lastSeen: Date | null;
	isOnline: boolean;
}

export interface FormattedMessage {
	id: number;
	sender: string;
	content: string;
	isFromMe: boolean;
	media: string | null;
	date: Date;
}

export const eventClassNames = ['UpdateUserStatus', 'UpdateShortMessage'] as const;

export interface ChannelDetails {
	title: string;
	username: string;
	channelusername: number | string;
	accessHash: number | string;
	isCreator: boolean;
	isBroadcast: boolean;
}

interface Message {
	flags: number;
	out: boolean;
	mentioned: boolean;
	mediaUnread: boolean;
	silent: boolean;
	post: boolean;
	fromScheduled: boolean;
	legacy: boolean;
	editHide: boolean;
	pinned: boolean;
	noforwards: boolean;
	invertMedia: boolean;
	flags2: number;
	offline: boolean;
	id: number;
	fromId: null;
	fromBoostsApplied: null;
	peerId: {
		channelId: string;
		className: string;
	};
	savedPeerId: null;
	fwdFrom: null;
	viaBotId: null;
	viaBusinessBotId: null;
	replyTo: null;
	date: number;
	message: string;
	media: MessageMedia;
	replyMarkup: null;
	entities: null;
	views: number;
	forwards: number;
	replies: null;
	editDate: null;
	postAuthor: null;
	groupedId: null;
	reactions: null;
	restrictionReason: null;
	ttlPeriod: null;
	quickReplyShortcutId: null;
	className: string;
}

export type MessageMedia = {
	flags: number;
	nopremium: boolean;
	spoiler: boolean;
	video: boolean;
	round: boolean;
	voice: boolean;
	document: {
		flags: number;
		id: string;
		accessHash: string;
		fileReference: {
			type: 'Buffer';
			data: number[];
		};
		date: number;
		mimeType: string;
		size: string;
		thumbs: (
			| {
					type: 'i';
					bytes: {
						type: 'Buffer';
						data: number[];
					};
					className: string;
			  }
			| {
					type: 'm';
					w: number;
					h: number;
					size: number;
					className: string;
			  }
		)[];
		videoThumbs: null;
		dcId: number;
		attributes: {
			w?: number;
			h?: number;
			fileName?: string;
			className: string;
		}[];
		className: string;
	};
	altDocument: null;
	ttlSeconds: null;
	className: string;
};

export default Message;

export type FilesData = {
	date: string | null;
	id: number;
	userId: string;
	fileName: string;
	mimeType: string;
	size: bigint;
	url: string;
	fileTelegramId: string;
	category: string;
}[];

export interface MessageMediaPhoto {
	flags: number;
	spoiler: boolean;
	photo: Photo;
	ttlSeconds: number | null;
	className: string;
}

export interface Photo {
	flags: number;
	hasStickers: boolean;
	id: string;
	accessHash: string;
	fileReference: FileReference;
	date: number;
	sizes: PhotoSize[];
	videoSizes: null;
	dcId: number;
	className: string;
}

interface FileReference {
	type: string;
	data: number[];
}

interface PhotoSize {
	type: string;
	bytes?: {
		type: string;
		data: number[];
	};
	className: string;
	w?: number;
	h?: number;
	size?: number;
	sizes?: number[];
}

export type MessageAction = {
	action: 'edit' | 'delete' | 'reply';
	id: number;
};
export type TGCliStore = {
	client: TelegramClient | null;
	updateClient: (client: TelegramClient) => void;
	selectedUser: ChatUser | null;
	setSelectedUser: (selectedUser: ChatUser | null) => void;
	messageAction: MessageAction | null;
	setMessageAction: (messageAction: MessageAction | null) => void;
};

type Integer = bigint; // Assuming you have a way to handle big integers

interface Media {
	className: 'MessageMediaWebPage' | 'MessageMediaDocument' | 'MessageMediaPhoto' | null;
	[key: string]: any;
}

interface MessageMediaWebPage extends Media {
	className: 'MessageMediaWebPage';
	flags: number;
	forceLargeMedia: boolean;
	forceSmallMedia: boolean;
	manual: boolean;
	safe: boolean;
	webpage: WebPage;
}

interface MessageMediaDocument extends Media {
	className: 'MessageMediaDocument';
	flags: number;
	nopremium: boolean;
	spoiler: boolean;
	document: Document;
	ttlSeconds: number | null;
}

interface MessageMediaPhoto extends Media {
	className: 'MessageMediaPhoto';
	flags: number;
	spoiler: boolean;
	photo: Photo;
	ttlSeconds: number | null;
}

type WebPage = WebPageDetails | WebPageEmpty;

interface WebPageEmpty {
	className: 'WebPageEmpty';
	flags: number;
	id: Integer;
	url: string;
}

interface WebPageDetails {
	className: 'WebPage';
	flags: number;
	hasLargeMedia: boolean;
	id: Integer;
	url: string;
	displayUrl: string;
	type: 'video' | 'telegram_bot' | 'telegram_user' | string;
	siteName: string;
	title: string;
	description: string | null;
	photo?: Photo;
	embedUrl: string | null;
	embedType: string | null;
	embedWidth: number | null;
	embedHeight: number | null;
	duration: number | null;
}

interface Document {
	className: 'Document';
	flags: number;
	id: Integer;
	accessHash: Integer;
	fileReference: Uint8Array;
	date: number;
	mimeType: string;
	size: Integer;
	dcId: number;
	attributes: any[]; // Replace with specific attribute types if known
}

interface Photo {
	className: 'Photo';
	flags: number;
	hasStickers: boolean;
	id: Integer;
	accessHash: Integer;
	fileReference: Uint8Array;
	date: number;
	sizes: Size[];
	dcId: number;
}

interface Size {
	type: string;
	width: number;
	height: number;
	size: number;
}

// Usage type:
type TelegramMedia = MessageMediaWebPage | MessageMediaDocument | MessageMediaPhoto | null;
