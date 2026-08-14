<script setup lang="ts">
import {ref} from 'vue'
import BaseButton from '@/components/base/BaseButton.vue'
import Expandable from '@/components/base/Expandable.vue'

const showDocs = ref(false)
</script>

<template>
	<BaseButton
		class="has-text-primary"
		@click="showDocs = !showDocs"
	>
		{{ $t('filters.query.help.link') }}
	</BaseButton>

	<Expandable
		:open="showDocs"
		class="content"
	>
		<p>{{ $t('filters.query.help.intro') }}</p>
		<ul>
			<li><strong><code>done</code></strong>: {{ $t('filters.query.help.fields.done') }}</li>
			<li><strong><code>priority</code></strong>: {{ $t('filters.query.help.fields.priority') }}</li>
			<li><strong><code>percentDone</code></strong>: {{ $t('filters.query.help.fields.percentDone') }}</li>
			<li><strong><code>controlFrequency</code></strong>: {{ $t('filters.query.help.fields.controlFrequency') }}</li>
			<li><strong><code>dueDate</code></strong>: {{ $t('filters.query.help.fields.dueDate') }}</li>
			<li><strong><code>startDate</code></strong>: {{ $t('filters.query.help.fields.startDate') }}</li>
			<li><strong><code>endDate</code></strong>: {{ $t('filters.query.help.fields.endDate') }}</li>
			<li><strong><code>doneAt</code></strong>: {{ $t('filters.query.help.fields.doneAt') }}</li>
			<li><strong><code>assignees</code></strong>: {{ $t('filters.query.help.fields.assignees') }}</li>
			<li><strong><code>labels</code></strong>: {{ $t('filters.query.help.fields.labels') }}</li>
			<li><strong><code>project</code></strong>: {{ $t('filters.query.help.fields.project') }}</li>
			<li><strong><code>reminders</code></strong>: {{ $t('filters.query.help.fields.reminders') }}</li>
			<li><strong><code>created</code></strong>: {{ $t('filters.query.help.fields.created') }}</li>
			<li><strong><code>updated</code></strong>: {{ $t('filters.query.help.fields.updated') }}</li>
		</ul>
		<p>{{ $t('filters.query.help.canUseDatemath') }}</p>
		<p>{{ $t('filters.query.help.operators.intro') }}</p>
		<ul>
			<li><strong><code>!=</code></strong>: {{ $t('filters.query.help.operators.notEqual') }}</li>
			<li><strong><code>=</code></strong>: {{ $t('filters.query.help.operators.equal') }}</li>
			<li><strong><code>&gt;</code></strong>: {{ $t('filters.query.help.operators.greaterThan') }}</li>
			<li><strong><code>&gt;=</code></strong>: {{ $t('filters.query.help.operators.greaterThanOrEqual') }}</li>
			<li><strong><code>&lt;</code></strong>: {{ $t('filters.query.help.operators.lessThan') }}</li>
			<li><strong><code>&lt;=</code></strong>: {{ $t('filters.query.help.operators.lessThanOrEqual') }}</li>
			<li><strong><code>like</code></strong>: {{ $t('filters.query.help.operators.like') }}</li>
			<li><strong><code>in</code></strong>: {{ $t('filters.query.help.operators.in') }}</li>
			<li><strong><code>not in</code></strong>: {{ $t('filters.query.help.operators.notIn') }}</li>
		</ul>
		<p>{{ $t('filters.query.help.logicalOperators.intro') }}</p>
		<ul>
			<li><strong><code>&amp;&amp;</code></strong>: {{ $t('filters.query.help.logicalOperators.and') }}</li>
			<li><strong><code>||</code></strong>: {{ $t('filters.query.help.logicalOperators.or') }}</li>
			<li><strong><code>(</code></strong> and <strong><code>)</code></strong>: {{ $t('filters.query.help.logicalOperators.parentheses') }}</li>
		</ul>
		<p>{{ $t('filters.query.help.examples.intro') }}</p>
		<ul>
			<li><strong><code>priority = 4</code></strong>: {{ $t('filters.query.help.examples.priorityEqual') }}</li>
			<li><strong><code>dueDate &lt; now</code></strong>: {{ $t('filters.query.help.examples.dueDatePast') }}</li>
			<li>
				<strong><code>done = false &amp;&amp; priority &gt;= 3</code></strong>:
				{{ $t('filters.query.help.examples.undoneHighPriority') }}
			</li>
			<li><strong><code>assignees in user1, user2</code></strong>: {{ $t('filters.query.help.examples.assigneesIn') }}</li>
			<li>
				<strong><code>(priority = 1 || priority = 2) &amp;&amp; dueDate &lt;= now</code></strong>:
				{{ $t('filters.query.help.examples.priorityOneOrTwoPastDue') }}
			</li>
			<li>
				<strong><code>controlFrequency = "daily"</code></strong>:
				{{ $t('filters.query.help.examples.controlFrequencyDaily') }}
			</li>
			<li>
				<strong><code>controlFrequency in daily, weekly, biweekly</code></strong>:
				{{ $t('filters.query.help.examples.controlFrequencyIn') }}
			</li>
			<li>
				<strong><code>controlFrequency in daily, weekly &amp;&amp; done = false</code></strong>:
				{{ $t('filters.query.help.examples.controlFrequencyBusy') }}
			</li>
		</ul>
	</Expandable>
</template>
