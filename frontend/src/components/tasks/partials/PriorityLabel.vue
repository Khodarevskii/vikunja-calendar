<template>
	<span
		v-if="!done && (showAll || priority >= minimumPriority)"
		:class="{
			'negligible': priority <= priorities.LOW,
			'not-so-high': priority > priorities.LOW && priority < priorities.HIGH,
			'high-priority': priority >= priorities.HIGH && priority < priorities.URGENT,
			'important-priority': priority >= priorities.URGENT && priority < priorities.DO_NOW,
			'doNow': priority === priorities.DO_NOW
		}"
		class="priority-label"
	>
		<span class="icon">
			<Icon
				v-if="priority >= priorities.HIGH"
				icon="exclamation-circle"
			/>
			<Icon
				v-else
				icon="exclamation"
			/>
		</span>
		<span>
			<template v-if="priority === priorities.UNSET">{{ $t('task.priority.unset') }}</template>
			<template v-if="priority === priorities.LOW">{{ $t('task.priority.low') }}</template>
			<template v-if="priority === priorities.MEDIUM">{{ $t('task.priority.medium') }}</template>
			<template v-if="priority === priorities.HIGH">{{ $t('task.priority.high') }}</template>
			<template v-if="priority === priorities.URGENT">{{ $t('task.priority.urgent') }}</template>
			<template v-if="priority === priorities.DO_NOW">{{ $t('task.priority.doNow') }}</template>
		</span>
	</span>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {PRIORITIES as priorities} from '@/constants/priorities'
import {useAuthStore} from '@/stores/auth'
	
withDefaults(defineProps<{
	priority: number,
	showAll?: boolean,
	done?: boolean
}>(), {
	showAll: false,
	done: false,
})

const authStore = useAuthStore()

const minimumPriority = computed(() => {
	return authStore.settings.frontendSettings.minimumPriority || priorities.MEDIUM
})
</script>

<style lang="scss" scoped>
.important-priority {
	color:#B91C1C;
	background-color: #FEE2E2;
	inline-size: auto !important;
	border-radius: 4px;
	padding: 0 0.5rem 0 0.25rem; 
}

.high-priority {
	color: #D97706;
	background-color: #FEF3C7;	
	border-radius: 4px;
	padding: 0 0.5rem 0 0.25rem; 
}

.not-so-high  {
	color: #0369A1;
	background-color: #E0F2FE;
	border-radius: 4px;
	padding: 0 0.5rem 0 0.25rem; 
}


.doNow{
	background-color: white;
	color:#B91C1C;
	inline-size: auto !important;
	border-radius: 4px;
	padding: 0 0.5rem 0 0.25rem; 
}

.icon {
	vertical-align: top;
	inline-size: auto !important;
	padding-inline-end: .5rem;
}
</style>
