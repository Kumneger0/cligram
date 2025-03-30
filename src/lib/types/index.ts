import { TelegramClient } from 'telegram';

export type TelegramUser = {
	flags: number;
	self: boolean;
	contact: boolean;
	mutualContact: boolean;
	deleted: boolean;
	bot: boolean;
	botChatHistory: boolean;
	botNochats: boolean;
	verified: boolean;
	restricted: boolean;
	min: boolean;
	botInlineGeo: boolean;
	support: boolean;
	scam: boolean;
	applyMinPhoto: boolean;
	fake: boolean;
	botAttachMenu: boolean;
	premium: boolean;
	attachMenuEnabled: boolean;
	flags2: number;
	botCanEdit: boolean;
	closeFriend: boolean;
	storiesHidden: boolean;
	storiesUnavailable: boolean;
	contactRequirePremium: boolean;
	botBusiness: boolean;
	id: string;
	accessHash: string;
	firstName: string;
	lastName: string | null;
	username: string;
	phone: string | null;
	photo: string | null;
	status: {
		wasOnline: number;
		className: string;
	} | null;
	botInfoVersion: number;
	restrictionReason: string | null;
	botInlinePlaceholder: string | null;
	langCode: string | null;
	emojiStatus: string | null;
	usernames: string | null;
	storiesMaxId: string | null;
	color: string | null;
	profileColor: string | null;
	className: string;
};

export type Message = {
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
	fromId: { userId: string; className: string } | null;
	fromBoostsApplied: unknown;
	peerId: { userId: string; className: string };
	savedPeerId: unknown;
	fwdFrom: unknown;
	viaBotId: unknown;
	viaBusinessBotId: unknown;
	replyTo: unknown;
	date: number;
	message: string;
	media: unknown;
	replyMarkup: unknown;
	entities: unknown;
	views: unknown;
	forwards: unknown;
	replies: unknown;
	editDate: number | null;
	postAuthor: unknown;
	groupedId: unknown;
	reactions: unknown;
	restrictionReason: unknown;
	ttlPeriod: unknown;
	quickReplyShortcutId: unknown;
	effect: unknown;
	factcheck: unknown;
	className: string;
};

export type MessagesSlice = {
	flags: number;
	inexact: boolean;
	count: number;
	nextRate: unknown;
	offsetIdOffset: unknown;
	messages: Message[];
	chats: unknown[];
	users: UserInfo[];
	className: string;
};

export type UserInfo = {
	firstName: string;
	isBot: boolean;
	peerId: bigInt.BigInteger;
	accessHash: bigInt.BigInteger;
	unreadCount: number;
	lastSeen:
		| {
				type: 'time';
				value: Date;
		  }
		| {
				type: 'status';
				value: string;
		  }
		| null;
	isOnline: boolean;
};

export type FormattedMessage = {
	id: number;
	sender: string;
	content: string;
	isFromMe: boolean;
	media: string | null;
	date: Date;
	isUnsupportedMessage: boolean;
	webPage?: {
		url: string;
		displayUrl: string | null;
	} | null;
	document?: {
		document: string;
	} | null;
	fromId?: bigInt.BigInteger | null;
};

export const eventClassNames = ['UpdateUserStatus', 'UpdateShortMessage'] as const;

export type ChannelDetails = {
	title: string;
	username: string;
	channelusername: number | string;
	accessHash: number | string;
	isCreator: boolean;
	isBroadcast: boolean;
};

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

export type MessageMediaPhoto = {
	flags: number;
	spoiler: boolean;
	photo: Photo;
	ttlSeconds: number | null;
	className: string;
};

export type Photo = {
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
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	classType: string;
};

type FileReference = {
	type: string;
	data: number[];
};

type PhotoSize = {
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
};

export type MessageAction = {
	action: 'edit' | 'delete' | 'reply';
	id: number;
};

type Integer = {
	value: bigint;
};

export type ChannelInfo = {
	title: string;
	username: string | undefined;
	channelId: string;
	accessHash: string;
	isCreator: boolean;
	isBroadcast: boolean;
	participantsCount: number | null;
	unreadCount: number;
};

type SeachMode = 'CONVERSATION' | 'CHANNELS_OR_ USERS' | null;

export type ChatType = 'user' | 'channel' | 'group' | 'bot';

export type TGCliStore = {
	client: TelegramClient | null;
	updateClient: (client: TelegramClient) => void;
	searchMode: SeachMode;
	setSearchMode: (searchMode: SeachMode) => void;
	selectedUser: UserInfo | ChannelInfo | null;
	setSelectedUser: (selectedUser: UserInfo | ChannelInfo | null) => void;
	messageAction: MessageAction | null;
	setMessageAction: (messageAction: MessageAction | null) => void;
	currentChatType: ChatType;
	setCurrentChatType: (currentChatType: ChatType) => void;
	currentlyFocused: 'chatArea' | 'sidebar' | null;
	setCurrentlyFocused: (currentlyFocused: 'chatArea' | 'sidebar' | null) => void;
};

export type ForwardMessageOptions = {
	fromPeer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger };
	id: number[];
	type: ChatType;
};

type ChatPhoto = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'ChatPhoto';
	classType: 'constructor';
	flags: number;
	hasVideo: boolean;
	photoId: Integer;
	strippedThumb: Buffer;
	dcId: number;
};

export type Channel = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'Channel';
	classType: 'constructor';
	flags: number;
	flags2: number;

	// Boolean flags
	creator: boolean;
	left: boolean;
	broadcast: boolean;
	verified: boolean;
	megagroup: boolean;
	restricted: boolean;
	signatures: boolean;
	min: boolean;
	scam: boolean;
	hasLink: boolean;
	hasGeo: boolean;
	slowmodeEnabled: boolean;
	callActive: boolean;
	callNotEmpty: boolean;
	fake: boolean;
	gigagroup: boolean;
	noforwards: boolean;
	joinToSend: boolean;
	joinRequest: boolean;
	forum: boolean;
	storiesHidden: boolean;
	storiesHiddenMin: boolean;
	storiesUnavailable: boolean;
	signatureProfiles: boolean;

	// Main properties
	id: Integer;
	accessHash: Integer;
	title: string;
	username: string;
	photo: ChatPhoto;
	date: number;
	participantsCount: number | null;
};

type BaseMedia = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'MessageMediaWebPage' | 'MessageMediaDocument' | 'MessageMediaPhoto';
	classType: string;
	flags: number;
};

export type Media = MessageMediaWebPage | MessageMediaDocument | MessageMediaPhoto | null;

export type MessageMediaWebPage = {
	forceLargeMedia: boolean;
	forceSmallMedia: boolean;
	manual: boolean;
	safe: boolean;
	webpage: WebPage | WebPageEmpty;
} & BaseMedia;

export type WebPage = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'WebPage';
	classType: string;
	flags: number;
	hasLargeMedia: boolean;
	id: bigint;
	url: string;
	displayUrl: string;
	hash: number;
	type: string;
	siteName: string;
	title: string;
	description: string | null;
	photo: Photo;
	embedUrl: string | null;
	embedType: string | null;
	embedWidth: number | null;
	embedHeight: number | null;
	duration: number | null;
	author: string | null;
	document: unknown | null;
	cachedPage: unknown | null;
	attributes: unknown | null;
};

export type WebPageEmpty = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'WebPageEmpty';
	classType: string;
	flags: number;
	id: bigint;
	url: string;
};

export type MessageMediaDocument = {
	nopremium: boolean;
	spoiler: boolean;
	video: boolean;
	round: boolean;
	voice: boolean;
	document: Document;
	altDocuments: unknown | null;
	videoCover: unknown | null;
	videoTimestamp: unknown | null;
	ttlSeconds: number | null;
} & BaseMedia;

/**
 * Document interface.
 */
export type Document = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'Document';
	classType: string;
	flags: number;
	id: bigint;
	accessHash: bigint;
	fileReference: Uint8Array;
	date: number;
	mimeType: string;
	size: bigint;
	thumbs: unknown | null;
	videoThumbs: unknown | null;
	dcId: number;
	attributes: unknown[];
};
