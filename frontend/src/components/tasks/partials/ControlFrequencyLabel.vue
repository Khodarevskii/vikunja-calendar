<template>
	<span
		v-if="show"
		v-tooltip="tooltip"
		class="control-freq-label"
		:class="`freq-${frequency}`"
	>
		<span class="freq-icon">
			<Icon icon="tachometer-alt" />
		</span>
		<span class="freq-text">{{ label }}</span>
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
// Mirror PriorityLabel: coloured icon + coloured text, no pill background.
.control-freq-label {
	display: inline-flex;
	align-items: center;
	white-space: nowrap;
	inline-size: auto !important; // override kanban card width constraints
}

.freq-icon {
	inline-size: auto !important;
	padding-inline-end: 0.5rem;
	display: inline-flex;
}

.freq-text {
	color: inherit;
}

// Palette — the whole element (icon + text) picks the same colour.
.freq-daily     { color: var(--danger); }
.freq-weekly    { color: var(--warning); }
.freq-biweekly  { color: var(--info); }
.freq-justDoIt  { color: var(--success); }

// Light theme default: dark grey for the neutral value.
.freq-onComplete { color: var(--grey-700); }

// Dark theme: keep the palette variables (they already flip) but soften the
// "onComplete" grey so it stays readable on the dark card background.
html.dark .freq-onComplete {
	color: var(--grey-200);
}
</style>
