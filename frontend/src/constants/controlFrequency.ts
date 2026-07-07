export const CONTROL_FREQUENCIES = {
	DAILY: 'daily',
	WEEKLY: 'weekly',
	BIWEEKLY: 'biweekly',
	ON_COMPLETE: 'onComplete',
	JUST_DO_IT: 'justDoIt',
} as const

export type ControlFrequency = typeof CONTROL_FREQUENCIES[keyof typeof CONTROL_FREQUENCIES]

export const CONTROL_FREQUENCY_DEFAULT: ControlFrequency = CONTROL_FREQUENCIES.ON_COMPLETE

/**
 * Ordered list used by the picker so the labels always appear in the same
 * sequence as in the spec.
 */
export const CONTROL_FREQUENCY_OPTIONS: ControlFrequency[] = [
	CONTROL_FREQUENCIES.DAILY,
	CONTROL_FREQUENCIES.WEEKLY,
	CONTROL_FREQUENCIES.BIWEEKLY,
	CONTROL_FREQUENCIES.ON_COMPLETE,
	CONTROL_FREQUENCIES.JUST_DO_IT,
]
