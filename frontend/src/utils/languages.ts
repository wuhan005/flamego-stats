/** The colors of the languages on GitHub (linguist). */
const LANGUAGE_COLORS: Record<string, string> = {
  Go: '#00ADD8',
  JavaScript: '#f1e05a',
  TypeScript: '#3178c6',
  Vue: '#41b883',
  Svelte: '#ff3e00',
  HTML: '#e34c26',
  CSS: '#663399',
  SCSS: '#c6538c',
  Less: '#1d365d',
  Python: '#3572A5',
  PHP: '#4F5D95',
  Shell: '#89e051',
  Rust: '#dea584',
  Java: '#b07219',
  Kotlin: '#A97BFF',
  Swift: '#F05138',
  Dart: '#00B4AB',
  C: '#555555',
  'C++': '#f34b7d',
  'C#': '#178600',
  Ruby: '#701516',
  Lua: '#000080',
  Nix: '#7e7eff',
  Dockerfile: '#384d54',
  Makefile: '#427819',
  Templ: '#66D0DD',
  'Jupyter Notebook': '#DA5B0B',
}

/** Returns the color of the language, the unknown ones get a stable color from the name. */
export function languageColor(language: string): string {
  const color = LANGUAGE_COLORS[language]
  if (color) return color
  let hash = 0
  for (const ch of language) hash = (hash * 31 + ch.charCodeAt(0)) >>> 0
  return `hsl(${hash % 360} 55% 55%)`
}
