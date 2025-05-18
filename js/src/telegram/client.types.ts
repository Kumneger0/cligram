type PeerUser =
	| {
			CONSTRUCTOR_ID: number;
			SUBCLASS_OF_ID: number;
			className: 'PeerUser';
			classType: 'constructor';
			userId: bigInt.BigInteger;
	  }
	| {
			CONSTRUCTOR_ID: number;
			SUBCLASS_OF_ID: number;
			className: 'PeerChannel';
			classType: 'constructor';
			channelId: bigInt.BigInteger;
	  }
	| {
			CONSTRUCTOR_ID: number;
			SUBCLASS_OF_ID: number;
			className: 'PeerChat';
			classType: 'constructor';
			chatId: bigInt.BigInteger;
	  };

type PeerNotifySettings = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'PeerNotifySettings';
	classType: 'constructor';
	flags: number;
	showPreviews: null;
	silent: null;
	muteUntil: null;
	iosSound: null;
	androidSound: null;
	otherSound: null;
	storiesMuted: null;
	storiesHideSender: null;
	storiesIosSound: null;
	storiesAndroidSound: null;
	storiesOtherSound: null;
};

type MessageMediaDocument = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'MessageMediaDocument';
	classType: 'constructor';
	flags: number;
	nopremium: boolean;
	spoiler: boolean;
	video: boolean;
	round: boolean;
	voice: boolean;
	// document: Object; // You might want to create a specific interface for this
	altDocuments: null;
	videoCover: null;
	videoTimestamp: null;
	ttlSeconds: null;
};

type Message = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'Message';
	classType: 'constructor';
	out: boolean;
	mentioned: boolean;
	mediaUnread: boolean;
	silent: boolean;
	post: boolean;
	fromScheduled: boolean;
	legacy: boolean;
	editHide: boolean;
	ttlPeriod: null;
	id: number;
	fromId: null;
	peerId: PeerUser;
	fwdFrom: null;
	viaBotId: null;
	replyTo: null;
	date: number;
	message: string;
	media: MessageMediaDocument;
	replyMarkup: null;
	entities: null;
	views: null;
	forwards: null;
	replies: null;
	editDate: null;
	pinned: boolean;
	postAuthor: null;
	groupedId: null;
	restrictionReason: null;
	action: undefined;
	noforwards: boolean;
	reactions: null;
	flags: number;
	invertMedia: boolean;
	flags2: number;
	offline: boolean;
	videoProcessingPending: boolean;
	fromBoostsApplied: null;
	savedPeerId: null;
	viaBusinessBotId: null;
	quickReplyShortcutId: null;
	effect: null;
	factcheck: null;
	reportDeliveryUntilDate: null;
};

type User = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'User';
	classType: 'constructor';
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
	botHasMainApp: boolean;
	id: bigInt.BigInteger;
	accessHash: bigInt.BigInteger;
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
	botActiveUsers: string | null;
	botVerificationIcon: string | null;
};

type InputPeerUser = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'InputPeerUser';
	classType: 'constructor';
	userId: bigInt.BigInteger;
	accessHash: bigInt.BigInteger;
};

type Dialog = {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'Dialog';
	classType: 'constructor';
	flags: number;
	pinned: boolean;
	unreadMark: boolean;
	viewForumAsMessages: boolean;
	peer: PeerUser;
	topMessage: number;
	readInboxMaxId: number;
	readOutboxMaxId: number;
	unreadCount: number;
	unreadMentionsCount: number;
	unreadReactionsCount: number;
	notifySettings: PeerNotifySettings;
	pts: number | null;
	draft: Draft | null;
	folderId: number | null;
	ttlPeriod: number | null;
};

type Draft = {
	linkPreview: boolean;
	date: number;
};

export type DialogInfo = {
	dialog: Dialog;
	pinned: boolean;
	folderId: number | null;
	archived: boolean;
	message: Message;
	date: number;
	entity: User;
	inputEntity: InputPeerUser;
	id: bigInt.BigInteger;
	title: string;
	name: string;
	unreadCount: number;
	unreadMentionsCount: number;
	draft: Draft;
	isUser: boolean;
	isGroup: boolean;
	isChannel: boolean;
};
