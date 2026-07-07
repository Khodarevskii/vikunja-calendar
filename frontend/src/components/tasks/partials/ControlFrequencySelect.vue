<template>
	<div class="select">
		<select
			v-model="value"
			:disabled="disabled || undefined"
		>
			<option
				v-for="opt in CONTROL_FREQUENCY_OPTIONS"
				:key="opt"
				:value="opt"
			>
				{{ $t(`task.control.values.${opt}`) }}
			</option>
		</select>
	</div>
</template>

<script setup lang="ts">
import {
	CONTROL_FREQUENCIES,
	CONTROL_FREQUENCY_OPTIONS,
	type ControlFrequency,
} from '@/constants/controlFrequency'

withDefaults(defineProps<{
	disabled?: boolean,
}>(), {
	disabled: false,
})

const value = defineModel<ControlFrequency>({
	required: true,
	default: CONTROL_FREQUENCIES.ON_COMPLETE,
})
</script>
