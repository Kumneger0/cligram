import { TelegramClient } from 'telegram';
import { Dialog } from './lib/types';
import { ChannelInfo } from './telegram/client';

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
	webPage?: {
		url: string;
		displayUrl: string | null;
	} | null;
	document?: {
		document: string;
	} | null;
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

type SeachMode = 'CONVERSATION' | 'CHANNELS_OR_ USERS' | null;

export type TGCliStore = {
	client: TelegramClient | null;
	updateClient: (client: TelegramClient) => void;
	searchMode: SeachMode;
	setSearchMode: (searchMode: SeachMode) => void;
	selectedUser: ChatUser | ChannelInfo | null;
	setSelectedUser: (selectedUser: ChatUser | ChannelInfo | null) => void;
	messageAction: MessageAction | null;
	setMessageAction: (messageAction: MessageAction | null) => void;
	currentChatType: Dialog['peer']['className'];
	setCurrentChatType: (currentChatType: Dialog['peer']['className']) => void;
};

interface Integer {
	value: bigint;
}

interface ChatPhoto {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'ChatPhoto';
	classType: 'constructor';
	flags: number;
	hasVideo: boolean;
	photoId: Integer;
	strippedThumb: Buffer;
	dcId: number;
}

interface ChatBannedRights {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'ChatBannedRights';
	classType: 'constructor';
	flags: number;
	viewMessages: boolean;
	sendMessages: boolean;
	sendMedia: boolean;
	sendStickers: boolean;
	sendGifs: boolean;
	sendGames: boolean;
	sendInline: boolean;
	embedLinks: boolean;
	sendPolls: boolean;
	changeInfo: boolean;
	inviteUsers: boolean;
	pinMessages: boolean;
	manageTopics: boolean;
	sendPhotos: boolean;
	sendVideos: boolean;
	sendRoundvideos: boolean;
	sendAudios: boolean;
	sendVoices: boolean;
	sendDocs: boolean;
	sendPlain: boolean;
	untilDate: number;
}

export interface Channel {
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

	// Optional properties
	restrictionReason: null | any;
	adminRights: null | any;
	bannedRights: null | any;
	defaultBannedRights: ChatBannedRights;
	participantsCount: null | number;
	usernames: null | any;
	storiesMaxId: null | number;
	color: null | any;
	profileColor: null | any;
	emojiStatus: null | any;
	level: null | number;
	subscriptionUntilDate: null | number;
	botVerificationIcon: null | any;
}
