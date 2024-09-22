interface PeerNotifySettings {
  flags: number;
  showPreviews: boolean | null;
  silent: boolean | null;
  muteUntil: number | null;
  iosSound: string | null;
  androidSound: string | null;
  otherSound: string | null;
  storiesMuted: boolean;
  storiesHideSender: boolean | null;
  storiesIosSound: string | null;
  storiesAndroidSound: string | null;
  storiesOtherSound: string | null;
}

interface PeerUser {
  userId: bigInt.BigInteger;
  className: "PeerUser" | "PeerChannel" | "PeerChat";
}

export interface Dialog {
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
  draft: any; // Adjust type based on actual structure
  folderId: number | null;
  ttlPeriod: number | null;
  className: "Dialog";
}

export interface MessagesResponse {
  count: number;
  dialogs: Dialog[];
  messages: any[];
  users: any[];
  className: "messages.DialogsSlice";
}

export interface User {
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
  status: string | null;
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
  fromId: { userId: string; className: string } | null;
  fromBoostsApplied: any; // Adjust type as needed
  peerId: { userId: string; className: string };
  savedPeerId: any; // Adjust type as needed
  fwdFrom: any; // Adjust type as needed
  viaBotId: any; // Adjust type as needed
  viaBusinessBotId: any; // Adjust type as needed
  replyTo: any; // Adjust type as needed
  date: number;
  message: string;
  media: any; // Adjust type as needed
  replyMarkup: any; // Adjust type as needed
  entities: any; // Adjust type as needed
  views: any; // Adjust type as needed
  forwards: any; // Adjust type as needed
  replies: any; // Adjust type as needed
  editDate: number | null;
  postAuthor: any; // Adjust type as needed
  groupedId: any; // Adjust type as needed
  reactions: any; // Adjust type as needed
  restrictionReason: any; // Adjust type as needed
  ttlPeriod: any; // Adjust type as needed
  quickReplyShortcutId: any; // Adjust type as needed
  effect: any; // Adjust type as needed
  factcheck: any; // Adjust type as needed
  className: string;
}

interface ChatUser {
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
  username: string | null;
  phone: string;
  photo: {
    flags: number;
    hasVideo: boolean;
    personal: boolean;
    photoId: string;
    strippedThumb: {
      type: string;
      data: number[];
    };
    dcId: number;
    className: string;
  };
  status: {
    wasOnline: number;
    className: string;
  };
  botInfoVersion: any; // Adjust type as needed
  restrictionReason: any; // Adjust type as needed
  botInlinePlaceholder: any; // Adjust type as needed
  langCode: any; // Adjust type as needed
  emojiStatus: any; // Adjust type as needed
  usernames: any; // Adjust type as needed
  storiesMaxId: any; // Adjust type as needed
  color: any; // Adjust type as needed
  profileColor: any; // Adjust type as needed
  className: string;
}

export interface MessagesSlice {
  flags: number;
  inexact: boolean;
  count: number;
  nextRate: any; // Adjust type as needed
  offsetIdOffset: any; // Adjust type as needed
  messages: Message[];
  chats: any[]; // Adjust type as needed
  users: ChatUser[];
  className: string;
}