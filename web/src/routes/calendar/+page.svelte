<script lang="ts">
	const now = new Date();
	const year = now.getFullYear();
	const month = now.getMonth();

	const monthName = now.toLocaleString('default', { month: 'long', year: 'numeric' });
	const daysInMonth = new Date(year, month + 1, 0).getDate();
	const firstDay = new Date(year, month, 1).getDay();

	const weeks: (number | null)[][] = [];
	let currentWeek: (number | null)[] = [];

	for (let i = 0; i < firstDay; i++) {
		currentWeek.push(null);
	}

	for (let day = 1; day <= daysInMonth; day++) {
		currentWeek.push(day);
		if (currentWeek.length === 7) {
			weeks.push(currentWeek);
			currentWeek = [];
		}
	}

	if (currentWeek.length > 0) {
		while (currentWeek.length < 7) {
			currentWeek.push(null);
		}
		weeks.push(currentWeek);
	}

	const today = now.getDate();
</script>

<svelte:head>
	<title>Calendar - Yunt Mail</title>
</svelte:head>

<div class="options-area">
	<h2>Calendar</h2>
	<p style="font-size:11px;color:var(--text-muted);margin-bottom:12px;">{monthName}</p>

	<table class="calendar-table">
		<thead>
			<tr>
				<th>Sun</th>
				<th>Mon</th>
				<th>Tue</th>
				<th>Wed</th>
				<th>Thu</th>
				<th>Fri</th>
				<th>Sat</th>
			</tr>
		</thead>
		<tbody>
			{#each weeks as week}
				<tr>
					{#each week as day}
						<td class:today={day === today} class:other-month={day === null}>
							{day ?? ''}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>

	<p style="margin-top:12px;font-size:11px;color:var(--text-muted);">
		Calendar feature coming soon. Events and scheduling will be available in a future update.
	</p>
</div>
