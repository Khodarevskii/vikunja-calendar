<template>
	<span
		v-if="show"
		v-tooltip="tooltip"
		class="control-freq-label"
		:class="`freq-${frequency}`"
	>
		<span class="icon">
			<Icon icon="tachometer-alt" />
		</span>
		<span class="label">{{ label }}</span>
	</span>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {
	CONTROL_FREQUENCIES,
	type ControlFrequency,
} from '@/constants/controlFrequency'

const props = withDefaults(defineProps<{
	frequency: ControlFrequency | string,
	/** Optionally hide the label for the default 'onComplete' value. */
	hideDefault?: boolean,
	done?: boolean,
}>(), {
	hideDefault: false,
	done: false,
})

const {t} = useI18n({useScope: 'global'})

const show = computed(() => {
	if (props.done) return false
	if (!props.frequency) return false
	if (props.hideDefault && props.frequency === CONTROL_FREQUENCIES.ON_COMPLETE) return false
	return true
})

const label = computed(() => t(`task.control.values.${props.frequency}`))
const tooltip = computed(() => `${t('task.control.title')}: ${label.value}`)
</script>

<style lang="scss" scoped>
.control-freq-label {
	display: inline-flex;
	align-items: center;
	gap: 0.35rem;
	padding: 0.15rem 0.55rem;
	border-radius: 999px;
	font-size: 0.8rem;
	font-weight: 600;
	line-height: 1.2;
	border: 1px solid transparent;
	white-space: nowrap;
}

.icon {
	display: inline-flex;
	inline-size: auto !important; // override kanban card constraints
}

// Palette. Uses Vikunja's semantic CSS variables so both light and dark
// themes are handled automatically — `--danger` / `--warning` / `--info` /
// `--success` are already tuned per theme.

.freq-daily {
	color: var(--white);
	background: var(--danger);
	border-color: var(--danger);
}

.freq-weekly {
	color: var(--scheme-invert);
	background: var(--warning);
	border-color: var(--warning);
}

.freq-biweekly {
	color: var(--white);
	background: var(--info);
	border-color: var(--info);
}

.freq-onComplete {
	color: var(--grey-700);
	background: var(--grey-100);
	border-color: var(--grey-300);
}

.freq-justDoIt {
	color: var(--white);
	background: var(--success);
	border-color: var(--success);
}

// Dark theme tweaks — CSS variables already flip most colours, but soft
// backgrounds need explicit remapping to avoid the "neon block" look.
:root[data-theme="dark"] {
	.freq-onComplete {
		color: var(--grey-200);
		background: var(--grey-800);
		border-color: var(--grey-700);
	}

	.freq-weekly {
		// Warning is bright orange in dark mode; darken the text for contrast.
		color: #2a1e00;
	}
}

// Automatic dark preference fallback (when user hasn't picked explicitly).
@media (prefers-color-scheme: dark) {
	:root:not([data-theme="light"]) {
		.freq-onComplete {
			color: var(--grey-200);
			background: var(--grey-800);
			border-color: var(--grey-700);
		}
		.freq-weekly {
			color: #2a1e00;
		}
	}
}
</style>
