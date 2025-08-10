<script>
	import '../app.css';
</script>

<div class="min-h-screen bg-gray-900 text-white">
	<nav class="bg-gray-800 border-b border-red-700">
		<div class="container mx-auto px-4">
			<div class="flex items-center justify-between h-16">
				<a href="/" class="flex items-center space-x-2">
					<span class="text-xl font-bold text-red-700">DBD Analytics</span>
				</a>
				<div class="flex space-x-6">
					<a href="/" class="hover:text-red-700 transition-colors">Home</a>
					<a href="/player" class="hover:text-red-700 transition-colors">Player Stats</a>
				</div>
			</div>
		</div>
	</nav>

	<main class="flex-1">
		<slot />
	</main>

	<footer class="bg-gray-800 border-t border-gray-600 mt-12">
		<div class="container mx-auto px-4 py-6 text-center text-gray-400">
			<p>&copy; 2025 DBD Analytics. Built with SvelteKit.</p>
		</div>
	</footer>
</div>
