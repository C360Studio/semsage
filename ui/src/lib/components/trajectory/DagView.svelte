<script lang="ts">
	/**
	 * DagView - CSS-only node-and-edge visualization for decompose_task DAGs.
	 *
	 * Computes topological levels so nodes are arranged in columns by dependency
	 * depth. Edges are rendered as SVG lines overlaid on the grid.
	 */

	import type { Dag, DagNode } from '$lib/types';

	interface Props {
		dag: Dag;
	}

	let { dag }: Props = $props();

	// Compute topological levels (column per dependency depth)
	const levels = $derived.by(() => {
		const levelMap: Record<string, number> = {};
		const nodes = dag.nodes;

		function getLevel(id: string, visited = new Set<string>()): number {
			if (levelMap[id] !== undefined) return levelMap[id];
			if (visited.has(id)) return 0; // cycle guard

			visited.add(id);
			const node = nodes.find((n) => n.id === id);
			if (!node || node.depends_on.length === 0) {
				levelMap[id] = 0;
				return 0;
			}

			const maxParentLevel = Math.max(...node.depends_on.map((dep) => getLevel(dep, new Set(visited))));
			levelMap[id] = maxParentLevel + 1;
			return levelMap[id];
		}

		nodes.forEach((n) => getLevel(n.id));
		return levelMap;
	});

	// Group nodes by level for layout
	const columns = $derived.by(() => {
		const cols: DagNode[][] = [];
		dag.nodes.forEach((n) => {
			const col = levels[n.id] ?? 0;
			if (!cols[col]) cols[col] = [];
			cols[col].push(n);
		});
		return cols;
	});

	// Node position map for edge routing (col * NODE_W, row * NODE_H)
	const NODE_W = 180;
	const NODE_H = 80;
	const COL_GAP = 60;
	const ROW_GAP = 16;

	const positions = $derived.by(() => {
		const pos: Record<string, { x: number; y: number }> = {};
		columns.forEach((col, ci) => {
			col.forEach((node, ri) => {
				pos[node.id] = {
					x: ci * (NODE_W + COL_GAP),
					y: ri * (NODE_H + ROW_GAP)
				};
			});
		});
		return pos;
	});

	const svgWidth = $derived(columns.length * (NODE_W + COL_GAP) - COL_GAP);
	const svgHeight = $derived(
		Math.max(...columns.map((col) => col.length)) * (NODE_H + ROW_GAP) - ROW_GAP
	);

	// Collect edges
	const edges = $derived(
		dag.nodes.flatMap((node) =>
			node.depends_on
				.filter((dep) => positions[dep] && positions[node.id])
				.map((dep) => ({ from: dep, to: node.id }))
		)
	);

	function truncate(text: string, max = 60): string {
		return text.length > max ? text.slice(0, max) + '…' : text;
	}
</script>

<div class="dag-view" aria-label="Task dependency graph">
	<div class="dag-scroll">
		<svg
			class="dag-svg"
			width={svgWidth}
			height={svgHeight}
			viewBox="0 0 {svgWidth} {svgHeight}"
			aria-hidden="true"
		>
			<defs>
				<marker
					id="arrowhead"
					markerWidth="8"
					markerHeight="6"
					refX="8"
					refY="3"
					orient="auto"
				>
					<polygon points="0 0, 8 3, 0 6" fill="var(--color-border)" />
				</marker>
			</defs>

			{#each edges as edge (edge.from + '-' + edge.to)}
				{@const fromPos = positions[edge.from]}
				{@const toPos = positions[edge.to]}
				{@const x1 = fromPos.x + NODE_W}
				{@const y1 = fromPos.y + NODE_H / 2}
				{@const x2 = toPos.x}
				{@const y2 = toPos.y + NODE_H / 2}
				{@const cx1 = x1 + (x2 - x1) * 0.5}
				{@const cy2 = y2}
				<path
					d="M{x1},{y1} C{cx1},{y1} {cx1},{cy2} {x2},{y2}"
					fill="none"
					stroke="var(--color-border)"
					stroke-width="1.5"
					marker-end="url(#arrowhead)"
				/>
			{/each}
		</svg>

		<div class="dag-nodes" style="width: {svgWidth}px; height: {svgHeight}px;">
			{#each dag.nodes as node (node.id)}
				{@const pos = positions[node.id]}
				<div
					class="dag-node"
					style="left: {pos.x}px; top: {pos.y}px; width: {NODE_W}px; height: {NODE_H}px;"
				>
					<div class="node-header">
						<span class="node-role">{node.role}</span>
						<span class="node-id">{node.id}</span>
					</div>
					<p class="node-prompt" title={node.prompt}>{truncate(node.prompt)}</p>
				</div>
			{/each}
		</div>
	</div>
</div>

<style>
	.dag-view {
		width: 100%;
	}

	.dag-scroll {
		overflow-x: auto;
		position: relative;
	}

	.dag-svg {
		position: absolute;
		top: 0;
		left: 0;
		pointer-events: none;
	}

	.dag-nodes {
		position: relative;
	}

	.dag-node {
		position: absolute;
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		padding: var(--space-2) var(--space-3);
		background: var(--color-bg-primary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		font-size: var(--font-size-xs);
		overflow: hidden;
	}

	.node-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-1);
	}

	.node-role {
		font-weight: var(--font-weight-semibold);
		color: var(--color-accent);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.node-id {
		font-family: var(--font-family-mono);
		font-size: 10px;
		color: var(--color-text-muted);
		white-space: nowrap;
		flex-shrink: 0;
	}

	.node-prompt {
		color: var(--color-text-secondary);
		line-height: var(--line-height-normal);
		margin: 0;
		overflow: hidden;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
	}
</style>
