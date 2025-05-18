import { z } from 'zod';
/**
 * Configuration schema for the Telegram CLI application
 * @property {object} chat - Chat-related settings
 * @property {boolean} chat.sendTypingState - Whether to show "typing..." status when composing messages
 * @property {('default'|'instant'|'never')} chat.readReceiptMode - Controls when to send read receipts:
 *   - default: Only send read state when user interacts (replies, etc.)
 *   - instant: Send read state as soon as message is opened
 *   - never: Never send read state, even when replying
 * @property {object} privacy - Privacy-related settings
 * @property {boolean} privacy.showOnlineStatus - Whether to show your online status to others
 * @property {('everyone'|'contacts'|'nobody')} privacy.lastSeenVisibility - Who can see your last seen status:
 *   - everyone: Visible to all users
 *   - contacts: Only visible to contacts
 *   - nobody: Hidden from everyone
 * @property {object} notifications - Notification settings
 * @property {boolean} notifications.enabled - Whether to show notifications
 * @property {boolean} notifications.showMessagePreview - Whether to show message content in notifications
 */
export const cliGramConfigSchema = z.object({
	chat: z.object({
		sendTypingState: z.boolean({ message: 'Invalid value for sendTypingState' }).optional(),
		readReceiptMode: z
			.enum(['default', 'instant', 'never'], { message: 'Invalid value for readReceiptMode' })
			.optional()
	}),
	privacy: z.object({
		lastSeenVisibility: z
			.enum(['everyone', 'contacts', 'nobody'], { message: 'Invalid value for lastSeenVisibility' })
			.optional()
	}),
	notifications: z.object({
		enabled: z.boolean({ message: 'Invalid value for enabled' }).optional(),
		showMessagePreview: z.boolean({ message: 'Invalid value for showMessagePreview' }).optional()
	})
});

export type CliGramConfigSchema = z.infer<typeof cliGramConfigSchema>;

export const DEFAULT_CONFIG: CliGramConfigSchema = {
	chat: {
		sendTypingState: true,
		readReceiptMode: 'default'
	},
	privacy: {
		lastSeenVisibility: undefined
	},
	notifications: {
		enabled: true,
		showMessagePreview: true
	}
};
