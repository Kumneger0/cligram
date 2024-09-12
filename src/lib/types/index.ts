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
  userId: string;
  className: "PeerUser";
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
