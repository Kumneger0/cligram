import { useTGCliStore } from '@/lib/store';
import { Box, Text } from 'ink';
import React from 'react';

const all = 'All';
const user = 'PeerUser';
const channel = 'PeerChannel';

const keyBindings = {
	general: {
		'ctrl+k': {
			mode: all,
			description: 'Search'
		},
		Tab: {
			mode: all,
			description: 'Switch focus'
		}
	},
	chatArea: {
		r: {
			mode: user,
			description: 'Reply'
		},
		d: {
			mode: user,
			description: 'Delete message'
		},
		f: {
			mode: all,
			description: 'Forward Message'
		},
		e: {
			mode: user,
			description: 'Edit message'
		},
		j: {
			mode: all,
			description: 'Go down'
		},
		k: {
			mode: all,
			description: 'Go up'
		}
	},
	sidebar: {
		j: {
			mode: all,
			description: 'Go down'
		},
		k: {
			mode: all,
			description: 'Go up'
		},
		c: {
			mode: user,
			description: 'Switch to channels'
		},
		u: {
			mode: channel,
			description: 'Switch to users'
		}
	}
} as const;

type ShowKeyBindingProps = {
	type: keyof typeof keyBindings;
};
function ShowKeyBinding({ type }: ShowKeyBindingProps) {
	const currentChatType = useTGCliStore((state) => state.currentChatType);

	const keyBindingToShow = Object.entries(
		type !== 'general' ? { ...keyBindings.general, ...keyBindings[type] } : keyBindings[type]
	).filter(([_key, value]) => value.mode === all || value.mode === currentChatType);
	return (
		<Box>
			<Text>
				{keyBindingToShow.map(([key, value]) => {
					return (
						<Text color="blue" key={key}>
							{key} - {value.description},{' '}
						</Text>
					);
				})}
			</Text>
		</Box>
	);
}

export default ShowKeyBinding;
