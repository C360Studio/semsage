<script lang="ts">
	import * as icons from 'lucide-svelte';

	interface Props {
		name: string;
		size?: number;
		strokeWidth?: number;
		class?: string;
	}

	let { name, size = 24, strokeWidth = 2, class: className = '' }: Props = $props();

	const iconMap: Record<string, typeof icons.Activity> = {
		activity: icons.Activity,
		'chevron-down': icons.ChevronDown,
		'chevron-up': icons.ChevronUp,
		'chevron-left': icons.ChevronLeft,
		'chevron-right': icons.ChevronRight,
		check: icons.Check,
		'check-circle': icons.CheckCircle,
		x: icons.X,
		send: icons.Send,
		loader: icons.Loader2,
		'message-square': icons.MessageSquare,
		'layout-dashboard': icons.LayoutDashboard,
		settings: icons.Settings,
		circle: icons.Circle,
		play: icons.Play,
		pause: icons.Pause,
		'square': icons.Square,
		clock: icons.Clock,
		slash: icons.Slash,
		user: icons.User,
		bot: icons.Bot,
		brain: icons.Brain,
		cpu: icons.Cpu,
		database: icons.Database,
		wrench: icons.Wrench,
		'alert-circle': icons.AlertCircle,
		'alert-triangle': icons.AlertTriangle,
		'help-circle': icons.HelpCircle,
		tag: icons.Tag,
		'external-link': icons.ExternalLink,
		'refresh-cw': icons.RefreshCw,
		plus: icons.Plus,
		info: icons.Info,
		file: icons.File,
		'file-text': icons.FileText,
		'book-open': icons.BookOpen,
		code: icons.Code,
		folder: icons.Folder,
		layers: icons.Layers,
		search: icons.Search,
		list: icons.List,
		menu: icons.Menu,
		network: icons.Network,
		'git-branch': icons.GitBranch,
		'arrow-right': icons.ArrowRight,
		'arrow-up-right': icons.ArrowUpRight,
		tree: icons.Network,
		eye: icons.Eye,
		'chevron-down-circle': icons.ChevronDown,
		zap: icons.Zap,
		'box': icons.Box,
		'arrow-left': icons.ArrowLeft,
		'copy': icons.Copy,
		'lightbulb': icons.Lightbulb,
		'file-code': icons.FileCode,
		'check-square': icons.CheckSquare,
		'link': icons.Link,
		'edit-3': icons.Edit3
	};

	const IconComponent = $derived(iconMap[name] || icons.Circle);
</script>

{#if IconComponent}
	<IconComponent {size} {strokeWidth} class={className} />
{/if}
