/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  darkMode: 'media', // Uses prefers-color-scheme for automatic dark mode
  theme: {
    extend: {
      colors: {
        // macOS system colors
        'macos-blue': '#007AFF',
        'macos-green': '#34C759',
        'macos-orange': '#FF9500',
        'macos-red': '#FF3B30',
        'macos-gray': '#8E8E93',
        // Sidebar colors
        'sidebar-light': 'rgba(255, 255, 255, 0.3)',
        'sidebar-dark': 'rgba(17, 24, 39, 0.3)',
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
